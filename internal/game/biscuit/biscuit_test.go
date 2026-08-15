package biscuit

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
)

// ============ 词条库测试 ============

func TestEntriesLib(t *testing.T) {
	if len(Names()) != len(entries) {
		t.Fatalf("names must cover all entries")
	}
	e, ok := Find("攻击力")
	if !ok || e.MinValue != 3 || e.MaxValue != 7.5 {
		t.Fatalf("find 攻击力 = %+v,%v", e, ok)
	}
	if _, ok := Find("不存在"); ok {
		t.Fatal("unknown entry must not be found")
	}
	min, max, ok := ValueBounds("会心")
	if !ok || min != 3 || max != 7 {
		t.Fatalf("valueBounds 会心 = %v,%v,%v", min, max, ok)
	}
	min, max, ok = SumBounds("攻击力", 2)
	if !ok || min != 6 || max != 15 {
		t.Fatalf("sumBounds 攻击力x2 = %v,%v,%v", min, max, ok)
	}
	if hint := RangeHint("生命值"); hint != "范围 3%~15%" {
		t.Fatalf("rangeHint 生命值 = %q", hint)
	}
	if RangeHint("") != "" {
		t.Fatal("empty rangeHint must be empty")
	}
}

// ============ 词条解析测试 ============

func TestExtractNumber(t *testing.T) {
	value, name, ok := extractNumber("生命值3")
	if !ok || value != 3 || name != "生命值" {
		t.Fatalf("extract 生命值3 = %v,%q,%v", value, name, ok)
	}
	value, name, ok = extractNumber("会心10.8")
	if !ok || value != 10.8 || name != "会心" {
		t.Fatalf("extract 会心10.8 = %v,%q,%v", value, name, ok)
	}
	if _, _, ok := extractNumber(""); ok {
		t.Fatal("empty must fail")
	}
}

func TestParseRaw(t *testing.T) {
	result := parseRaw("攻击力3%生命值7.9%会心3.7%")
	if len(result) != 3 {
		t.Fatalf("parseRaw len=%d", len(result))
	}
	if result[0].Name != "攻击力" || result[0].Value != 3 {
		t.Fatalf("first=%+v", result[0])
	}
	if result[1].Name != "生命值" || result[1].Value != 7.9 {
		t.Fatalf("second=%+v", result[1])
	}
	if result[2].Name != "会心" || result[2].Value != 3.7 {
		t.Fatalf("third=%+v", result[2])
	}
	if len(parseRaw("")) != 0 {
		t.Fatal("empty raw must parse empty")
	}
}

// ============ 规则检查测试 ============

func TestCheckSlots(t *testing.T) {
	effects := []Effect{
		{Name: "攻击力", Value: 5},
		{Name: "会心", Value: 6},
	}
	targets := []config.BiscuitTarget{
		{Enabled: true, Name: "攻击力", MinPercent: 4},
		{Enabled: true, Name: "会心", MinPercent: 5},
	}
	ok, msg := checkSlots(effects, targets)
	if !ok || msg != "毕业" {
		t.Fatalf("slots must graduate: %v %q", ok, msg)
	}

	// 阈值更高：不满足。
	targets[0].MinPercent = 6
	if ok, _ := checkSlots(effects, targets); ok {
		t.Fatal("higher threshold must fail")
	}

	// 同名词条复用检查：两个攻击力规则只能匹配不同词条。
	effects = []Effect{
		{Name: "攻击力", Value: 5},
		{Name: "攻击力", Value: 6},
	}
	targets = []config.BiscuitTarget{
		{Enabled: true, Name: "攻击力", MinPercent: 5},
		{Enabled: true, Name: "攻击力", MinPercent: 5},
	}
	if ok, _ := checkSlots(effects, targets); !ok {
		t.Fatal("two attack rules must match two distinct slots")
	}

	// 无启用规则。
	if ok, msg := checkSlots(effects, nil); ok || msg != "无槽位规则" {
		t.Fatalf("no targets: %v %q", ok, msg)
	}
}

func TestCheckSums(t *testing.T) {
	effects := []Effect{
		{Name: "攻击力", Value: 5},
		{Name: "攻击力", Value: 4},
		{Name: "攻击力", Value: 3},
	}
	rules := []config.BiscuitSumRule{
		{Enabled: true, Name: "攻击力", Count: 2, MinSum: 8},
	}
	ok, _ := checkSums(effects, rules)
	if !ok {
		t.Fatal("top-2 sum 5+4=9 >= 8 must pass")
	}
	rules[0].MinSum = 10
	if ok, _ := checkSums(effects, rules); ok {
		t.Fatal("top-2 sum 9 < 10 must fail")
	}
	if ok, msg := checkSums(effects, nil); ok || msg != "未配置总和规则" {
		t.Fatalf("no rules: %v %q", ok, msg)
	}
}

func TestCheckEither(t *testing.T) {
	effects := []Effect{{Name: "攻击力", Value: 12}}
	// 槽位不满足但总和满足。
	ok, msg := check(effects,
		[]config.BiscuitTarget{{Enabled: true, Name: "会心", MinPercent: 5}},
		[]config.BiscuitSumRule{{Enabled: true, Name: "攻击力", Count: 1, MinSum: 10}})
	if !ok {
		t.Fatalf("sum fallback must pass: %q", msg)
	}
	// 两者都不满足。
	ok, _ = check(effects,
		[]config.BiscuitTarget{{Enabled: true, Name: "会心", MinPercent: 5}},
		[]config.BiscuitSumRule{{Enabled: true, Name: "攻击力", Count: 1, MinSum: 99}})
	if ok {
		t.Fatal("both must fail")
	}
}

// ============ 任务入口测试 ============

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

func TestRunGraduatesAndDisables(t *testing.T) {
	setupTest(t, nil, nil)
	// OCR 返回符合毕业条件的词条：默认配置冷却时间≥5、会心≥6 或攻击力总和≥11。
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		effectOcrRect: "冷却时间5%会心6%攻击力3%",
	}}
	setupTest(t, nil, eng)

	if err := Run(nil); err != nil {
		t.Fatalf("Run must not error, got %v", err)
	}
	// 毕业后 enabled 应被写回 false（user_config.biscuit.Enabled）。
	raw, err := store.Default().Get("user_config", nil)
	if err != nil {
		t.Fatalf("read user_config: %v", err)
	}
	var sections map[string]map[string]any
	if err := store.Decode(raw, &sections); err != nil {
		t.Fatalf("decode user_config: %v", err)
	}
	biscuitSection := sections["biscuit"]
	enabled, ok := biscuitSection["Enabled"]
	if !ok || enabled != false {
		t.Fatalf("biscuit Enabled must be false after graduation: %v", biscuitSection)
	}
}
