package mining

import (
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"app/internal/game/kingdom"
	"app/internal/game/mine"
	libcolor "app/internal/lib/color"
	"app/internal/lib/ocr"
	"app/internal/lib/store"
	"app/internal/lib/touch"
	"app/internal/vision"
)

// ============ 测试辅助（与 survey 包同款） ============

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

// fakeOcr 可编程 OCR 引擎。
type fakeOcr struct {
	byRect map[image.Rectangle]string
}

func (f *fakeOcr) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
	if raw, ok := f.byRect[rect]; ok {
		return raw, nil
	}
	return "", nil
}

func (f *fakeOcr) FindTapPoint(text string, rect image.Rectangle) (int, int, bool) {
	return 0, 0, false
}

func setupTest(t *testing.T, frame *image.NRGBA, eng *fakeOcr) *touchRecorder {
	t.Helper()
	rec := &touchRecorder{}
	libcolor.SetFrameSource(&fakeFrame{img: frame})
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
		libcolor.SetFrameSource(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
		store.SetDefault(nil)
		ocr.SetEngine(nil)
	})
	return rec
}

func fSpec(f vision.Feature) string { return f.Points }

// ============ 会话测试 ============

func TestSessionBusyLifecycle(t *testing.T) {
	setupTest(t, nil, nil)
	if ok, _ := CheckReady(); !ok {
		t.Fatal("no record must be ready")
	}
	EnterBusyWait(1200)
	ok, remain := CheckReady()
	if ok || remain <= 0 || remain > 1200 {
		t.Fatalf("must be busy: ok=%v remain=%d", ok, remain)
	}
	if RestoreProgress() <= 0 {
		t.Fatal("restoreProgress must report remain")
	}
	Clear()
	if ok, _ := CheckReady(); !ok {
		t.Fatal("cleared session must be ready again")
	}
}

func TestSessionExpiredBusyReady(t *testing.T) {
	setupTest(t, nil, nil)
	store.Default().Set("mine_mining_session", map[string]any{"allBusyUntil": float64(1)})
	if ok, remain := CheckReady(); !ok || remain != 0 {
		t.Fatalf("expired busy must be ready: ok=%v remain=%d", ok, remain)
	}
}

// ============ 页面测试 ============

func TestPageDetection(t *testing.T) {
	features := mine.Mining()

	setupTest(t, frameOf(fSpec(features.Page.Feature)), nil)
	if !IsMiningPage() {
		t.Fatal("mining page feature must be detected")
	}

	setupTest(t, frameOf(fSpec(features.SetupFeature)), nil)
	if !IsSetup() {
		t.Fatal("setup feature must be detected")
	}

	setupTest(t, frameOf(fSpec(features.SetupReadyFeature)), nil)
	if !IsSetupReady() {
		t.Fatal("setupReady feature must be detected")
	}
}

func TestPageCompletedTaskFindAndTap(t *testing.T) {
	// 完成标记特征：画锚点 + 全部相对色点。
	completed := mine.Mining().CompletedTask
	img := frameOf()
	paintFindDef(img, completed, 100, 150)
	setupTest(t, img, nil)
	if !HasCompletedTask() {
		t.Fatal("completed task must be found")
	}
	rec := setupTest(t, img, nil)
	if !TapCompletedSlot() {
		t.Fatal("tapCompletedSlot must succeed")
	}
	taps := rec.taps()
	if len(taps) != 1 || taps[0] != (image.Point{X: 100, Y: 150}) {
		t.Fatalf("completed slot must be tapped, got %+v", taps)
	}
}

// paintFindDef 在 img 上绘制找色定义的锚点与全部相对色点。
func paintFindDef(img *image.NRGBA, def vision.FindDef, ax, ay int) {
	paint := func(spec string, x, y int) {
		hex := spec
		if dash := strings.LastIndex(hex, "-"); dash >= 0 {
			hex = hex[:dash]
		}
		rgb, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return
		}
		img.SetNRGBA(x, y, color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 0xff})
	}
	paint(def.FirstColor, ax, ay)
	parts := strings.Split(def.OffsetColors, "|")
	for i := 0; i+2 < len(parts); i += 3 {
		dx, _ := strconv.Atoi(parts[i])
		dy, _ := strconv.Atoi(parts[i+1])
		paint(parts[i+2], ax+dx, ay+dy)
	}
}

func TestPageFreeSlotDetection(t *testing.T) {
	// freeLocationFeature 首色 c67654，锚点需满足全部相对色点（简化：只画锚点不画偏移 → 不命中）。
	setupTest(t, frameOf("100|200|c67654-000000"), nil)
	if HasFreeSlot() {
		t.Fatal("free slot without offsets must not be found")
	}
}

func TestPageRewardPageByOcr(t *testing.T) {
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		mine.Mining().RewardPage.TitleOcr: `[{"words":"获得开采奖励","location":[0,0]}]`,
	}}
	setupTest(t, nil, eng)
	if !IsRewardPage() {
		t.Fatal("reward page title must be detected via OCR")
	}
	setupTest(t, nil, &fakeOcr{byRect: map[image.Rectangle]string{}})
	if IsRewardPage() {
		t.Fatal("reward page must not be detected without title")
	}
}

func TestPageReadChooseQuota(t *testing.T) {
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		mine.Mining().CanChooseNum: "3/6",
	}}
	setupTest(t, nil, eng)
	cur, max, raw, ok := ReadChooseQuota()
	if !ok || cur != 3 || max != 6 || raw != "3/6" {
		t.Fatalf("quota=%d/%d raw=%q ok=%v", cur, max, raw, ok)
	}
}

// ============ 任务逻辑测试 ============

func TestResolveCardPriorityUsesConfigAndDefaults(t *testing.T) {
	setupTest(t, nil, nil)
	// 默认配置全有特征 → 顺序与 config 一致。
	got := resolveCardPriority()
	want := []string{"butterAmber", "amberFossil", "sugarOre", "purpleFossil", "emeraldFossil", "flourStone"}
	if len(got) != len(want) {
		t.Fatalf("priority=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("priority=%v want %v", got, want)
		}
	}
}

func TestReturnToKingdomFromMineHome(t *testing.T) {
	home := mine.MineHome()
	kingdomHome := kingdom.Home()
	// 点击返回后帧切换为王国首页 → Wait 立即命中。
	rec := &touchRecorder{}
	fs := &switchableFrame{get: func() *image.NRGBA {
		if len(rec.taps()) > 0 {
			return frameOf(fSpec(kingdomHome.Feature))
		}
		return frameOf(fSpec(home.Feature))
	}}
	setupTest(t, nil, nil)
	libcolor.SetFrameSource(fs)
	touch.SetPerform(touch.Perform{
		Tap:    rec.tap,
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})

	if !ReturnToKingdom() {
		t.Fatal("ReturnToKingdom must succeed from mine home")
	}
	if len(rec.taps()) != 1 {
		t.Fatalf("mine home back tap expected, got %+v", rec.taps())
	}
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
