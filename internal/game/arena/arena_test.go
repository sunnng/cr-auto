package arena

import (
	"image"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"app/internal/config"
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

func TestArenaSessionBattleCounts(t *testing.T) {
	setupTest(t, nil, nil)
	d := Get()
	if TotalBattles(d) != 0 {
		t.Fatal("fresh session must have 0 battles")
	}
	Update(func(d *ArenaData) {
		d.Wins = 2
		d.Losses = 1
		d.Tickets = 5
	})
	d = Get()
	if TotalBattles(d) != 3 {
		t.Fatalf("total battles=%d want 3", TotalBattles(d))
	}
	if !IsReachMaxBattles(config.ArenaConfig{MaxBattles: 3}, d) {
		t.Fatal("3 battles must reach cap 3 (>=)")
	}
	if IsReachMaxBattles(config.ArenaConfig{MaxBattles: 0}, d) {
		t.Fatal("maxBattles=0 must mean unlimited")
	}
	Update(func(d *ArenaData) { d.Wins = 1; d.Losses = 1 })
	d = Get()
	if IsReachMaxBattles(config.ArenaConfig{MaxBattles: 3}, d) {
		t.Fatal("2 battles must not reach cap 3")
	}
}

func TestArenaSessionRefreshTimer(t *testing.T) {
	setupTest(t, nil, nil)
	if GetTimeUntilRefresh() != 0 {
		t.Fatal("no refresh record must report 0")
	}
	SetNextFreeRefreshAt(99999999999)
	if GetTimeUntilRefresh() <= 0 {
		t.Fatal("future refresh must report remain")
	}
	ClearNextFreeRefresh()
	if GetTimeUntilRefresh() != 0 {
		t.Fatal("cleared refresh must report 0")
	}
}

func TestHudTextCapFormat(t *testing.T) {
	setupTest(t, nil, nil)
	Update(func(d *ArenaData) { d.Wins = 1 })
	text := HudText(config.ArenaConfig{MaxBattles: 5}, Get())
	if !strings.Contains(text, "总1/5") {
		t.Fatalf("hud text must show 总1/5, got %q", text)
	}
	text = HudText(config.ArenaConfig{}, Get())
	if !strings.Contains(text, "总1/∞") {
		t.Fatalf("hud text must show 总1/∞, got %q", text)
	}
}

// ============ 页面测试 ============

func TestArenaPageDetection(t *testing.T) {
	features := FeatureLib()
	setupTest(t, frameOf(fSpec(features.Lobby.Feature)), nil)
	if !IsLobby() {
		t.Fatal("lobby feature must be detected")
	}
}

func TestParseBattleResult(t *testing.T) {
	cases := map[string]string{
		"胜利":    "胜利",
		"失败":    "失败",
		"平局":    "平局",
		"我们胜利了": "胜利",
		"":      "",
	}
	for in, want := range cases {
		got, ok := ParseBattleResult(in)
		if want == "" {
			if ok {
				t.Fatalf("ParseBattleResult(%q) must fail", in)
			}
			continue
		}
		if !ok || got != want {
			t.Fatalf("ParseBattleResult(%q)=%q,%v want %q", in, got, ok, want)
		}
	}
}

func TestReadRefreshCountdown(t *testing.T) {
	features := FeatureLib()
	cases := map[string]int{
		"5分30秒": 5*60 + 30,
		"3分":    3 * 60,
		"45秒":   45,
	}
	for in, want := range cases {
		eng := &fakeOcr{byRect: map[image.Rectangle]string{
			features.Lobby.RefreshOcr: in,
		}}
		setupTest(t, nil, eng)
		got, ok := ReadRefreshCountdown()
		if !ok || got != want {
			t.Fatalf("ReadRefreshCountdown(%q)=%d,%v want %d", in, got, ok, want)
		}
	}
	// 冒号格式被 keepHanAlphaNum 剔除，与 Lua 一致无法解析。
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		features.Lobby.RefreshOcr: "12:34",
	}}
	setupTest(t, nil, eng)
	if got, ok := ReadRefreshCountdown(); ok {
		t.Fatalf("colon format must not parse after filtering, got %d", got)
	}
}

// ============ 任务入口测试 ============

func TestRunDisabledSkips(t *testing.T) {
	setupTest(t, nil, nil)
	// 默认配置 arena.enabled=false → ErrSkip。
	if err := Run(nil); err == nil {
		t.Fatal("disabled task must skip with error")
	}
}
