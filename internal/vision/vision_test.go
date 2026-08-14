package vision

import (
	"image"
	"image/color"
	"testing"
)

// newFrame builds a width*height frame filled with base, then paints
// points from the override map (x*1000+y -> color).
func newFrame(width, height int, base color.NRGBA, overrides map[int]color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, base)
		}
	}
	for key, c := range overrides {
		x := key / 1000
		y := key % 1000
		img.SetNRGBA(x, y, c)
	}
	return img
}

func TestParseFeaturePoints(t *testing.T) {
	f := Feature{Points: "1380|60|f7e5cb-101010,59|323|b3001b-101010,96|825|fbed78-101010", Sim: 0.9}
	pts, err := parsePoints(f.Points)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 3 {
		t.Fatalf("expected 3 points, got %d", len(pts))
	}
	want := []Point{
		{X: 1380, Y: 60, R: 0xf7, G: 0xe5, B: 0xcb, Tol: 0x10},
		{X: 59, Y: 323, R: 0xb3, G: 0x00, B: 0x1b, Tol: 0x10},
		{X: 96, Y: 825, R: 0xfb, G: 0xed, B: 0x78, Tol: 0x10},
	}
	for i, w := range want {
		if pts[i] != w {
			t.Fatalf("point %d = %+v, want %+v", i, pts[i], w)
		}
	}
}

func TestParseFeaturePointWithoutToleranceDefaultsToDefaultTolerance(t *testing.T) {
	pts, err := parsePoints("10|20|ff0000")
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 1 || pts[0].Tol != defaultTolerance {
		t.Fatalf("unexpected points: %+v", pts)
	}
}

func TestParseFeatureRejectsMalformedPoint(t *testing.T) {
	for _, bad := range []string{"1|2", "1|2|gggggg-101010", "x|2|ff0000-101010", "1|2|ff0000-101010,9"} {
		if _, err := parsePoints(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestMatchExact(t *testing.T) {
	f := Feature{Points: "1|1|ff0000-000000,2|2|00ff00-000000"}
	img := newFrame(4, 4, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xff}, 2002: {G: 0xff},
	})
	if !Match(img, f) {
		t.Fatal("expected exact multi-point match")
	}
}

func TestMatchWithinTolerance(t *testing.T) {
	f := Feature{Points: "1|1|ff0000-101010"}
	img := newFrame(4, 4, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xf2, G: 0x0a, B: 0x08},
	})
	if !Match(img, f) {
		t.Fatal("expected match within per-channel tolerance")
	}
}

func TestMatchOutsideTolerance(t *testing.T) {
	f := Feature{Points: "1|1|ff0000-101010"}
	img := newFrame(4, 4, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xe0},
	})
	if Match(img, f) {
		t.Fatal("expected no match beyond tolerance")
	}
}

func TestMatchRequiresAllPointsByDefault(t *testing.T) {
	f := Feature{Points: "1|1|ff0000-000000,2|2|00ff00-000000"}
	img := newFrame(4, 4, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xff},
	})
	if Match(img, f) {
		t.Fatal("missing one of two points must not match with sim=1")
	}
}

func TestMatchSimAllowsFractionOfPoints(t *testing.T) {
	f := Feature{Points: "1|1|ff0000-000000,2|2|00ff00-000000,3|3|0000ff-000000,4|4|ffffff-000000", Sim: 0.7}
	img := newFrame(6, 6, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xff}, 2002: {G: 0xff}, 3003: {B: 0xff},
	})
	if !Match(img, f) {
		t.Fatal("three of four points must satisfy sim=0.7")
	}
}

func TestMatchSimFailsBelowFraction(t *testing.T) {
	f := Feature{Points: "1|1|ff0000-000000,2|2|00ff00-000000,3|3|0000ff-000000,4|4|ffffff-000000", Sim: 0.9}
	img := newFrame(6, 6, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xff}, 2002: {G: 0xff}, 3003: {B: 0xff},
	})
	if Match(img, f) {
		t.Fatal("three of four points must fail sim=0.9")
	}
}

