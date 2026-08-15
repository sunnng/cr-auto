package game

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"testing"

	"app/internal/game/kingdom"
	"app/internal/game/mine"
	"app/internal/game/popup"
	"app/internal/vision"
)

// paintPointSpecs 把特征串的色点原样画到帧上（识别诊断测试用）。
func paintPointSpecs(img *image.NRGBA, spec string) {
	for _, chunk := range strings.Split(spec, ",") {
		parts := strings.Split(chunk, "|")
		if len(parts) < 3 {
			continue
		}
		x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		hex := strings.TrimSpace(parts[2])
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

func frameFromFeature(f vision.Feature) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 1600, 900))
	paintPointSpecs(img, f.Points)
	return img
}

func TestDetectSceneNilAndBlankFrame(t *testing.T) {
	d := DetectScene(nil)
	if d.Best != "" || d.Confidence != 0 || len(d.Candidates) != 0 || len(d.Anchors) != 0 {
		t.Fatalf("nil frame must detect nothing, got %+v", d)
	}

	d = DetectScene(image.NewNRGBA(image.Rect(0, 0, 1600, 900)))
	if d.Best != "" || d.Confidence != 0 || len(d.Candidates) != 0 {
		t.Fatalf("blank frame must detect nothing, got %+v", d)
	}
}

func TestDetectSceneKingdomHome(t *testing.T) {
	img := frameFromFeature(kingdom.Home().Feature)
	d := DetectScene(img)
	if d.Best != SceneKingdomHome {
		t.Fatalf("best scene=%q want %q", d.Best, SceneKingdomHome)
	}
	if d.Confidence != 1 {
		t.Fatalf("confidence=%v want 1", d.Confidence)
	}
	if len(d.Candidates) == 0 || d.Candidates[0].Key != SceneKingdomHome || d.Candidates[0].Matched != d.Candidates[0].Total {
		t.Fatalf("kingdom home must be the top candidate: %+v", d.Candidates)
	}
	if len(d.Anchors) == 0 {
		t.Fatal("anchors must include the best scene's points")
	}
	for _, a := range d.Anchors {
		if !a.Matched {
			t.Fatalf("all best-scene anchors must match, got %+v", a)
		}
	}
}

func TestDetectSceneUnstableNetwork(t *testing.T) {
	img := frameFromFeature(popup.UnstableNetworkDef().Feature)
	d := DetectScene(img)
	if d.Best != SceneUnstableNetwork {
		t.Fatalf("best scene=%q want %q", d.Best, SceneUnstableNetwork)
	}
}

func TestDetectSceneMineVentureDomain(t *testing.T) {
	img := frameFromFeature(mine.MineVenture().Setup.Feature)
	d := DetectScene(img)
	if d.Best != SceneMineVenture {
		t.Fatalf("best scene=%q want %q", d.Best, SceneMineVenture)
	}
}

func TestDetectScenePartialMatchScoresByRatio(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1600, 900))
	points := strings.Split(kingdom.Home().Feature.Points, ",")
	if len(points) < 2 {
		t.Fatal("kingdom home feature needs at least two points")
	}
	paintPointSpecs(img, strings.Join(points[:len(points)/2], ","))

	d := DetectScene(img)
	if d.Best != SceneKingdomHome {
		t.Fatalf("best scene=%q want %q", d.Best, SceneKingdomHome)
	}
	if d.Confidence >= 1 || d.Confidence <= 0 {
		t.Fatalf("partial match confidence=%v want (0,1)", d.Confidence)
	}
	if len(d.Candidates) == 0 || d.Candidates[0].Matched >= d.Candidates[0].Total {
		t.Fatalf("partial match must report matched<total: %+v", d.Candidates)
	}
}
