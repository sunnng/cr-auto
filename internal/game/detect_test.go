package game

import (
	"testing"

	"app/internal/game/kingdom"
	"app/internal/game/mine"
	"app/internal/game/popup"
	libcolor "app/internal/lib/color"
	"app/internal/vision"
)

func installDetectScreen(t *testing.T, s libcolor.Screen) {
	t.Helper()
	libcolor.SetScreen(s)
	t.Cleanup(func() { libcolor.SetScreen(nil) })
}

func TestDetectSceneNilAndBlankFrame(t *testing.T) {
	installDetectScreen(t, nil)
	d := DetectScene()
	if d.Best != "" || d.Confidence != 0 || len(d.Candidates) != 0 || len(d.Anchors) != 0 {
		t.Fatalf("no screen must detect nothing, got %+v", d)
	}

	installDetectScreen(t, libcolor.NewScriptedScreen())
	d = DetectScene()
	if d.Best != "" || d.Confidence != 0 || len(d.Candidates) != 0 {
		t.Fatalf("empty screen must detect nothing, got %+v", d)
	}
}

func TestDetectSceneKingdomHome(t *testing.T) {
	f := kingdom.Home().Feature
	installDetectScreen(t, libcolor.HitFeatures(f))
	d := DetectScene()
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
	installDetectScreen(t, libcolor.HitFeatures(popup.UnstableNetworkDef().Feature))
	d := DetectScene()
	if d.Best != SceneUnstableNetwork {
		t.Fatalf("best scene=%q want %q", d.Best, SceneUnstableNetwork)
	}
}

func TestDetectSceneMineVentureDomain(t *testing.T) {
	installDetectScreen(t, libcolor.HitFeatures(mine.MineVenture().Setup.Feature))
	d := DetectScene()
	if d.Best != SceneMineVenture {
		t.Fatalf("best scene=%q want %q", d.Best, SceneMineVenture)
	}
}

func TestDetectScenePartialMatchScoresByRatio(t *testing.T) {
	pts, err := vision.ParsePoints(kingdom.Home().Feature.Points)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) < 2 {
		t.Fatal("kingdom home feature needs at least two points")
	}
	s := libcolor.NewScriptedScreen()
	for _, p := range pts[:len(pts)/2] {
		s.HitPoint(p)
	}
	installDetectScreen(t, s)

	d := DetectScene()
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
