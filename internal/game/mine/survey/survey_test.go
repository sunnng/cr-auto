package survey

import (
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"app/internal/config"
	"app/internal/game/kingdom"
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

// frameOf 生成包含指定特征点（"x|y|rrggbb-…"）的帧。
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

// setupTest 注入假帧、触控、OCR 与存储；返回触控记录器。
func setupTest(t *testing.T, hits libcolor.Screen) *touchRecorder {
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
	ocr.SetEngine(&fakeOcrEngine{})
	t.Cleanup(func() {
		libcolor.SetScreen(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
		store.SetDefault(nil)
		ocr.SetEngine(nil)
	})
	return rec
}

type fakeOcrEngine struct{}

func (f *fakeOcrEngine) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
	return "", nil
}

func (f *fakeOcrEngine) FindTapPoint(text string, rect image.Rectangle) (int, int, bool) {
	return 0, 0, false
}

func fSpec(f vision.Feature) string { return f.Points }

// ============ 会话测试 ============

func TestSessionFarWaitLifecycle(t *testing.T) {
	setupTest(t, nil)
	if ok, _ := CheckFarWait(); !ok {
		t.Fatal("no record must be ready")
	}
	EnterFarWait(600)
	ok, remain := CheckFarWait()
	if ok || remain <= 0 || remain > 600 {
		t.Fatalf("must be waiting: ok=%v remain=%d", ok, remain)
	}
	if RestoreProgress() <= 0 {
		t.Fatal("restoreProgress must report remain")
	}
	Clear()
	if ok, _ := CheckFarWait(); !ok {
		t.Fatal("cleared session must be ready again")
	}
}

func TestSessionExpiredRecordReady(t *testing.T) {
	setupTest(t, nil)
	// 直接写入已到期的记录。
	store.Default().Set("mine_venture_session", map[string]any{"farWaitUntil": float64(1)})
	if ok, remain := CheckFarWait(); !ok || remain != 0 {
		t.Fatalf("expired record must be ready: ok=%v remain=%d", ok, remain)
	}
	if RestoreProgress() != 0 {
		t.Fatal("expired record must report 0 remain")
	}
}

// ============ 页面测试 ============

func TestPageDetectsVentureDomain(t *testing.T) {
	v := mine.MineVenture()
	for _, f := range []vision.Feature{v.Setup.Feature, v.Ready.Feature, v.Running.Feature, v.Settle.Feature} {
		setupTest(t, libcolor.HitFeatures(f))
		if !IsMineVentureDomain() {
			t.Fatalf("must detect venture domain for %s", f.Points)
		}
	}
	setupTest(t, nil)
	if IsMineVentureDomain() {
		t.Fatal("empty frame must not be venture domain")
	}
}

func TestPageRunningAndStop(t *testing.T) {
	running := mine.MineVenture().Running
	setupTest(t, libcolor.HitFeatures(running.Feature))
	if !IsRunning() {
		t.Fatal("running page must be detected")
	}
}

// ============ 初始状态测试 ============

func TestResolveInitialStateByPage(t *testing.T) {
	cfg := mineCfgForTest()

	setupTest(t, libcolor.HitFeatures(mine.MineHome().Feature))
	state, remain, run := resolveInitialState(cfg)
	if !run || state != "navigate" || remain != 0 {
		t.Fatalf("mine home → navigate: state=%q remain=%d run=%v", state, remain, run)
	}

	setupTest(t, libcolor.HitFeatures(kingdom.Home().Feature))
	state, _, run = resolveInitialState(cfg)
	if !run || state != "navigate" {
		t.Fatalf("kingdom home → navigate: state=%q run=%v", state, run)
	}

	setupTest(t, libcolor.HitFeatures(mine.MineVenture().Ready.Feature))
	state, _, run = resolveInitialState(cfg)
	if !run || state != "prepare" {
		t.Fatalf("venture domain → prepare: state=%q run=%v", state, run)
	}

	setupTest(t, nil)
	state, _, run = resolveInitialState(cfg)
	if !run || state != "detect" {
		t.Fatalf("unknown page → detect: state=%q run=%v", state, run)
	}
}

func TestResolveInitialStateFarWaitBlocks(t *testing.T) {
	setupTest(t, libcolor.HitFeatures(mine.MineHome().Feature))
	EnterFarWait(600)
	state, remain, run := resolveInitialState(mineCfgForTest())
	if run || state != "" || remain <= 0 {
		t.Fatalf("far wait must block: state=%q remain=%d run=%v", state, remain, run)
	}
}

func mineCfgForTest() config.MineConfig {
	return config.Static.User.Mine
}
