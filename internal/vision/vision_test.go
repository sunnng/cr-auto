package vision

import (
	"image"
	"testing"
)

func TestParseFeaturePoints(t *testing.T) {
	f := Feature{Points: "1380|60|f7e5cb-101010,59|323|b3001b-101010,96|825|fbed78-101010", Sim: 0.9}
	pts, err := ParsePoints(f.Points)
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
	pts, err := ParsePoints("10|20|ff0000")
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 1 || pts[0].Tol != defaultTolerance {
		t.Fatalf("unexpected points: %+v", pts)
	}
}

func TestParseFeatureRejectsMalformedPoint(t *testing.T) {
	for _, bad := range []string{"1|2", "1|2|gggggg-101010", "x|2|ff0000-101010", "1|2|ff0000-101010,9"} {
		if _, err := ParsePoints(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestDetectsColorsReplacesPipes(t *testing.T) {
	f := Feature{Points: "1380|60|f7e5cb-101010,59|323|b3001b-101010"}
	got := DetectsColors(f)
	want := "1380,60,f7e5cb-101010,59,323,b3001b-101010"
	if got != want {
		t.Fatalf("DetectsColors=%q want %q", got, want)
	}
}

func TestDetectsColorsRejectsMalformed(t *testing.T) {
	if DetectsColors(Feature{Points: "bad"}) != "" {
		t.Fatal("malformed feature must convert to empty")
	}
	if DetectsColors(Feature{}) != "" {
		t.Fatal("empty feature must convert to empty")
	}
}

func TestFindColorsCommaOffsets(t *testing.T) {
	def := FindDef{
		Region:       image.Rect(0, 0, 10, 10),
		FirstColor:   "ff0000-000000",
		OffsetColors: "1,1,00ff00-000000",
	}
	got := FindColors(def)
	want := "ff0000-000000,1,1,00ff00-000000"
	if got != want {
		t.Fatalf("FindColors=%q want %q", got, want)
	}
}

func TestFindColorsPipeOffsets(t *testing.T) {
	def := FindDef{
		FirstColor:   "ff0000-101010",
		OffsetColors: "-4|-26|00ff00-101010|1|1|0000ff-101010",
	}
	got := FindColors(def)
	want := "ff0000-101010,-4,-26,00ff00-101010,1,1,0000ff-101010"
	if got != want {
		t.Fatalf("FindColors=%q want %q", got, want)
	}
}

func TestFindColorsFirstOnly(t *testing.T) {
	got := FindColors(FindDef{FirstColor: "ff0000-000000"})
	if got != "ff0000-000000" {
		t.Fatalf("FindColors=%q", got)
	}
}

func TestFindColorsRejectsBad(t *testing.T) {
	if FindColors(FindDef{}) != "" {
		t.Fatal("missing first color")
	}
	if FindColors(FindDef{FirstColor: "ff0000", OffsetColors: "1,1"}) != "" {
		t.Fatal("incomplete offset triplet")
	}
}

func TestCmpSpec(t *testing.T) {
	if got := CmpSpec(Point{R: 0xff, G: 0x00, B: 0x00}); got != "ff0000" {
		t.Fatalf("zero tol: %q", got)
	}
	if got := CmpSpec(Point{R: 0xf7, G: 0xe5, B: 0xcb, Tol: 0x10}); got != "f7e5cb-101010" {
		t.Fatalf("with tol: %q", got)
	}
}

func TestParseColorSpec(t *testing.T) {
	specs := ParseColorSpec("FFFFFF|CCCCCC-101010")
	if len(specs) != 2 {
		t.Fatalf("len=%d", len(specs))
	}
	if specs[0] != (ColorSpec{R: 0xff, G: 0xff, B: 0xff, Tol: defaultTolerance}) {
		t.Fatalf("first=%+v", specs[0])
	}
	if specs[1].Tol != 0x10 || specs[1].R != 0xcc {
		t.Fatalf("second=%+v", specs[1])
	}
}