func TestMatchOutOfBoundsPointNeverMatches(t *testing.T) {
	f := Feature{Points: "99|99|ff0000-000000"}
	img := newFrame(4, 4, color.NRGBA{0, 0, 0, 255}, nil)
	if Match(img, f) {
		t.Fatal("out-of-bounds point must not match")
	}
}

func TestMatchNilFrame(t *testing.T) {
	f := Feature{Points: "1|1|ff0000-000000"}
	if Match(nil, f) {
		t.Fatal("nil frame must not match")
	}
}

func TestMatchAnyReturnsFirstHit(t *testing.T) {
	img := newFrame(4, 4, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		2002: {G: 0xff},
	})
	features := []Feature{
		{Points: "1|1|ff0000-000000"},
		{Points: "2|2|00ff00-000000"},
	}
	ok, which := MatchAny(img, features)
	if !ok || which != 1 {
		t.Fatalf("expected second feature to match, got ok=%v which=%d", ok, which)
	}
}

func TestMatchAnyNoneHit(t *testing.T) {
	img := newFrame(4, 4, color.NRGBA{0, 0, 0, 255}, nil)
	ok, which := MatchAny(img, []Feature{{Points: "1|1|ff0000-000000"}})
	if ok || which != -1 {
		t.Fatalf("expected no match, got ok=%v which=%d", ok, which)
	}
}

func TestFindMultiColorHit(t *testing.T) {
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		2002: {R: 0xff}, // anchor
		3003: {G: 0xff}, // offset (1,1)
		4012: {B: 0xff}, // offset (2,10) -> (4,12) out of bounds, must not matter
	})
	def := FindDef{
		Region:       image.Rect(0, 0, 10, 10),
		FirstColor:   "ff0000-000000",
		OffsetColors: "1,1,00ff00-000000",
		Sim:          1,
	}
	x, y, ok := FindMultiColor(img, def)
	if !ok || x != 2 || y != 2 {
		t.Fatalf("expected anchor (2,2), got (%d,%d) ok=%v", x, y, ok)
	}
}

func TestFindMultiColorRequiresOffsets(t *testing.T) {
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		2002: {R: 0xff},
	})
	def := FindDef{
		Region:       image.Rect(0, 0, 10, 10),
		FirstColor:   "ff0000-000000",
		OffsetColors: "1,1,00ff00-000000",
		Sim:          1,
	}
	if _, _, ok := FindMultiColor(img, def); ok {
		t.Fatal("anchor without matching offset must not be found")
	}
}

func TestFindMultiColorAnyFirstColor(t *testing.T) {
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		2002: {G: 0xff},
	})
	def := FindDef{
		Region:       image.Rect(0, 0, 10, 10),
		FirstColor:   "ff0000-000000|00ff00-000000",
		OffsetColors: "1,1,00ff00-000000",
		Sim:          1,
	}
	// Offset (3,3) must be the plain background; only the first-color list
	// decides the anchor. Rebuild with the offset satisfied.
	img2 := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		2002: {G: 0xff}, 3003: {G: 0xff},
	})
	x, y, ok := FindMultiColor(img2, def)
	if !ok || x != 2 || y != 2 {
		t.Fatalf("expected anchor via alternate first color, got (%d,%d) ok=%v", x, y, ok)
	}
	_ = img
}

func TestFindMultiColorOffsetColorAlternates(t *testing.T) {
	// 相对色点支持 "|" 分隔的多候选：偏移 (1,1) 命中 绿 或 蓝 都算。
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		2002: {R: 0xff}, // 命中点
		3003: {B: 0xff}, // 偏移 (1,1) = 蓝
	})
	def := FindDef{
		Region:       image.Rect(0, 0, 10, 10),
		FirstColor:   "ff0000-000000",
		OffsetColors: "1,1,00ff00-000000|0000ff-000000",
		Sim:          1,
	}
	x, y, ok := FindMultiColor(img, def)
	if !ok || x != 2 || y != 2 {
		t.Fatalf("expected anchor via alternate offset color, got (%d,%d) ok=%v", x, y, ok)
	}
}

