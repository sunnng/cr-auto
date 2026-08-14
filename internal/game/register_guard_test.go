package game

import (
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"app/internal/core"
	"app/internal/game/popup"
	libcolor "app/internal/lib/color"
	"app/internal/lib/store"
	"app/internal/lib/touch"
)

// touchRecorder 记录触控点。
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
	libcolor.SetFrameSource(nil)
	libcolor.SetSleep(func(ms int) {})
	touch.SetPerform(touch.Perform{
		Tap:    rec.tap,
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})
	store.SetDefault(store.New(filepath.Join(t.TempDir(), "store.json")))
	t.Cleanup(func() {
		libcolor.SetFrameSource(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
		store.SetDefault(nil)
	})
	return rec
}

// popupFrame 生成带弹窗特征的帧。
func popupFrame() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 1600, 900))
	spec := popup.UnstableNetworkDef().Feature.Points
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
		img.SetNRGBA(x, y, color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 0xff})
	}
	return img
}

type switchableFrame struct {
	mu  sync.Mutex
	get func() *image.NRGBA
}

func (s *switchableFrame) Capture() (*image.NRGBA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.get(), nil
}

func TestRegisterAllGuardTrapHandlesUnstableNetwork(t *testing.T) {
	rec := setupRegisterTest(t)

	// 点击确认后弹窗消失。
	fs := &switchableFrame{get: func() *image.NRGBA {
		if len(rec.taps()) > 0 {
			return image.NewNRGBA(image.Rect(0, 0, 1600, 900))
		}
		return popupFrame()
	}}
	libcolor.SetFrameSource(fs)

	s := core.NewScheduler()
	g := core.NewGuard()
	RegisterAll(s, g)

	if !g.Check() {
		t.Fatal("guard must hit the unstable-network trap")
	}
	taps := rec.taps()
	if len(taps) != 1 || taps[0] != (image.Point{X: 775, Y: 621}) {
		t.Fatalf("trap must confirm at 775,621, got %+v", taps)
	}
	// 弹窗已消失：再次扫描不再命中。
	if g.Check() {
		t.Fatal("guard must not hit after popup gone")
	}
}
