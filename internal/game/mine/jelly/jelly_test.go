package jelly

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

func TestSessionWaitLifecycle(t *testing.T) {
	setupTest(t, nil, nil)
	if ok, _ := CheckReady(); !ok {
		t.Fatal("no record must be ready")
	}
	EnterWait(3600)
	ok, remain := CheckReady()
	if ok || remain <= 0 || remain > 3600 {
		t.Fatalf("must be waiting: ok=%v remain=%d", ok, remain)
	}
	if RestoreProgress() <= 0 {
		t.Fatal("restoreProgress must report remain")
	}
	Clear()
	if ok, _ := CheckReady(); !ok {
		t.Fatal("cleared session must be ready again")
	}
}

// ============ 页面测试 ============

func TestJellyPageDetection(t *testing.T) {
	features := mine.Jelly()
	setupTest(t, frameOf(fSpec(features.Feature)), nil)
	if !IsJellyPage() {
		t.Fatal("jelly page feature must be detected")
	}
	setupTest(t, frameOf(fSpec(features.Config.Feature)), nil)
	if !IsConfigPage() {
		t.Fatal("config page feature must be detected")
	}
	setupTest(t, frameOf(fSpec(features.ClaimAllFeature)), nil)
	if !CanClaimAll() {
		t.Fatal("claim-all feature must be detected")
	}
}

func TestFindConfigBtnByOcr(t *testing.T) {
	features := mine.Jelly()
	// location 为裁剪图相对坐标，Find 返回绝对坐标（相对 + 区域原点）。
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		features.OcrRegion: `[{"words":"配置","location":[[400,620],[430,620],[430,640],[400,640]]}]`,
	}}
	setupTest(t, nil, eng)
	x, y, ok := FindConfigBtn()
	if !ok || x != 689 || y != 1216 {
		t.Fatalf("config button at (%d,%d) ok=%v", x, y, ok)
	}
}

func TestParseRemainTimeText(t *testing.T) {
	cases := map[string]int{
		"2天3小时4分钟5秒": 2*86400 + 3*3600 + 4*60 + 5,
		"3小时30分钟":    3*3600 + 30*60,
		"45分钟":       45 * 60,
		"50秒":        50,
	}
	for in, want := range cases {
		got, ok := ParseRemainTimeText(in)
		if !ok || got != want {
			t.Fatalf("ParseRemainTimeText(%q) = %d,%v want %d", in, got, ok, want)
		}
	}
	if _, ok := ParseRemainTimeText("无法识别"); ok {
		t.Fatal("unparseable text must fail")
	}
}

func TestReadRemainTimeFromItems(t *testing.T) {
	features := mine.Jelly()
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		features.OcrRegion: `[{"words":"解除完成","location":[0,0]},{"words":"剩余 1小时30分钟","location":[0,0]}]`,
	}}
	setupTest(t, nil, eng)
	remain, ok := ReadRemainTime()
	if !ok || remain != 5400 {
		t.Fatalf("remain=%d ok=%v", remain, ok)
	}
}

// ============ 任务入口测试 ============

func TestRunDisabledSkips(t *testing.T) {
	setupTest(t, nil, nil)
	// 默认配置 jellyEnabled=false → ErrSkip。
	if err := Run(nil); err == nil {
		t.Fatal("disabled task must skip with error")
	}
}