func TestFindMultiColorScanDirection(t *testing.T) {
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xff}, 8008: {G: 0xff},
	})
	mkDef := func() FindDef {
		return FindDef{
			Region:       image.Rect(0, 0, 10, 10),
			FirstColor:   "ff0000-000000|00ff00-000000",
			OffsetColors: "",
			Sim:          1,
		}
	}
	def := mkDef()
	def.Dir = 0
	x, y, ok := FindMultiColor(img, def)
	if !ok || x != 1 || y != 1 {
		t.Fatalf("dir 0 must find top-left first, got (%d,%d)", x, y)
	}
	def.Dir = 3
	x, y, ok = FindMultiColor(img, def)
	if !ok || x != 8 || y != 8 {
		t.Fatalf("dir 3 must find bottom-right first, got (%d,%d)", x, y)
	}
}

func TestFindMultiColorLuaPipeSeparatedOffsets(t *testing.T) {
	// Lua 特征库的 findMultiColorT offsetColors 用 "|" 分隔三元组
	// （"dx|dy|color|dx|dy|color|..."），vision 需兼容（逗号格式优先）。
	img := newFrame(200, 200, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		100100: {R: 0xff}, // anchor (100,100)
		96074:  {G: 0xff}, // offset (-4,-26) -> (96,74)
		101101: {B: 0xff}, // offset (1,1) -> (101,101)
	})
	def := FindDef{
		Region:       image.Rect(0, 0, 200, 200),
		FirstColor:   "ff0000-101010",
		OffsetColors: "-4|-26|00ff00-101010|1|1|0000ff-101010",
		Sim:          1,
	}
	x, y, ok := FindMultiColor(img, def)
	if !ok || x != 100 || y != 100 {
		t.Fatalf("expected anchor (100,100) via pipe offsets, got (%d,%d) ok=%v", x, y, ok)
	}
}

func TestFindMultiColorPipeOffsetsRequireAll(t *testing.T) {
	img := newFrame(200, 200, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		100100: {R: 0xff}, // anchor only, no offsets
	})
	def := FindDef{
		Region:       image.Rect(0, 0, 200, 200),
		FirstColor:   "ff0000-101010",
		OffsetColors: "-4|-26|00ff00-101010|1|1|0000ff-101010",
		Sim:          1,
	}
	if _, _, ok := FindMultiColor(img, def); ok {
		t.Fatal("pipe offsets must be required for the anchor")
	}
}

func TestFindMultiColorMiss(t *testing.T) {
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, nil)
	def := FindDef{Region: image.Rect(0, 0, 10, 10), FirstColor: "ff0000-000000", Sim: 1}
	if _, _, ok := FindMultiColor(img, def); ok {
		t.Fatal("empty frame must not match")
	}
}

func TestFindMultiColorRespectsRegion(t *testing.T) {
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		1001: {R: 0xff}, 4004: {R: 0xff},
	})
	def := FindDef{Region: image.Rect(2, 2, 10, 10), FirstColor: "ff0000-000000", Sim: 1}
	x, y, ok := FindMultiColor(img, def)
	if !ok || x != 4 || y != 4 {
		t.Fatalf("region must exclude anchor (1,1), got (%d,%d)", x, y)
	}
}

func TestFindMultiColorAllCollectsEveryAnchor(t *testing.T) {
	img := newFrame(10, 10, color.NRGBA{0, 0, 0, 255}, map[int]color.NRGBA{
		2002: {R: 0xff}, 5005: {R: 0xff},
	})
	def := FindDef{Region: image.Rect(0, 0, 10, 10), FirstColor: "ff0000-000000", Sim: 1}
	points := FindMultiColorAll(img, def)
	if len(points) != 2 {
		t.Fatalf("expected two anchors, got %d", len(points))
	}
	if points[0] != (image.Point{X: 2, Y: 2}) || points[1] != (image.Point{X: 5, Y: 5}) {
		t.Fatalf("unexpected anchors: %+v", points)
	}
}
