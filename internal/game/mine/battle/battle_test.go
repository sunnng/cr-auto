package battle

import (
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"app/internal/game/mine"
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

func TestBattleSessionCooldown(t *testing.T) {
	setupTest(t, nil, nil)
	if GetTimeUntilNext(21600) != 0 {
		t.Fatal("no record must be ready")
	}
	RecordBattle()
	remain := GetTimeUntilNext(3600)
	if remain <= 0 || remain > 3600 {
		t.Fatalf("cooldown remain=%d", remain)
	}
	Clear()
	if GetTimeUntilNext(21600) != 0 {
		t.Fatal("cleared session must be ready")
	}
}

// ============ 页面测试 ============

func TestBattlePageDetection(t *testing.T) {
	features := mine.Battle()
	setupTest(t, frameOf(fSpec(features.Feature)), nil)
	if !IsBattlePage() {
		t.Fatal("battle page feature must be detected")
	}
	setupTest(t, frameOf(), nil)
	if IsBattlePage() {
		t.Fatal("empty frame must not be battle page")
	}
}

func TestBattlePageFindQuickButton(t *testing.T) {
	features := mine.Battle()
	// 画锚点 + 偏移点。
	img := frameOf()
	paintFindDef(img, features.QuickBattleBtn, 560, 760)
	setupTest(t, img, nil)
	x, y, ok := FindQuickBattleButton()
	if !ok {
		t.Fatal("quick battle button must be found")
	}
	if x != 560 || y != 760 {
		t.Fatalf("button at (%d,%d)", x, y)
	}
}

func TestReadClockCount(t *testing.T) {
	features := mine.Battle()
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		features.QuickDialog.CountOcr: "1/12,611",
	}}
	setupTest(t, nil, eng)
	used, owned, raw, ok := ReadClockCount()
	if !ok || used != 1 || owned != 12611 || raw != "1/12,611" {
		t.Fatalf("clock=%d/%d raw=%q ok=%v", used, owned, raw, ok)
	}

	eng = &fakeOcr{byRect: map[image.Rectangle]string{
		features.QuickDialog.CountOcr: "无法解析",
	}}
	setupTest(t, nil, eng)
	_, _, raw, ok = ReadClockCount()
	if ok || raw == "" {
		t.Fatalf("unparseable clock must fail with raw: raw=%q ok=%v", raw, ok)
	}
}

func TestRecognizeSoulStoneType(t *testing.T) {
	features := mine.Battle()
	// 妖精王（史诗）特征：画锚点 + 偏移。
	img := frameOf()
	paintFindDef(img, features.SoulStones["史诗"]["妖精王"], 300, 650)
	setupTest(t, img, nil)
	if got := RecognizeSoulStoneType(map[string]bool{"妖精王": true}); got != "妖精王" {
		t.Fatalf("soul stone = %q", got)
	}
	if got := RecognizeSoulStoneType(map[string]bool{"莓果": true}); got != "" {
		t.Fatalf("missing soul stone must return empty, got %q", got)
	}
	// 未配置目标 → 空。
	if got := RecognizeSoulStoneType(nil); got != "" {
		t.Fatalf("no targets must return empty, got %q", got)
	}
}

func TestResolveTargetSoulStones(t *testing.T) {
	setupTest(t, nil, nil)
	targets := resolveTargetSoulStones()
	if !targets["妖精王"] || !targets["莓果"] || !targets["雷神武将"] {
		t.Fatalf("default targets=%v", targets)
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
