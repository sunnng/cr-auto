package color

import (
	"errors"
	"image"
	"image/color"
	"sync"
	"testing"
	"time"

	"app/internal/lib/touch"
	"app/internal/vision"
)

var errNoFrame = errors.New("color: 无帧来源")

// fakeSource 每次 Capture 按序返回给定帧；耗尽后重复最后一帧。
type fakeSource struct {
	frames []*image.NRGBA
	idx    int
}

func (f *fakeSource) Capture() (*image.NRGBA, error) {
	if len(f.frames) == 0 {
		return nil, errNoFrame
	}
	frame := f.frames[f.idx]
	if f.idx < len(f.frames)-1 {
		f.idx++
	}
	return frame, nil
}

func frameWith(colorAt map[int]uint32) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.SetNRGBA(x, y, color.NRGBA{})
		}
	}
	for key, rgb := range colorAt {
		img.SetNRGBA(key/1000, key%1000, color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb)})
	}
	return img
}

func install(t *testing.T, src FrameSource) func() {
	t.Helper()
	prevSource, prevHook, prevNow, prevSleep := frameSource, guardHook, nowFn, sleepFn
	SetFrameSource(src)
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
		SetFrameSource(prevSource)
		SetGuardHook(prevHook)
		SetNow(prevNow)
		SetSleep(prevSleep)
	}
}

func TestMatchUsesCurrentFrame(t *testing.T) {
	src := &fakeSource{frames: []*image.NRGBA{
		frameWith(map[int]uint32{1001: 0xff0000}),
	}}
	defer install(t, src)()
	f := vision.Feature{Points: "1|1|ff0000-000000"}
	if !Match(f) {
		t.Fatal("feature must match red pixel at (1,1)")
	}
	if Match(vision.Feature{Points: "2|2|00ff00-000000"}) {
		t.Fatal("feature must not match background")
	}
}

func TestMatchWithoutFrameSource(t *testing.T) {
	defer install(t, nil)()
	if Match(vision.Feature{Points: "1|1|ff0000-000000"}) {
		t.Fatal("no frame source must not match")
	}
}

func TestWaitPollsUntilHit(t *testing.T) {
	red := frameWith(map[int]uint32{1001: 0xff0000})
	black := frameWith(nil)
	src := &fakeSource{frames: []*image.NRGBA{black, black, red}}
	defer install(t, src)()
	f := vision.Feature{Points: "1|1|ff0000-000000"}
	ok, which := Wait(f, 10000, 50)
	if !ok || which != -1 {
		t.Fatalf("wait must hit, ok=%v which=%d", ok, which)
	}
}

func TestWaitTimesOut(t *testing.T) {
	src := &fakeSource{frames: []*image.NRGBA{frameWith(nil)}}
	defer install(t, src)()
	ok, _ := Wait(vision.Feature{Points: "1|1|ff0000-000000"}, 10, 2)
	if ok {
		t.Fatal("wait must time out")
	}
}

func TestWaitAny(t *testing.T) {
	green := frameWith(map[int]uint32{3003: 0x00ff00})
	src := &fakeSource{frames: []*image.NRGBA{frameWith(nil), green}}
	defer install(t, src)()
	features := []vision.Feature{
		{Points: "1|1|ff0000-000000"},
		{Points: "3|3|00ff00-000000"},
	}
	ok, which := Wait(features, 10000, 50)
	if !ok || which != 1 {
		t.Fatalf("wait any: ok=%v which=%d", ok, which)
	}
}

func TestWaitGone(t *testing.T) {
	red := frameWith(map[int]uint32{1001: 0xff0000})
	src := &fakeSource{frames: []*image.NRGBA{red, red, frameWith(nil)}}
	defer install(t, src)()
	f := vision.Feature{Points: "1|1|ff0000-000000"}
	if !WaitGone(f, 10000, 50) {
		t.Fatal("feature must disappear")
	}
}

func TestTapUntilMatchTapsUntilFeatureAppears(t *testing.T) {
	green := frameWith(map[int]uint32{2002: 0x00ff00})
	src := &fakeSource{frames: []*image.NRGBA{frameWith(nil), frameWith(nil), green}}
	defer install(t, src)()

	var taps int
	touch.SetPerform(touch.Perform{
		Tap: func(x, y int) { taps++ },
		Sleep: func(ms int) {
			// 每次点击后模拟画面变化：已由帧序列表达。
		},
		Random: func(min, max int) int { return max },
	})
	defer touch.SetPerform(touch.Perform{})

	ok, which := TapUntilMatch(
		image.Point{X: 5, Y: 5},
		vision.Feature{Points: "2|2|00ff00-000000"},
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
	src := &fakeSource{frames: []*image.NRGBA{frameWith(nil)}}
	defer install(t, src)()
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
	src := &fakeSource{frames: []*image.NRGBA{
		frameWith(map[int]uint32{4004: 0xff0000, 5005: 0x00ff00}),
	}}
	defer install(t, src)()
	def := vision.FindDef{
		Region:       image.Rect(0, 0, 10, 10),
		FirstColor:   "ff0000-000000",
		OffsetColors: "1,1,00ff00-000000",
		Sim:          1,
	}
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
	src := &fakeSource{frames: []*image.NRGBA{
		frameWith(map[int]uint32{2002: 0xff0000, 5005: 0xff0000}),
	}}
	defer install(t, src)()
	def := vision.FindDef{Region: image.Rect(0, 0, 10, 10), FirstColor: "ff0000-000000", Sim: 1}
	points := FindAll(def)
	if len(points) != 2 {
		t.Fatalf("points=%v", points)
	}
}
