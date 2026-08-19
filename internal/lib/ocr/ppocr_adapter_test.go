package ocr

import "testing"

func TestFormatScanTextJoinsLabels(t *testing.T) {
	raw := FormatScan([]RegionText{{Words: "配置", X: 10, Y: 20, W: 40, H: 16}}, ReturnTypeText)
	if raw != "配置" {
		t.Fatalf("text=%q", raw)
	}
}

func TestFormatScanJSONIncludesLocation(t *testing.T) {
	raw := FormatScan([]RegionText{{Words: "配置", X: 10, Y: 20, W: 40, H: 16}}, ReturnTypeJSON)
	items, text := decode(raw)
	if text != "配置" || len(items) != 1 {
		t.Fatalf("items=%+v text=%q", items, text)
	}
	if _, _, ok := localCenter(items[0].Location); !ok {
		t.Fatal("json location must decode to a center")
	}
}

func TestFindCenterMatchesSubstring(t *testing.T) {
	x, y, ok := FindCenter([]RegionText{{Words: "王国竞技场", X: 100, Y: 200, W: 80, H: 20}}, "竞技场")
	if !ok || x != 140 || y != 210 {
		t.Fatalf("center=(%d,%d) ok=%v", x, y, ok)
	}
}
