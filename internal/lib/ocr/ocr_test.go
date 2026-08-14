package ocr

import (
	"image"
	"testing"
)

// fakeEngine 桌面测试用假引擎：按区域返回预先设定的结果串。
type fakeEngine struct {
	results map[image.Rectangle]string
	byRect  []struct {
		rect image.Rectangle
		raw  string
	}
}

func (f *fakeEngine) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
	if raw, ok := f.results[rect]; ok {
		return raw, nil
	}
	for _, e := range f.byRect {
		if e.rect == rect {
			return e.raw, nil
		}
	}
	return "", nil
}

func (f *fakeEngine) FindTapPoint(text string, rect image.Rectangle) (int, int, bool) {
	return 0, 0, false
}

func rect(x1, y1, x2, y2 int) image.Rectangle {
	return image.Rect(x1, y1, x2, y2)
}

func TestScanTextReturnType(t *testing.T) {
	SetEngine(&fakeEngine{results: map[image.Rectangle]string{
		rect(0, 0, 100, 50): "第 6 层",
	}})
	r, ok := Scan(rect(0, 0, 100, 50), LineMode, ReturnTypeText)
	if !ok || r.Text != "第 6 层" {
		t.Fatalf("unexpected result: %+v ok=%v", r, ok)
	}
}

func TestScanJSONDecodesItemsAndText(t *testing.T) {
	SetEngine(&fakeEngine{results: map[image.Rectangle]string{
		rect(10, 20, 110, 120): `[{"words":"配置","location":[[20,30],[40,30],[40,50],[20,50]]},{"words":"洋菜冻","location":[100,100]}]`,
	}})
	r, ok := Scan(rect(10, 20, 110, 120), MultiMode, ReturnTypeJSON)
	if !ok {
		t.Fatal("scan must succeed")
	}
	if r.Text != "配置洋菜冻" {
		t.Fatalf("joined text=%q", r.Text)
	}
	if len(r.Items) != 2 || r.Items[0].Words != "配置" {
		t.Fatalf("items=%+v", r.Items)
	}
}

func TestScanUninitializedEngineWarnsOnceAndFails(t *testing.T) {
	SetEngine(nil)
	if _, ok := Scan(rect(0, 0, 10, 10), LineMode, ReturnTypeText); ok {
		t.Fatal("nil engine must fail")
	}
	if _, ok := Scan(rect(0, 0, 10, 10), LineMode, ReturnTypeText); ok {
		t.Fatal("nil engine must fail twice")
	}
}

func TestScanEmptyRawFails(t *testing.T) {
	SetEngine(&fakeEngine{})
	if _, ok := Scan(rect(0, 0, 10, 10), LineMode, ReturnTypeText); ok {
		t.Fatal("empty raw must fail")
	}
}

func TestNumber(t *testing.T) {
	SetEngine(&fakeEngine{results: map[image.Rectangle]string{
		rect(0, 0, 100, 50): "6",
	}})
	n, ok := Number(rect(0, 0, 100, 50))
	if !ok || n != 6 {
		t.Fatalf("n=%d ok=%v", n, ok)
	}
	if _, ok := Number(rect(0, 0, 1, 1)); ok {
		t.Fatal("missing number must fail")
	}
}

func TestParseFraction(t *testing.T) {
	cases := map[string]struct {
		cur, max int
		ok       bool
	}{
		"2/5":      {2, 5, true},
		"3 / 4":    {3, 4, true},
		"1/12,611": {1, 12611, true}, // 逗号为千分位，对应 Lua gsub 后解析
		"已选 7/10":  {7, 10, true},
		"":         {0, 0, false},
		"没有":       {0, 0, false},
	}
	for in, want := range cases {
		cur, max, ok := ParseFraction(in)
		if cur != want.cur || max != want.max || ok != want.ok {
			t.Fatalf("ParseFraction(%q) = %d,%d,%v want %d,%d,%v", in, cur, max, ok, want.cur, want.max, want.ok)
		}
	}
}

func TestFractionFallsBackToSplitNumbers(t *testing.T) {
	SetEngine(&fakeEngine{
		results: map[image.Rectangle]string{},
		byRect: []struct {
			rect image.Rectangle
			raw  string
		}{
			{rect(0, 0, 50, 50), "3"},
			{rect(50, 0, 100, 50), "5"},
		},
	})
	// 整体区域无文本，左右两半各识别数字。
	cur, max, raw, ok := Fraction(rect(0, 0, 100, 50))
	if !ok || cur != 3 || max != 5 {
		t.Fatalf("Fraction = %d,%d,%q,%v", cur, max, raw, ok)
	}
}

func TestHasAndFind(t *testing.T) {
	SetEngine(&fakeEngine{results: map[image.Rectangle]string{
		rect(0, 0, 200, 100): `[{"words":"配置","location":[[20,30],[40,30],[40,50],[20,50]]}]`,
	}})
	if !Has("配置", rect(0, 0, 200, 100)) {
		t.Fatal("Has must find 配置")
	}
	if Has("不存在", rect(0, 0, 200, 100)) {
		t.Fatal("Has must not find missing text")
	}
	x, y, ok := Find("配置", rect(0, 0, 200, 100))
	if !ok || x != 30 || y != 40 {
		t.Fatalf("Find = %d,%d,%v", x, y, ok)
	}
}
