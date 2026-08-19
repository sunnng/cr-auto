// Package color 对应 Lua 工程的 lib/color.lua：比色 / 找色 / 等待页面门面。
// 识别走注入的 Screen（设备端为 AutoGo images 图色 API + 截图隐身；桌面测试注入脚本桩）。
package color

import (
	"image"
	"time"

	"app/internal/lib/touch"
	"app/internal/vision"
)

const (
	defaultTimeoutMs  = 10000
	defaultIntervalMs = 500
	defaultTapDelayMs = 800
)

// Screen 图色接缝，形状对齐 AutoGo images 比色/找色 API（不含 displayId）。
type Screen interface {
	Begin()
	End()
	DetectsMultiColors(colors string, sim float32) bool
	CmpColor(x, y int, colorStr string, sim float32) bool
	FindMultiColors(x1, y1, x2, y2 int, colors string, sim float32, dir int) (x, y int)
	FindMultiColorsAll(x1, y1, x2, y2 int, colors string, sim float32, dir int) []image.Point
}

var (
	screen    Screen
	guardHook func()
	nowFn     = time.Now
	sleepFn   func(ms int)
)

// SetScreen 注入图色实现；nil 时识别一律不命中。
func SetScreen(s Screen) { screen = s }

// Ready 是否已注入图色实现。
func Ready() bool { return screen != nil }

// Session 把 fn 包在一次 Begin/End 内（可嵌套；设备端引用计数截图隐身）。
func Session(fn func()) { session(fn) }

func session(fn func()) {
	if screen == nil {
		fn()
		return
	}
	screen.Begin()
	defer screen.End()
	fn()
}

func simOrOne(sim float32) float32 {
	if sim <= 0 {
		return 1
	}
	return sim
}

// SetGuardHook 注册主线程守卫回调（由 Runtime 注入 Guard.Check）。
func SetGuardHook(fn func()) { guardHook = fn }

// SetNow 注入时钟（测试用）。
func SetNow(fn func() time.Time) { nowFn = fn }

// SetSleep 注入休眠（测试用；未注入时以真实时间等待）。
func SetSleep(fn func(ms int)) { sleepFn = fn }

// TickGuard 执行一次守卫扫描（wait 轮询内调用）。
func TickGuard() {
	if guardHook != nil {
		guardHook()
	}
}

func sleep(ms int) {
	if sleepFn != nil {
		sleepFn(ms)
		return
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func waitInterval(intervalMs int) {
	TickGuard()
	sleep(intervalMs)
}

func sleepFragment(ms, stepMs int) {
	left := ms
	for left > 0 {
		TickGuard()
		chunk := min(left, stepMs)
		sleep(chunk)
		left -= chunk
	}
}

// SleepMs 普通休眠（对应 Lua 全局 sleep()；不触发守卫扫描）。
func SleepMs(ms int) { sleep(ms) }

// Sleep 分片休眠：每片间隙执行守卫扫描（对应 Lua Guard.sleep，供业务模块等待时清弹窗）。
func Sleep(ms, stepMs int) {
	if stepMs <= 0 {
		stepMs = 500
	}
	sleepFragment(ms, stepMs)
}

func detects(feature vision.Feature) bool {
	if screen == nil {
		return false
	}
	colors := vision.DetectsColors(feature)
	if colors == "" {
		return false
	}
	return screen.DetectsMultiColors(colors, simOrOne(feature.Sim))
}

// Match 单特征多点比色（AutoGo DetectsMultiColors）。
func Match(feature vision.Feature) bool {
	var ok bool
	session(func() { ok = detects(feature) })
	return ok
}

// MatchAny 多个特征任一匹配；返回命中的下标，未命中返回 -1。
func MatchAny(features []vision.Feature) (bool, int) {
	ok, which := false, -1
	session(func() {
		for i, f := range features {
			if detects(f) {
				ok, which = true, i
				return
			}
		}
	})
	return ok, which
}

// MatchPoints 逐点 CmpColor（识别诊断锚点与场景置信度）。特征串非法时返回 (nil, false)。
func MatchPoints(feature vision.Feature) ([]vision.PointResult, bool) {
	if screen == nil {
		return nil, false
	}
	points, err := vision.ParsePoints(feature.Points)
	if err != nil {
		return nil, false
	}
	sim := simOrOne(feature.Sim)
	results := make([]vision.PointResult, 0, len(points))
	matched := 0
	session(func() {
		for _, p := range points {
			hit := screen.CmpColor(p.X, p.Y, vision.CmpSpec(p), sim)
			if hit {
				matched++
			}
			results = append(results, vision.PointResult{Point: p, Matched: hit})
		}
	})
	return results, matched == len(points)
}

// Any 通用匹配（Guard 等用）：接受单特征、多特征表或自定义判定函数。
// 返回 (是否命中, 命中下标)，函数与单特征命中时下标为 -1。
func Any(target any) (bool, int) {
	switch t := target.(type) {
	case func() bool:
		return t(), -1
	case vision.Feature:
		return Match(t), -1
	case []vision.Feature:
		return MatchAny(t)
	default:
		return false, -1
	}
}

// Wait 轮询直到 target 命中。
func Wait(target any, timeoutMs, intervalMs int) (bool, int) {
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}
	if intervalMs <= 0 {
		intervalMs = defaultIntervalMs
	}
	if ok, which := Any(target); ok {
		return true, which
	}
	deadline := nowFn().Add(time.Duration(timeoutMs) * time.Millisecond)
	for nowFn().Before(deadline) {
		if ok, which := Any(target); ok {
			return true, which
		}
		waitInterval(intervalMs)
	}
	return false, -1
}

// WaitMatch 轮询直到单特征匹配；命中后可选额外等待。
func WaitMatch(feature vision.Feature, timeoutMs, intervalMs, sleepMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}
	if intervalMs <= 0 {
		intervalMs = defaultIntervalMs
	}
	if Match(feature) {
		if sleepMs > 0 {
			sleepFragment(sleepMs, intervalMs)
		}
		return true
	}
	deadline := nowFn().Add(time.Duration(timeoutMs) * time.Millisecond)
	for nowFn().Before(deadline) {
		if Match(feature) {
			if sleepMs > 0 {
				sleepFragment(sleepMs, intervalMs)
			}
			return true
		}
		waitInterval(intervalMs)
	}
	return false
}

