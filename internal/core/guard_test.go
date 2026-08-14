package core

import (
	"image"
	icolor "image/color"
	"testing"

	"app/internal/lib/color"
	"app/internal/vision"
)

// frameSource 固定帧来源。
type frameSource struct {
	frames []*image.NRGBA
	idx    int
}

func (f *frameSource) Capture() (*image.NRGBA, error) {
	frame := f.frames[f.idx]
	if f.idx < len(f.frames)-1 {
		f.idx++
	}
	return frame, nil
}

func redFrameAt(keys ...int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.SetNRGBA(x, y, icolor.NRGBA{})
		}
	}
	for _, key := range keys {
		img.SetNRGBA(key/1000, key%1000, icolor.NRGBA{R: 0xff})
	}
	return img
}

func installGuardFrame(t *testing.T, frame *image.NRGBA) func() {
	t.Helper()
	color.SetFrameSource(&frameSource{frames: []*image.NRGBA{frame}})
	return func() { color.SetFrameSource(nil) }
}

func TestGuardRegisterClearAndCount(t *testing.T) {
	g := NewGuard()
	if g.TrapCount() != 0 {
		t.Fatal("fresh guard must be empty")
	}
	g.Register("弹窗A", func() bool { return false }, func() {}, 10)
	g.Register("弹窗B", func() bool { return false }, func() {}, 5)
	if g.TrapCount() != 2 {
		t.Fatalf("count=%d", g.TrapCount())
	}
	g.Clear()
	if g.TrapCount() != 0 {
		t.Fatal("clear must empty the guard")
	}
}

func TestGuardCheckHandlesFirstHit(t *testing.T) {
	defer installGuardFrame(t, redFrameAt(1001))()
	var handled []string
	g := NewGuard()
	g.Register("弹窗A", func() bool { return false }, func() { handled = append(handled, "A") }, 0)
	g.Register("弹窗B", vision.Feature{Points: "1|1|ff0000-000000"}, func() { handled = append(handled, "B") }, 0)
	if !g.Check() {
		t.Fatal("check must handle the matching trap")
	}
	if len(handled) != 1 || handled[0] != "B" {
		t.Fatalf("handled=%v", handled)
	}
}

func TestGuardCheckPriorityOrder(t *testing.T) {
	defer installGuardFrame(t, redFrameAt(1001))()
	var handled []string
	g := NewGuard()
	// 低优先级先注册，高优先级后注册；两者都命中时高优先级先处理。
	g.Register("low", func() bool { return true }, func() { handled = append(handled, "low") }, 1)
	g.Register("high", func() bool { return true }, func() { handled = append(handled, "high") }, 10)
	if !g.Check() {
		t.Fatal("check must handle")
	}
	if len(handled) != 1 || handled[0] != "high" {
		t.Fatalf("priority order violated: %v", handled)
	}
}

func TestGuardCheckNameTieBreak(t *testing.T) {
	defer installGuardFrame(t, redFrameAt(1001))()
	var handled []string
	g := NewGuard()
	g.Register("zeta", func() bool { return true }, func() { handled = append(handled, "zeta") }, 5)
	g.Register("alpha", func() bool { return true }, func() { handled = append(handled, "alpha") }, 5)
	g.Check()
	if len(handled) != 1 || handled[0] != "alpha" {
		t.Fatalf("equal priority must break ties by name: %v", handled)
	}
}

func TestGuardCheckSkipsWhenNothingMatches(t *testing.T) {
	defer installGuardFrame(t, redFrameAt(1001))()
	g := NewGuard()
	g.Register("miss", func() bool { return false }, func() { t.Fatal("must not run") }, 0)
	if g.Check() {
		t.Fatal("check must report no hit")
	}
}

func TestGuardCheckStopsOnFirstHandled(t *testing.T) {
	defer installGuardFrame(t, redFrameAt(1001))()
	var handled []string
	g := NewGuard()
	g.Register("hit", func() bool { return true }, func() { handled = append(handled, "hit") }, 0)
	g.Register("alsoHit", func() bool { return true }, func() { handled = append(handled, "alsoHit") }, 0)
	g.Check()
	if len(handled) != 1 {
		t.Fatalf("check must stop after first handled trap: %v", handled)
	}
}

func TestGuardCheckHandlerPanicReportsFailure(t *testing.T) {
	defer installGuardFrame(t, redFrameAt(1001))()
	g := NewGuard()
	g.Register("panic", func() bool { return true }, func() { panic("boom") }, 0)
	if g.Check() {
		t.Fatal("handler panic must make check return false")
	}
}

func TestGuardSleepFragmentsWithGuardChecks(t *testing.T) {
	defer installGuardFrame(t, redFrameAt(1001))()
	var checks, sleeps []int
	g := NewGuard()
	g.SetSleep(func(ms int) { sleeps = append(sleeps, ms) })
	g.Register("count", func() bool {
		checks = append(checks, 1)
		return false
	}, func() {}, 0)
	g.Sleep(1200, 500)
	// 分片：500 + 500 + 200，每次分片前各做一次守卫扫描。
	if len(sleeps) != 3 || sleeps[0] != 500 || sleeps[1] != 500 || sleeps[2] != 200 {
		t.Fatalf("sleeps=%v", sleeps)
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 guard checks, got %d", len(checks))
	}
}
