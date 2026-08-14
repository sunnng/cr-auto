// Package touch 对应 Lua 工程的 lib/touch.lua：点击、按键、增强滑动。
// 触控执行体通过 SetPerform 注入：设备端由 main 映射到 AutoGo motion，桌面测试注入假实现。
// 设备固定 1600×900 @ 320dpi，坐标原样使用。
package touch

import (
	"image"
	"math"
)

// Perform 触控执行体。
type Perform struct {
	Tap       func(x, y int)
	TouchDown func(id, x, y int)
	TouchMove func(id, x, y, durationMs int)
	TouchUp   func(id, x, y int) bool
	Back      func()
	Sleep     func(ms int)
	Random    func(min, max int) int
}

var perform Perform

// SetPerform 注入触控执行体；未注入时所有触控为 no-op（桩）。
func SetPerform(p Perform) { perform = p }

const defaultFingerID = 1

func jitter(v int) int {
	if perform.Random == nil {
		return v
	}
	return v + perform.Random(-3, 3)
}

func sleep(ms int) {
	if perform.Sleep != nil {
		perform.Sleep(ms)
	}
}

// TapR 点击坐标（抖动后 tap），delayMs 后休眠；未给 delay 时随机 300-600ms。
func TapR(x, y int, delayMs int) {
	if perform.Tap != nil {
		perform.Tap(jitter(x), jitter(y))
	}
	if delayMs > 0 {
		sleep(delayMs)
	} else if perform.Random != nil {
		sleep(perform.Random(300, 600))
	}
}

// TapXy TapR 的别名。
func TapXy(x, y, delayMs int) { TapR(x, y, delayMs) }

// TapArea 在矩形区域内随机点击。
func TapArea(rect image.Rectangle, delayMs int) {
	x := rect.Min.X
	y := rect.Min.Y
	if dx := rect.Dx(); dx > 0 && perform.Random != nil {
		x = rect.Min.X + perform.Random(0, dx-1)
	}
	if dy := rect.Dy(); dy > 0 && perform.Random != nil {
		y = rect.Min.Y + perform.Random(0, dy-1)
	}
	TapR(x, y, delayMs)
}

// TapAreaSafe 区域为空时跳过；返回是否已点击。
func TapAreaSafe(rect image.Rectangle, delayMs int) bool {
	if rect.Empty() {
		return false
	}
	TapArea(rect, delayMs)
	return true
}

// PressBack 按返回键。
func PressBack(delayMs int) {
	if perform.Back != nil {
		perform.Back()
	}
	if delayMs > 0 {
		sleep(delayMs)
	}
}

// SwipeOpts 增强滑动参数，对应 Lua touch.swipeEx 的 opts。
type SwipeOpts struct {
	X1, Y1, X2, Y2 int
	MoveMs         int // 从起点滑到终点的总耗时，默认 600ms
	HoldMs         int // 滑到终点后松手前停留，默认 200ms
	DownMs         int // 按下后开始移动前等待，默认 50ms
	Steps          int // 把滑动拆成几段，默认 1
	PauseMs        int // 多段滑动每段间暂停，默认 0
	UpMs           int // 松手后再等待，默认 0
	ID             int // 手指 ID，默认 1
	BeforeUp       func()
}

// SwipeEx 增强滑动；返回 touchUp 是否成功。
func SwipeEx(opts SwipeOpts) bool {
	if perform.TouchDown == nil || perform.TouchUp == nil {
		return false
	}
	if opts.X1 == 0 && opts.Y1 == 0 && opts.X2 == 0 && opts.Y2 == 0 {
		return false
	}
	id := opts.ID
	if id == 0 {
		id = defaultFingerID
	}
	moveMs := max(1, opts.MoveMs)
	holdMs := max(0, opts.HoldMs)
	downMs := max(0, opts.DownMs)
	steps := max(1, opts.Steps)
	pauseMs := max(0, opts.PauseMs)
	upMs := max(0, opts.UpMs)

	perform.TouchDown(id, opts.X1, opts.Y1)
	if downMs > 0 {
		sleep(downMs)
	}

	segMs := max(1, moveMs/steps)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		xi := opts.X1 + int(math.Round(float64(opts.X2-opts.X1)*t))
		yi := opts.Y1 + int(math.Round(float64(opts.Y2-opts.Y1)*t))
		if perform.TouchMove != nil {
			perform.TouchMove(id, xi, yi, segMs)
		}
		if pauseMs > 0 && i < steps {
			sleep(pauseMs)
		}
	}

	if holdMs > 0 {
		sleep(holdMs)
	}
	if opts.BeforeUp != nil {
		opts.BeforeUp()
	}
	ok := perform.TouchUp(id, opts.X2, opts.Y2)
	if upMs > 0 {
		sleep(upMs)
	}
	return ok
}

// SwipeX 水平滑动。
func SwipeX(x1, x2, y int, opts SwipeOpts) bool {
	opts.X1, opts.Y1 = x1, y
	opts.X2, opts.Y2 = x2, y
	return SwipeEx(opts)
}

// SwipeY 垂直滑动。
func SwipeY(y1, y2, x int, opts SwipeOpts) bool {
	opts.X1, opts.Y1 = x, y1
	opts.X2, opts.Y2 = x, y2
	return SwipeEx(opts)
}