// WaitGone 轮询直到特征不再匹配。
func WaitGone(target any, timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}
	if intervalMs <= 0 {
		intervalMs = defaultIntervalMs
	}
	if ok, _ := Any(target); !ok {
		return true
	}
	deadline := nowFn().Add(time.Duration(timeoutMs) * time.Millisecond)
	for nowFn().Before(deadline) {
		if ok, _ := Any(target); !ok {
			return true
		}
		waitInterval(intervalMs)
	}
	return false
}

// TapOpts 持续点击直到目标特征出现的参数。
type TapOpts struct {
	TimeoutMs  int
	IntervalMs int
	TapDelayMs int
	MaxTaps    int
	SleepMs    int
}

// TapUntilMatch 持续点击 tapTarget 直到 feature 命中。
func TapUntilMatch(tapTarget any, feature any, opts TapOpts) (bool, int) {
	if tapTarget == nil {
		return false, -1
	}
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 15000
	}
	if opts.IntervalMs <= 0 {
		opts.IntervalMs = defaultIntervalMs
	}
	if opts.TapDelayMs <= 0 {
		opts.TapDelayMs = defaultTapDelayMs
	}

	tapCount := 0
	deadline := nowFn().Add(time.Duration(opts.TimeoutMs) * time.Millisecond)
	for nowFn().Before(deadline) {
		if ok, which := Any(feature); ok {
			if opts.SleepMs > 0 {
				sleepFragment(opts.SleepMs, opts.IntervalMs)
			}
			return true, which
		}
		if opts.MaxTaps > 0 && tapCount >= opts.MaxTaps {
			break
		}
		TickGuard()
		if !performTap(tapTarget, opts.TapDelayMs) {
			return false, -1
		}
		tapCount++
		waitInterval(opts.IntervalMs)
	}
	return false, -1
}

func performTap(target any, tapDelayMs int) bool {
	switch t := target.(type) {
	case func():
		t()
		return true
	case image.Point:
		touch.TapR(t.X, t.Y, tapDelayMs)
		return true
	case image.Rectangle:
		if t.Empty() {
			return false
		}
		touch.TapArea(t, tapDelayMs)
		return true
	default:
		return false
	}
}

func findOnce(def vision.FindDef) (int, int, bool) {
	if screen == nil {
		return 0, 0, false
	}
	r := def.Region
	if r.Empty() {
		return 0, 0, false
	}
	colors := vision.FindColors(def)
	if colors == "" {
		return 0, 0, false
	}
	x, y := screen.FindMultiColors(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, colors, simOrOne(def.Sim), def.Dir)
	if x < 0 || y < 0 {
		return 0, 0, false
	}
	return x, y, true
}

// Find 在区域内找色，返回首个命中点坐标。
func Find(def vision.FindDef) (x, y int, ok bool) {
	session(func() { x, y, ok = findOnce(def) })
	return x, y, ok
}

// TapFind 找色并点击。
func TapFind(def vision.FindDef, delayMs int) bool {
	x, y, ok := Find(def)
	if !ok {
		return false
	}
	touch.TapR(x, y, delayMs)
	return true
}

// FindPoint 找色返回首个坐标点。
func FindPoint(def vision.FindDef) (image.Point, bool) {
	x, y, ok := Find(def)
	if !ok {
		return image.Point{}, false
	}
	return image.Point{X: x, Y: y}, true
}

// FindAll 区域内找色返回全部坐标。
func FindAll(def vision.FindDef) []image.Point {
	var points []image.Point
	session(func() {
		if screen == nil {
			return
		}
		r := def.Region
		if r.Empty() {
			return
		}
		colors := vision.FindColors(def)
		if colors == "" {
			return
		}
		raw := screen.FindMultiColorsAll(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, colors, simOrOne(def.Sim), def.Dir)
		for _, p := range raw {
			if p.X < 0 || p.Y < 0 {
				continue
			}
			points = append(points, p)
		}
	})
	return points
}

// MatchRGB 单点比色（AutoGo CmpColor）。spec 与 sim 原样交给 Screen。
func MatchRGB(x, y int, spec string, sim float32) bool {
	if screen == nil || spec == "" {
		return false
	}
	var ok bool
	session(func() { ok = screen.CmpColor(x, y, spec, simOrOne(sim)) })
	return ok
}
