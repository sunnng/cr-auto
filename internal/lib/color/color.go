// Package color 对应 Lua 工程的 lib/color.lua：帧上的比色 / 找色 / 等待页面门面。
// 每帧识别都由 FrameSource 提供 *image.NRGBA（ADR-0003：识别不调用 AutoGo 实时屏幕 API，
// 纯 Go 算法在同一帧上完成），设备端由 main 注入“截图隐身 + CaptureScreen”，桌面测试注入固定帧。
package color

import (
	"image"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/touch"
	"app/internal/vision"
)

const tag = "[Color]"

// 轮询缺省参数（对应 Lua lib/color.lua 的 `or 10000` / `or 500` / `or 800`）。
const (
	defaultTimeoutMs  = 10000
	defaultIntervalMs = 500
	defaultTapDelayMs = 800
)

// FrameSource 帧来源：每次识别获取一帧。
type FrameSource interface {
	Capture() (*image.NRGBA, error)
}

var (
	frameSource FrameSource
	guardHook   func()
	nowFn       = time.Now
	sleepFn     func(ms int)
)

// SetFrameSource 注入帧来源；nil 时识别一律不命中。
func SetFrameSource(src FrameSource) { frameSource = src }

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

func capture() *image.NRGBA {
	if frameSource == nil {
		return nil
	}
	frame, err := frameSource.Capture()
	if err != nil {
		logger.Warn(tag, "截图失败: %v", err)
		return nil
	}
	return frame
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

// Match 单特征比色是否匹配（当前帧）。
func Match(feature vision.Feature) bool {
	frame := capture()
	if frame == nil {
		return false
	}
	return vision.Match(frame, feature)
}

// MatchAny 多个特征任一匹配；返回命中的下标，未命中返回 -1。
func MatchAny(features []vision.Feature) (bool, int) {
	frame := capture()
	if frame == nil {
		return false, -1
	}
	return vision.MatchAny(frame, features)
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
	SleepMs    int // 命中后的额外等待
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

// Find 在区域内找色，返回首个命中点坐标。
func Find(def vision.FindDef) (x, y int, ok bool) {
	frame := capture()
	if frame == nil {
		return 0, 0, false
	}
	return vision.FindMultiColor(frame, def)
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
	frame := capture()
	if frame == nil {
		return nil
	}
	return vision.FindMultiColorAll(frame, def)
}
