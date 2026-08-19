package dialog

import (
	"image"
	"sync"
	"testing"

	libcolor "app/internal/lib/color"
	"app/internal/lib/touch"
	"app/internal/vision"
)

type fakeTouches struct {
	mu     sync.Mutex
	points []image.Point
}

func (f *fakeTouches) tap(x, y int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.points = append(f.points, image.Point{X: x, Y: y})
}

func (f *fakeTouches) taps() []image.Point {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]image.Point(nil), f.points...)
}

func setupDialogTest(t *testing.T, hits libcolor.Screen) *fakeTouches {
	t.Helper()
	ft := &fakeTouches{}
	if hits == nil {
		hits = libcolor.NewScriptedScreen()
	}
	libcolor.SetScreen(hits)
	libcolor.SetSleep(func(ms int) {})
	touch.SetPerform(touch.Perform{
		Tap:    ft.tap,
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})
	t.Cleanup(func() {
		libcolor.SetScreen(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
	})
	return ft
}

var testFeature = vision.Feature{Points: "100|100|ff0000-000000", Sim: 1}

func TestDialogIsVisibleAndTapConfirm(t *testing.T) {
	ft := setupDialogTest(t, libcolor.HitFeatures(testFeature))
	d := New(Def{
		Name:       "网络联机状态不稳定",
		Feature:    testFeature,
		ConfirmBtn: image.Rect(775, 621, 828, 647),
	}, "[Test]")

	if !d.IsVisible() {
		t.Fatal("dialog with matching feature must be visible")
	}
	ok, reason := d.Handle(HandleOpts{Mode: modeIfVisible, Action: "confirm"})
	if !ok || reason != "" {
		t.Fatalf("handle failed: ok=%v reason=%q", ok, reason)
	}
	taps := ft.taps()
	if len(taps) != 1 || taps[0] != (image.Point{X: 775, Y: 621}) {
		t.Fatalf("confirm must tap button top-left, got %+v", taps)
	}
}

func TestDialogNotVisibleSkipped(t *testing.T) {
	ft := setupDialogTest(t, nil)
	d := New(Def{Name: "弹窗", Feature: testFeature}, "[Test]")
	ok, reason := d.Handle(HandleOpts{Mode: modeIfVisible, Action: "confirm"})
	if !ok || reason != "" {
		t.Fatalf("invisible dialog must be skipped: ok=%v reason=%q", ok, reason)
	}
	if len(ft.taps()) != 0 {
		t.Fatal("invisible dialog must not tap")
	}
}

func TestDialogConfirmAndWaitGone(t *testing.T) {
	want := vision.DetectsColors(testFeature)
	s := libcolor.NewScriptedScreen()
	ft := setupDialogTest(t, s)
	s.DetectsFn = func(colors string, sim float32) bool {
		if colors != want {
			return false
		}
		return len(ft.taps()) == 0
	}

	d := New(Def{
		Name:       "弹窗",
		Feature:    testFeature,
		ConfirmBtn: image.Rect(10, 10, 20, 20),
	}, "[Test]")
	ok, reason := d.Handle(HandleOpts{Mode: modeFlow, Action: "confirm", WaitGoneMs: 2000})
	if !ok || reason != "" {
		t.Fatalf("handle with waitGone must succeed: ok=%v reason=%q", ok, reason)
	}
}

func TestToGuardHandlerConfirm(t *testing.T) {
	ft := setupDialogTest(t, libcolor.HitFeatures(testFeature))
	d := New(Def{
		Name:       "网络联机状态不稳定",
		Feature:    testFeature,
		ConfirmBtn: image.Rect(775, 621, 828, 647),
	}, "[Register]")
	handler := d.ToGuardHandler(HandleOpts{Action: "confirm", WaitGoneMs: 2000})
	s := libcolor.HitFeatures(testFeature)
	libcolor.SetScreen(s)
	s.DetectsFn = func(colors string, sim float32) bool {
		if colors != vision.DetectsColors(testFeature) {
			return false
		}
		return len(ft.taps()) == 0
	}
	handler()
	taps := ft.taps()
	if len(taps) != 1 {
		t.Fatalf("guard handler must confirm once, taps=%+v", taps)
	}
}

func TestResolveUntilIdleHandlesBothDialogs(t *testing.T) {
	green := vision.Feature{Points: "200|200|00ff00-000000", Sim: 1}
	s := libcolor.NewScriptedScreen()
	ft := setupDialogTest(t, s)
	c1 := vision.DetectsColors(testFeature)
	c2 := vision.DetectsColors(green)
	s.DetectsFn = func(colors string, sim float32) bool {
		if len(ft.taps()) >= 2 {
			return false
		}
		return colors == c1 || colors == c2
	}

	d1 := New(Def{Name: "confirmCookie", Feature: testFeature, ConfirmBtn: image.Rect(10, 10, 20, 20)}, "[Test]")
	d2 := New(Def{Name: "cookieCountWarning", Feature: green, ConfirmBtn: image.Rect(30, 30, 40, 40)}, "[Test]")

	ok, summary := ResolveUntilIdle([]Candidate{
		{Name: "confirmCookie", Dialog: d1, Priority: 10, Opts: HandleOpts{Mode: modeIfVisible, Action: "confirm", WaitGoneMs: -1}},
		{Name: "cookieCountWarning", Dialog: d2, Priority: 10, Opts: HandleOpts{Mode: modeIfVisible, Action: "confirm", WaitGoneMs: -1}},
	}, ResolveOpts{TimeoutMs: 5000, MinWaitMs: 10, SettleMs: 10, IntervalMs: 1, MaxHandled: 2, Tag: "[Test]"})

	if !ok {
		t.Fatalf("resolveUntilIdle must succeed, summary=%+v", summary)
	}
	if summary.Handled != 2 {
		t.Fatalf("both dialogs must be handled, got %d (%v)", summary.Handled, summary.Names)
	}
	if len(ft.taps()) != 2 {
		t.Fatalf("two confirms expected, taps=%+v", ft.taps())
	}
}

func TestResolveUntilIdleNoDialogs(t *testing.T) {
	setupDialogTest(t, nil)
	d := New(Def{Name: "弹窗", Feature: testFeature}, "[Test]")
	ok, summary := ResolveUntilIdle([]Candidate{
		{Name: "弹窗", Dialog: d, Priority: 10, Opts: HandleOpts{Mode: modeIfVisible}},
	}, ResolveOpts{TimeoutMs: 2000, MinWaitMs: 10, SettleMs: 10, IntervalMs: 1, Tag: "[Test]"})
	if !ok || summary.Handled != 0 {
		t.Fatalf("no dialogs must yield ok with handled=0: ok=%v summary=%+v", ok, summary)
	}
}
