package starlight

import (
	"image"
	"path/filepath"
	"sync"
	"testing"

	libcolor "app/internal/lib/color"
	"app/internal/lib/ocr"
	"app/internal/lib/store"
	"app/internal/lib/touch"
)

// ============ 测试辅助 ============

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

type fakeOcr struct {
	byRect map[image.Rectangle]string
}

func (f *fakeOcr) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
	if raw, ok := f.byRect[rect]; ok {
		if returnType == "json" {
			return `[{"words":"` + raw + `","location":[[0,0],[10,0],[10,10],[0,10]]}]`, nil
		}
		return raw, nil
	}
	return "", nil
}

func (f *fakeOcr) FindTapPoint(text string, rect image.Rectangle) (int, int, bool) {
	return 0, 0, false
}

func setupTest(t *testing.T, hits libcolor.Screen, eng *fakeOcr) *touchRecorder {
	t.Helper()
	rec := &touchRecorder{}
	if hits == nil {
		hits = libcolor.NewScriptedScreen()
	}
	libcolor.SetScreen(hits)
	libcolor.SetSleep(func(ms int) {})
	touch.SetPerform(touch.Perform{
		Tap:    rec.tap,
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})
	store.SetDefault(store.New(filepath.Join(t.TempDir(), "store.json")))
	if eng == nil {
		eng = &fakeOcr{byRect: map[image.Rectangle]string{}}
	}
	ocr.SetEngine(eng)
	t.Cleanup(func() {
		libcolor.SetScreen(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
		store.SetDefault(nil)
		ocr.SetEngine(nil)
	})
	return rec
}

// ============ 会话测试 ============

func TestSessionDoneTodayLifecycle(t *testing.T) {
	setupTest(t, nil, nil)
	if IsDoneToday() {
		t.Fatal("fresh store must not be done today")
	}
	MarkDoneToday()
	if !IsDoneToday() {
		t.Fatal("markDoneToday must set today's date")
	}
	Clear()
	if IsDoneToday() {
		t.Fatal("clear must reset done date")
	}
}

// ============ 页面测试 ============

func TestStarlightPageDetection(t *testing.T) {
	features := FeatureLib()
	setupTest(t, libcolor.HitFeatures(features.Home.Feature), nil)
	if !IsHomePage() {
		t.Fatal("home feature must be detected")
	}
	setupTest(t, libcolor.HitFeatures(features.Manual.Feature), nil)
	if !IsManualPage() {
		t.Fatal("manual feature must be detected")
	}
	setupTest(t, libcolor.HitFeatures(features.Vanilla.Feature), nil)
	if !IsVanillaIslandPage() {
		t.Fatal("vanilla feature must be detected")
	}
	setupTest(t, libcolor.HitFeatures(features.Task.Feature), nil)
	if !IsTaskPage() {
		t.Fatal("task feature must be detected")
	}
}

func TestFindClaimableBtn(t *testing.T) {
	features := FeatureLib()
	def := features.Task.ClaimableBtn
	s := libcolor.NewScriptedScreen().FindAt(def, image.Pt(def.Region.Min.X, def.Region.Min.Y))
	setupTest(t, s, nil)
	x, y, ok := FindClaimableBtn()
	if !ok || x != def.Region.Min.X || y != def.Region.Min.Y {
		t.Fatalf("claimable btn at (%d,%d) ok=%v want (%d,%d)", x, y, ok, def.Region.Min.X, def.Region.Min.Y)
	}
}
