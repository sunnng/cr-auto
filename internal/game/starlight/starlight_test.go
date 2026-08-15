package starlight

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
	setupTest(t, frameOf(fSpec(features.Home.Feature)), nil)
	if !IsHomePage() {
		t.Fatal("home feature must be detected")
	}
	setupTest(t, frameOf(fSpec(features.Manual.Feature)), nil)
	if !IsManualPage() {
		t.Fatal("manual feature must be detected")
	}
	setupTest(t, frameOf(fSpec(features.Vanilla.Feature)), nil)
	if !IsVanillaIslandPage() {
		t.Fatal("vanilla feature must be detected")
	}
	setupTest(t, frameOf(fSpec(features.Task.Feature)), nil)
	if !IsTaskPage() {
		t.Fatal("task feature must be detected")
	}
}

func TestFindClaimableBtn(t *testing.T) {
	features := FeatureLib()
	def := features.Task.ClaimableBtn
	setupTest(t, frameOf(frameForFindDef(def)), nil)
	x, y, ok := FindClaimableBtn()
	if !ok || x != def.Region.Min.X || y != def.Region.Min.Y {
		t.Fatalf("claimable btn at (%d,%d) ok=%v want (%d,%d)", x, y, ok, def.Region.Min.X, def.Region.Min.Y)
	}
}

// frameForFindDef 生成含找色定义锚点+偏移色点的帧（管道分隔偏移串）。
func frameForFindDef(def vision.FindDef) string {
	first := strings.Split(def.FirstColor, "|")[0]
	hex := first
	if dash := strings.LastIndex(hex, "-"); dash >= 0 {
		hex = hex[:dash]
	}
	var out []string
	out = append(out, strconv.Itoa(def.Region.Min.X)+"|"+strconv.Itoa(def.Region.Min.Y)+"|"+hex)
	// 偏移串形如 "dx|dy|color|dx|dy|color..."。
	fields := strings.Split(def.OffsetColors, "|")
	for i := 0; i+2 < len(fields); i += 3 {
		dx, _ := strconv.Atoi(fields[i])
		dy, _ := strconv.Atoi(fields[i+1])
		colorHex := fields[i+2]
		if dash := strings.LastIndex(colorHex, "-"); dash >= 0 {
			colorHex = colorHex[:dash]
		}
		out = append(out, strconv.Itoa(def.Region.Min.X+dx)+"|"+strconv.Itoa(def.Region.Min.Y+dy)+"|"+colorHex)
	}
	return strings.Join(out, ",")
}
