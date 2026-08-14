package dialog

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"sync"
	"testing"

	libcolor "app/internal/lib/color"
	"app/internal/lib/touch"
	"app/internal/vision"
)

// fakeFrame 固定帧来源。
type fakeFrame struct {
	img *image.NRGBA
}

func (f *fakeFrame) Capture() (*image.NRGBA, error) { return f.img, nil }

// switchableFrame 每次截图返回不同帧（模拟弹窗消失/出现）。
type switchableFrame struct {
	mu  sync.Mutex
	get func() *image.NRGBA
}

func (s *switchableFrame) Capture() (*image.NRGBA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.get(), nil
}

// newFrame 生成一张带指定特征点（"x|y|rrggbb-…" 串）的帧。
func newFrame(points ...string) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 1600, 900))
	for _, spec := range points {
		for _, chunk := range strings.Split(spec, ",") {
			parts := strings.Split(chunk, "|")
			if len(parts) < 3 {
				continue
			}
			x, _ := strconv.Atoi(parts[0])
			y, _ := strconv.Atoi(parts[1])
			hex := parts[2]
			if dash := strings.LastIndex(hex, "-"); dash >= 0 {
				hex = hex[:dash]
			}
			rgb, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				continue
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 0xff,
			})
		}
	}
	return img
}

// fakeTouches 记录触控点（TapArea 经 Random 固定 0 折算为区域左上角点击）。
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

func setupDialogTest(t *testing.T, frame *image.NRGBA) *fakeTouches {
	t.Helper()
	ft := &fakeTouches{}
	libcolor.SetFrameSource(&fakeFrame{img: frame})
	libcolor.SetSleep(func(ms int) {})
	touch.SetPerform(touch.Perform{
		Tap:    ft.tap,
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})
	t.Cleanup(func() {
		libcolor.SetFrameSource(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
	})
	return ft
}

const testFeature = "100|100|ff0000-000000"

func TestDialogIsVisibleAndTapConfirm(t *testing.T) {
	ft := setupDialogTest(t, newFrame(testFeature))
	d := New(Def{
		Name:       "网络联机状态不稳定",
		Feature:    vision.Feature{Points: testFeature, Sim: 1},
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
	ft := setupDialogTest(t, newFrame())
	d := New(Def{Name: "弹窗", Feature: vision.Feature{Points: testFeature, Sim: 1}}, "[Test]")
	ok, reason := d.Handle(HandleOpts{Mode: modeIfVisible, Action: "confirm"})
	if !ok || reason != "" {
		t.Fatalf("invisible dialog must be skipped: ok=%v reason=%q", ok, reason)
	}
	if len(ft.taps()) != 0 {
		t.Fatal("invisible dialog must not tap")
	}
}

func TestDialogConfirmAndWaitGone(t *testing.T) {
	frames := []*image.NRGBA{newFrame(testFeature), newFrame()}
	fs := &switchableFrame{get: func() *image.NRGBA { return frames[0] }}
	setupDialogTest(t, nil)
	libcolor.SetFrameSource(fs)

	// 点击后切到“弹窗已消失”的帧。
	d := New(Def{
		Name:       "弹窗",
		Feature:    vision.Feature{Points: testFeature, Sim: 1},
		ConfirmBtn: image.Rect(10, 10, 20, 20),
	}, "[Test]")
	steps := 0
	fs.get = func() *image.NRGBA {
		steps++
		if steps > 2 {
			return frames[1]
		}
		return frames[0]
	}
	ok, reason := d.Handle(HandleOpts{Mode: modeFlow, Action: "confirm", WaitGoneMs: 2000})
	if !ok || reason != "" {
		t.Fatalf("handle with waitGone must succeed: ok=%v reason=%q", ok, reason)
	}
}

func TestToGuardHandlerConfirm(t *testing.T) {
	ft := setupDialogTest(t, newFrame(testFeature))
	d := New(Def{
		Name:       "网络联机状态不稳定",
		Feature:    vision.Feature{Points: testFeature, Sim: 1},
		ConfirmBtn: image.Rect(775, 621, 828, 647),
	}, "[Register]")
	handler := d.ToGuardHandler(HandleOpts{Action: "confirm", WaitGoneMs: 2000})
	handler()
	taps := ft.taps()
	if len(taps) != 1 {
		t.Fatalf("guard handler must confirm once, taps=%+v", taps)
	}
}

func TestResolveUntilIdleHandlesBothDialogs(t *testing.T) {
	// 两个弹窗特征都在同一帧上 → 两轮各处理一个，直至空闲。
	ft := setupDialogTest(t, newFrame(testFeature, "200|200|00ff00-000000"))
	frames := []*image.NRGBA{newFrame(testFeature, "200|200|00ff00-000000"), newFrame()}
	fs := &switchableFrame{get: func() *image.NRGBA { return frames[0] }}
	libcolor.SetFrameSource(fs)

	d1 := New(Def{Name: "confirmCookie", Feature: vision.Feature{Points: testFeature, Sim: 1}, ConfirmBtn: image.Rect(10, 10, 20, 20)}, "[Test]")
	d2 := New(Def{Name: "cookieCountWarning", Feature: vision.Feature{Points: "200|200|00ff00-000000", Sim: 1}, ConfirmBtn: image.Rect(30, 30, 40, 40)}, "[Test]")

	steps := 0
	fs.get = func() *image.NRGBA {
		steps++
		if steps > 4 {
			return frames[1]
		}
		return frames[0]
	}

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
	setupDialogTest(t, newFrame())
	d := New(Def{Name: "弹窗", Feature: vision.Feature{Points: testFeature, Sim: 1}}, "[Test]")
	ok, summary := ResolveUntilIdle([]Candidate{
		{Name: "弹窗", Dialog: d, Priority: 10, Opts: HandleOpts{Mode: modeIfVisible}},
	}, ResolveOpts{TimeoutMs: 2000, MinWaitMs: 10, SettleMs: 10, IntervalMs: 1, Tag: "[Test]"})
	if !ok || summary.Handled != 0 {
		t.Fatalf("no dialogs must yield ok with handled=0: ok=%v summary=%+v", ok, summary)
	}
}
