package game

import (
	"testing"

	"app/internal/core"
	"app/internal/game/kingdom"
	libcolor "app/internal/lib/color"
	"app/internal/lib/touch"
	"app/internal/vision"
)

func TestRegisterSafetyGuardsStopsOnSensitiveMatch(t *testing.T) {
	setupRegisterTest(t)
	var reason string
	SetSafetyStop(func(r string) { reason = r })
	t.Cleanup(func() { SetSafetyStop(nil) })

	feat := kingdom.Home().Feature
	libcolor.SetScreen(libcolor.HitFeatures(feat))

	g := core.NewGuard()
	RegisterSafetyGuards(g, nil, []vision.Feature{feat})

	if !g.Check() {
		t.Fatal("sensitive page must hit")
	}
	if reason == "" {
		t.Fatal("safety stop callback must run")
	}
}

func TestRegisterSafetyGuardsPressesBackOnResourceSpend(t *testing.T) {
	var backed bool
	rec := setupRegisterTest(t)
	touch.SetPerform(touch.Perform{
		Tap:    rec.tap,
		Back:   func() { backed = true },
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})
	var reason string
	SetSafetyStop(func(r string) { reason = r })
	t.Cleanup(func() { SetSafetyStop(nil) })

	feat := kingdom.Home().Feature
	libcolor.SetScreen(libcolor.HitFeatures(feat))

	g := core.NewGuard()
	RegisterSafetyGuards(g, []vision.Feature{feat}, nil)

	if !g.Check() {
		t.Fatal("resource spend must hit")
	}
	if !backed {
		t.Fatal("resource spend must press back instead of confirming")
	}
	if reason == "" {
		t.Fatal("safety stop callback must run")
	}
}

func TestSafetyGuardsReadyFollowsCapturedFeatures(t *testing.T) {
	if SafetyGuardsReady() {
		t.Fatal("uncaptured safety features must not report ready")
	}
}
