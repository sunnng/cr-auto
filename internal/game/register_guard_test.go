package game

import (
	"image"
	"path/filepath"
	"sync"
	"testing"

	"app/internal/core"
	"app/internal/game/popup"
	libcolor "app/internal/lib/color"
	"app/internal/lib/store"
	"app/internal/lib/touch"
	"app/internal/vision"
)

type touchRecorder struct {
	mu     sync.Mutex
	points []image.Point
}

func (t *touchRecorder) tap(x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.points = append(t.points, image.Point{X: x, Y: y})
}

func (t *touchRecorder) taps() []image.Point {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]image.Point(nil), t.points...)
}

func setupRegisterTest(t *testing.T) *touchRecorder {
	t.Helper()
	rec := &touchRecorder{}
	libcolor.SetScreen(libcolor.NewScriptedScreen())
	libcolor.SetSleep(func(ms int) {})
	touch.SetPerform(touch.Perform{
		Tap:    rec.tap,
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})
	store.SetDefault(store.New(filepath.Join(t.TempDir(), "store.json")))
	t.Cleanup(func() {
		libcolor.SetScreen(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
		store.SetDefault(nil)
	})
	return rec
}

func TestRegisterAllGuardTrapHandlesUnstableNetwork(t *testing.T) {
	rec := setupRegisterTest(t)
	feat := popup.UnstableNetworkDef().Feature
	want := vision.DetectsColors(feat)
	s := libcolor.NewScriptedScreen()
	s.DetectsFn = func(colors string, sim float32) bool {
		if colors != want {
			return false
		}
		return len(rec.taps()) == 0
	}
	libcolor.SetScreen(s)

	sched := core.NewScheduler()
	g := core.NewGuard()
	RegisterAll(sched, g)

	if !g.Check() {
		t.Fatal("guard must hit the unstable-network trap")
	}
	taps := rec.taps()
	if len(taps) != 1 || taps[0] != (image.Point{X: 775, Y: 621}) {
		t.Fatalf("trap must confirm at 775,621, got %+v", taps)
	}
	if g.Check() {
		t.Fatal("guard must not hit after popup gone")
	}
}
