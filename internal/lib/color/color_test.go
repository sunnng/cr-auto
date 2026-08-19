package color

import (
	"image"
	"sync"
	"testing"
	"time"

	"app/internal/lib/touch"
	"app/internal/vision"
)

func install(t *testing.T, s Screen) func() {
	t.Helper()
	prevScreen, prevHook, prevNow, prevSleep := screen, guardHook, nowFn, sleepFn
	SetScreen(s)
	var mu sync.Mutex
	now := time.Unix(1000, 0)
	SetNow(func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	SetSleep(func(ms int) {
		mu.Lock()
		now = now.Add(time.Duration(ms) * time.Millisecond)
		mu.Unlock()
	})
	return func() {
		SetScreen(prevScreen)
		SetGuardHook(prevHook)
		SetNow(prevNow)
		SetSleep(prevSleep)
	}
}

func TestMatchUsesScreen(t *testing.T) {
	f := vision.Feature{Points: "1|1|ff0000-000000"}
	defer install(t, HitFeatures(f))()
	if !Match(f) {
		t.Fatal("feature must match")
	}
	if Match(vision.Feature{Points: "2|2|00ff00-000000"}) {
		t.Fatal("unregistered feature must not match")
	}
}

func TestMatchWithoutScreen(t *testing.T) {
	defer install(t, nil)()
	if Match(vision.Feature{Points: "1|1|ff0000-000000"}) {
		t.Fatal("no screen must not match")
	}
}

func TestWaitPollsUntilHit(t *testing.T) {
	f := vision.Feature{Points: "1|1|ff0000-000000"}
	want := vision.DetectsColors(f)
	n := 0
	s := NewScriptedScreen()
	s.DetectsFn = func(colors string, sim float32) bool {
		n++
		return n >= 3 && colors == want
	}
	defer install(t, s)()
	ok, which := Wait(f, 10000, 50)
	if !ok || which != -1 {
		t.Fatalf("wait must hit, ok=%v which=%d", ok, which)
	}
}

func TestWaitTimesOut(t *testing.T) {
	defer install(t, NewScriptedScreen())()
	ok, _ := Wait(vision.Feature{Points: "1|1|ff0000-000000"}, 10, 2)
	if ok {
		t.Fatal("wait must time out")
	}
}

func TestWaitAny(t *testing.T) {
	green := vision.Feature{Points: "3|3|00ff00-000000"}
	s := NewScriptedScreen()
	defer install(t, s)()
	features := []vision.Feature{
		{Points: "1|1|ff0000-000000"},
		green,
	}
	s.Hit(green)
	ok, which := Wait(features, 10000, 50)
	if !ok || which != 1 {
		t.Fatalf("wait any: ok=%v which=%d", ok, which)
	}
}

func TestWaitGone(t *testing.T) {
	f := vision.Feature{Points: "1|1|ff0000-000000"}
	want := vision.DetectsColors(f)
	n := 0
	s := NewScriptedScreen()
	s.DetectsFn = func(colors string, sim float32) bool {
		if colors != want {
			return false
		}
		n++
		return n < 3
	}
	defer install(t, s)()
	if !WaitGone(f, 10000, 50) {
		t.Fatal("feature must disappear")
	}
}

func TestTapUntilMatchTapsUntilFeatureAppears(t *testing.T) {
	green := vision.Feature{Points: "2|2|00ff00-000000"}
	want := vision.DetectsColors(green)
	n := 0
	s := NewScriptedScreen()
	s.DetectsFn = func(colors string, sim float32) bool {
		n++
		return n >= 3 && colors == want
	}
	defer install(t, s)()

	var taps int
	touch.SetPerform(touch.Perform{
		Tap:    func(x, y int) { taps++ },
		Random: func(min, max int) int { return max },
	})
	defer touch.SetPerform(touch.Perform{})

	ok, which := TapUntilMatch(
		image.Point{X: 5, Y: 5},
		green,
		TapOpts{TimeoutMs: 10000, IntervalMs: 50, TapDelayMs: 10},
	)
	if !ok || which != -1 {
		t.Fatalf("tap until match failed: ok=%v", ok)
	}
	if taps < 2 {
		t.Fatalf("expected at least two taps, got %d", taps)
	}
}

func TestTapUntilMatchMaxTaps(t *testing.T) {
	defer install(t, NewScriptedScreen())()
	var taps int
	touch.SetPerform(touch.Perform{
		Tap:    func(x, y int) { taps++ },
		Random: func(min, max int) int { return max },
	})
	defer touch.SetPerform(touch.Perform{})

	ok, _ := TapUntilMatch(
		image.Point{X: 5, Y: 5},
		vision.Feature{Points: "2|2|00ff00-000000"},
		TapOpts{TimeoutMs: 10000, IntervalMs: 5, TapDelayMs: 1, MaxTaps: 3},
	)
	if ok {
		t.Fatal("must not match")
	}
	if taps != 3 {
		t.Fatalf("expected 3 taps, got %d", taps)
	}
}

func TestFindAndTapFind(t *testing.T) {
	def := vision.FindDef{
		Region:       image.Rect(0, 0, 10, 10),
		FirstColor:   "ff0000-000000",
		OffsetColors: "1,1,00ff00-000000",
		Sim:          1,
	}
	s := NewScriptedScreen().FindAt(def, image.Pt(4, 4))
	defer install(t, s)()
	x, y, ok := Find(def)
	if !ok || x != 4 || y != 4 {
		t.Fatalf("find: (%d,%d) ok=%v", x, y, ok)
	}
	pt, ok := FindPoint(def)
	if !ok || pt.X != 4 || pt.Y != 4 {
		t.Fatalf("findPoint: %+v ok=%v", pt, ok)
	}

	var taps []image.Point
	touch.SetPerform(touch.Perform{
		Tap:    func(x, y int) { taps = append(taps, image.Point{X: x, Y: y}) },
		Random: func(min, max int) int { return max },
	})
	defer touch.SetPerform(touch.Perform{})
	if !TapFind(def, 0) {
		t.Fatal("tapFind must hit")
	}
	if len(taps) != 1 {
		t.Fatalf("taps=%v", taps)
	}
}

func TestFindAll(t *testing.T) {
	def := vision.FindDef{Region: image.Rect(0, 0, 10, 10), FirstColor: "ff0000-000000", Sim: 1}
	s := NewScriptedScreen().FindAt(def, image.Pt(2, 2), image.Pt(5, 5))
	defer install(t, s)()
	points := FindAll(def)
	if len(points) != 2 {
		t.Fatalf("points=%v", points)
	}
}

func TestMatchRGB(t *testing.T) {
	s := NewScriptedScreen()
	s.cmp[cmpKey{x: 10, y: 20, spec: "ff0000-101010"}] = true
	defer install(t, s)()
	if !MatchRGB(10, 20, "ff0000-101010", 0.95) {
		t.Fatal("cmp must hit")
	}
	if MatchRGB(0, 0, "ff0000-101010", 0.95) {
		t.Fatal("other point must miss")
	}
}

func TestMatchPoints(t *testing.T) {
	f := vision.Feature{Points: "1|1|ff0000-000000,2|2|00ff00-000000"}
	s := NewScriptedScreen()
	pts, err := vision.ParsePoints(f.Points)
	if err != nil {
		t.Fatal(err)
	}
	s.HitPoint(pts[0])
	defer install(t, s)()
	results, ok := MatchPoints(f)
	if ok {
		t.Fatal("partial must not satisfy all-points")
	}
	if len(results) != 2 || !results[0].Matched || results[1].Matched {
		t.Fatalf("results=%+v", results)
	}
}

func TestSessionNestsBeginEnd(t *testing.T) {
	var begins, ends int
	s := &countingScreen{inner: NewScriptedScreen(), onBegin: func() { begins++ }, onEnd: func() { ends++ }}
	defer install(t, s)()
	Session(func() {
		Match(vision.Feature{Points: "1|1|ff0000-000000"})
	})
	if begins != 2 || ends != 2 {
		t.Fatalf("nested session begins=%d ends=%d", begins, ends)
	}
}

type countingScreen struct {
	inner   Screen
	onBegin func()
	onEnd   func()
}

func (c *countingScreen) Begin() {
	c.onBegin()
	c.inner.Begin()
}
func (c *countingScreen) End() {
	c.onEnd()
	c.inner.End()
}
func (c *countingScreen) DetectsMultiColors(colors string, sim float32) bool {
	return c.inner.DetectsMultiColors(colors, sim)
}
func (c *countingScreen) CmpColor(x, y int, colorStr string, sim float32) bool {
	return c.inner.CmpColor(x, y, colorStr, sim)
}
func (c *countingScreen) FindMultiColors(x1, y1, x2, y2 int, colors string, sim float32, dir int) (int, int) {
	return c.inner.FindMultiColors(x1, y1, x2, y2, colors, sim, dir)
}
func (c *countingScreen) FindMultiColorsAll(x1, y1, x2, y2 int, colors string, sim float32, dir int) []image.Point {
	return c.inner.FindMultiColorsAll(x1, y1, x2, y2, colors, sim, dir)
}
