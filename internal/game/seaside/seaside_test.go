package seaside

import (
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	libcolor "app/internal/lib/color"
	"app/internal/lib/ocr"
	"app/internal/lib/store"
	"app/internal/lib/touch"
	"app/internal/vision"
)

// ============ 测试辅助 ============

type fakeFrame struct{ img *image.NRGBA }

func (f *fakeFrame) Capture() (*image.NRGBA, error) { return f.img, nil }

func frameOf(points ...string) *image.NRGBA {
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
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 0xff})
		}
	}
	return img
}

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
		startupBypassPending = true
		startupBypassActive = false
	})
	return rec
}

func fSpec(f vision.Feature) string { return f.Points }

// ============ 会话测试 ============

func TestSessionRestockLifecycle(t *testing.T) {
	setupTest(t, nil, nil)
	if RestoreProgress() != 0 {
		t.Fatal("no record must have zero progress")
	}
	// 消费启动首轮强制，避免干扰补货等待检查。
	CheckReady()
	ScheduleAfterRestock(3600)
	remain := RestoreProgress()
	if remain <= 0 || remain > 3600+3600 {
		t.Fatalf("restoreProgress must report remain, got %d", remain)
	}
	ok, remain := CheckReady()
	if ok || remain <= 0 {
		t.Fatalf("must be waiting: ok=%v remain=%d", ok, remain)
	}
	Clear()
	if ok, _ := CheckReady(); !ok {
		t.Fatal("cleared session must be ready")
	}
}

func TestSessionStartupBypass(t *testing.T) {
	setupTest(t, nil, nil)
	// 首轮强制执行：首次 checkReady 一定可运行。
	ok, _ := CheckReady()
	if !ok {
		t.Fatal("first checkReady must bypass (startup bypass)")
	}
	// 写入补货等待后，第二次 checkReady 必须等待。
	ScheduleAfterRestock(3600)
	ok, remain := CheckReady()
	if ok || remain <= 0 {
		t.Fatalf("second checkReady must wait: ok=%v remain=%d", ok, remain)
	}
	// consumeStartupBypass 已被 checkReady 置为 true 并保持到任务侧。
}

func TestConsumeStartupBypass(t *testing.T) {
	setupTest(t, nil, nil)
	if ConsumeStartupBypass() {
		t.Fatal("bypass must not be active before first checkReady")
	}
	CheckReady()
	if !ConsumeStartupBypass() {
		t.Fatal("startup bypass must be active after first checkReady")
	}
	if ConsumeStartupBypass() {
		t.Fatal("startup bypass must be one-shot")
	}
}

// ============ 页面测试 ============

func TestSeasidePageDetection(t *testing.T) {
	features := FeatureLib()
	setupTest(t, libcolor.HitFeatures(features.Page.Feature), nil)
	if !IsCurrent() {
		t.Fatal("page feature must be detected")
	}
}

func TestStockKeysSorted(t *testing.T) {
	keys := StockKeys()
	if len(keys) == 0 {
		t.Fatal("stock keys must not be empty")
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Fatalf("stock keys must be sorted: %v", keys)
		}
	}
}

func TestReadRestockSeconds(t *testing.T) {
	features := FeatureLib()
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		features.Page.RefreshOcr: "3:00:00",
	}}
	setupTest(t, libcolor.HitFeatures(features.Page.Feature), eng)
	sec, raw, ok := ReadRestockSeconds()
	if !ok || sec != 3*3600 {
		t.Fatalf("restock=%d raw=%q ok=%v", sec, raw, ok)
	}

	eng2 := &fakeOcr{byRect: map[image.Rectangle]string{
		features.Page.RefreshOcr: "免费刷新",
	}}
	setupTest(t, libcolor.HitFeatures(features.Page.Feature), eng2)
	sec, _, ok = ReadRestockSeconds()
	if !ok || sec != 0 {
		t.Fatalf("free refresh must read 0, got %d ok=%v", sec, ok)
	}
}

func TestIsFreeRefreshOcr(t *testing.T) {
	features := FeatureLib()
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		features.Page.RefreshOcr: "免费刷新",
	}}
	setupTest(t, libcolor.HitFeatures(features.Page.Feature), eng)
	if !IsFreeRefresh() {
		t.Fatal("free refresh must be detected")
	}
}

// ============ 任务入口测试 ============

func TestRunDisabledSkips(t *testing.T) {
	setupTest(t, nil, nil)
	// 默认配置 seasideMarket.enabled=false → ErrSkip。
	if err := Run(nil); err == nil {
		t.Fatal("disabled task must skip with error")
	}
}
