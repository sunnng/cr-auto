package touch

import (
	"image"
	"testing"
)

type fakeRecord struct {
	taps        []image.Point
	downs       []image.Point
	moves       []image.Point
	ups         int
	backs       int
	sleeps      []int
	beforeUpRun int
	randValues  []int
	randIdx     int
}

func fakePerform(t *testing.T) (*fakeRecord, func()) {
	t.Helper()
	rec := &fakeRecord{}
	prev := perform
	perform = Perform{
		Tap:       func(x, y int) { rec.taps = append(rec.taps, image.Point{X: x, Y: y}) },
		TouchDown: func(id, x, y int) { rec.downs = append(rec.downs, image.Point{X: x, Y: y}) },
		TouchMove: func(id, x, y, ms int) { rec.moves = append(rec.moves, image.Point{X: x, Y: y}) },
		TouchUp:   func(id, x, y int) bool { rec.ups++; return true },
		Back:      func() { rec.backs++ },
		Sleep:     func(ms int) { rec.sleeps = append(rec.sleeps, ms) },
		Random: func(min, max int) int {
			rec.randValues = append(rec.randValues, max)
			return max
		},
	}
	restore := func() { perform = prev }
	return rec, restore
}

func TestTapRAppliesJitterAndSleeps(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	TapR(100, 200, 800)
	if len(rec.taps) != 1 {
		t.Fatalf("taps=%v", rec.taps)
	}
	got := rec.taps[0]
	if got.X < 97 || got.X > 103 || got.Y < 197 || got.Y > 203 {
		t.Fatalf("tap must stay within jitter range: %+v", got)
	}
	if len(rec.sleeps) != 1 || rec.sleeps[0] != 800 {
		t.Fatalf("sleeps=%v", rec.sleeps)
	}
}

func TestTapRWithoutDelaySleepsRandom(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	TapR(10, 10, 0)
	if len(rec.sleeps) != 1 {
		t.Fatalf("expected one sleep, got %v", rec.sleeps)
	}
}

func TestTapAreaPointLandsInsideRect(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	TapArea(image.Rect(100, 200, 200, 300), 0)
	if len(rec.taps) != 1 {
		t.Fatalf("taps=%v", rec.taps)
	}
	p := rec.taps[0]
	if p.X < 100-3 || p.X > 200+3 || p.Y < 200-3 || p.Y > 300+3 {
		t.Fatalf("tap outside rect (with jitter): %+v", p)
	}
}

func TestTapAreaSafeSkipsNilRect(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	if TapAreaSafe(image.Rectangle{}, 0) {
		t.Fatal("empty rect must not tap")
	}
	if !TapAreaSafe(image.Rect(0, 0, 10, 10), 0) {
		t.Fatal("valid rect must tap")
	}
	if len(rec.taps) != 1 {
		t.Fatalf("taps=%v", rec.taps)
	}
}

func TestPressBack(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	PressBack(500)
	if rec.backs != 1 {
		t.Fatal("back not pressed")
	}
	if len(rec.sleeps) != 1 || rec.sleeps[0] != 500 {
		t.Fatalf("sleeps=%v", rec.sleeps)
	}
}

func TestSwipeExInterpolatesSteps(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	ok := SwipeEx(SwipeOpts{X1: 0, Y1: 0, X2: 100, Y2: 0, MoveMs: 100, Steps: 4, HoldMs: 10, DownMs: 5, UpMs: 3})
	if !ok {
		t.Fatal("swipe must report success")
	}
	if len(rec.downs) != 1 || rec.downs[0] != (image.Point{}) {
		t.Fatalf("downs=%v", rec.downs)
	}
	if len(rec.moves) != 4 {
		t.Fatalf("moves=%v", rec.moves)
	}
	want := []image.Point{{X: 25}, {X: 50}, {X: 75}, {X: 100}}
	for i := range want {
		if rec.moves[i] != want[i] {
			t.Fatalf("move[%d]=%+v want %+v", i, rec.moves[i], want[i])
		}
	}
	if rec.ups != 1 {
		t.Fatal("touch up missing")
	}
}

func TestSwipeExRequiresEndpoints(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	if SwipeEx(SwipeOpts{X1: 0, Y1: 0}) {
		t.Fatal("swipe without x2/y2 must fail")
	}
	if len(rec.downs) != 0 {
		t.Fatal("invalid swipe must not touch")
	}
}

func TestSwipeExBeforeUpRunsBeforeUp(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	called := false
	ok := SwipeEx(SwipeOpts{
		X1: 0, Y1: 0, X2: 10, Y2: 0, Steps: 1,
		BeforeUp: func() {
			called = true
			if rec.ups != 0 {
				t.Fatal("beforeUp must run before touch up")
			}
		},
	})
	if !ok || !called {
		t.Fatal("beforeUp hook must run")
	}
}

func TestSwipeXAndYHelpers(t *testing.T) {
	rec, restore := fakePerform(t)
	defer restore()
	SwipeX(0, 100, 50, SwipeOpts{MoveMs: 60})
	SwipeY(0, 100, 50, SwipeOpts{MoveMs: 60})
	if len(rec.downs) != 2 {
		t.Fatalf("downs=%v", rec.downs)
	}
	if rec.downs[0] != (image.Point{X: 0, Y: 50}) || rec.downs[1] != (image.Point{X: 50, Y: 0}) {
		t.Fatalf("helper endpoints wrong: %+v", rec.downs)
	}
}

func TestActionCountIncrementsPerAction(t *testing.T) {
	_, restore := fakePerform(t)
	defer restore()
	ResetActionCount()
	TapR(10, 10, 100)
	PressBack(0)
	SwipeEx(SwipeOpts{X1: 0, Y1: 0, X2: 20, Y2: 20, MoveMs: 50})
	if got := ActionCount(); got != 3 {
		t.Fatalf("action count=%d want 3", got)
	}
}

func TestActionHookReceivesRunningCount(t *testing.T) {
	_, restore := fakePerform(t)
	defer restore()
	var counts []int
	prevHook := actionHook
	actionHook = func(count int) { counts = append(counts, count) }
	defer func() { actionHook = prevHook }()
	ResetActionCount()
	TapR(10, 10, 100)
	TapR(20, 20, 100)
	if len(counts) != 2 || counts[0] != 1 || counts[1] != 2 {
		t.Fatalf("hook counts=%v want [1 2]", counts)
	}
}

func TestResetActionCount(t *testing.T) {
	_, restore := fakePerform(t)
	defer restore()
	TapR(10, 10, 100)
	ResetActionCount()
	if ActionCount() != 0 {
		t.Fatalf("count after reset=%d", ActionCount())
	}
	TapR(20, 20, 100)
	if ActionCount() != 1 {
		t.Fatalf("count after reset+1 tap=%d", ActionCount())
	}
}
