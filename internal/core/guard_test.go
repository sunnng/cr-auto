package core

import (
	"testing"

	"app/internal/lib/color"
	"app/internal/vision"
)

func installGuardScreen(t *testing.T, s color.Screen) func() {
	t.Helper()
	color.SetScreen(s)
	return func() { color.SetScreen(nil) }
}

func TestGuardRegisterClearAndCount(t *testing.T) {
	g := NewGuard()
	if g.TrapCount() != 0 {
		t.Fatal("fresh guard must be empty")
	}
	g.Register("弹窗A", func() bool { return false }, func() {}, 10)
	g.Register("弹窗B", func() bool { return false }, func() {}, 5)
	if g.TrapCount() != 2 {
		t.Fatalf("count=%d", g.TrapCount())
	}
	g.Clear()
	if g.TrapCount() != 0 {
		t.Fatal("clear must empty the guard")
	}
}

func TestGuardCheckHandlesFirstHit(t *testing.T) {
	defer installGuardScreen(t, color.HitFeatures(vision.Feature{Points: "1|1|ff0000-000000"}))()
	var handled []string
	g := NewGuard()
	g.Register("弹窗A", func() bool { return false }, func() { handled = append(handled, "A") }, 0)
	g.Register("弹窗B", vision.Feature{Points: "1|1|ff0000-000000"}, func() { handled = append(handled, "B") }, 0)
	if !g.Check() {
		t.Fatal("check must handle the matching trap")
	}
	if len(handled) != 1 || handled[0] != "B" {
		t.Fatalf("handled=%v", handled)
	}
}

func TestGuardCheckPriorityOrder(t *testing.T) {
	defer installGuardScreen(t, color.NewScriptedScreen())()
	var handled []string
	g := NewGuard()
	g.Register("low", func() bool { return true }, func() { handled = append(handled, "low") }, 1)
	g.Register("high", func() bool { return true }, func() { handled = append(handled, "high") }, 10)
	if !g.Check() {
		t.Fatal("check must handle")
	}
	if len(handled) != 1 || handled[0] != "high" {
		t.Fatalf("priority order violated: %v", handled)
	}
}

func TestGuardCheckNameTieBreak(t *testing.T) {
	defer installGuardScreen(t, color.NewScriptedScreen())()
	var handled []string
	g := NewGuard()
	g.Register("zeta", func() bool { return true }, func() { handled = append(handled, "zeta") }, 5)
	g.Register("alpha", func() bool { return true }, func() { handled = append(handled, "alpha") }, 5)
	g.Check()
	if len(handled) != 1 || handled[0] != "alpha" {
		t.Fatalf("equal priority must break ties by name: %v", handled)
	}
}

func TestGuardCheckSkipsWhenNothingMatches(t *testing.T) {
	defer installGuardScreen(t, color.HitFeatures(vision.Feature{Points: "1|1|ff0000-000000"}))()
	g := NewGuard()
	g.Register("miss", func() bool { return false }, func() { t.Fatal("must not run") }, 0)
	if g.Check() {
		t.Fatal("check must report no hit")
	}
}

func TestGuardCheckStopsOnFirstHandled(t *testing.T) {
	defer installGuardScreen(t, color.NewScriptedScreen())()
	var handled []string
	g := NewGuard()
	g.Register("hit", func() bool { return true }, func() { handled = append(handled, "hit") }, 0)
	g.Register("alsoHit", func() bool { return true }, func() { handled = append(handled, "alsoHit") }, 0)
	g.Check()
	if len(handled) != 1 {
		t.Fatalf("check must stop after first handled trap: %v", handled)
	}
}

func TestGuardCheckHandlerPanicReportsFailure(t *testing.T) {
	defer installGuardScreen(t, color.NewScriptedScreen())()
	g := NewGuard()
	g.Register("panic", func() bool { return true }, func() { panic("boom") }, 0)
	if g.Check() {
		t.Fatal("handler panic must make check return false")
	}
}

func TestGuardSleepFragmentsWithGuardChecks(t *testing.T) {
	defer installGuardScreen(t, color.NewScriptedScreen())()
	var checks, sleeps []int
	g := NewGuard()
	g.SetSleep(func(ms int) { sleeps = append(sleeps, ms) })
	g.Register("count", func() bool {
		checks = append(checks, 1)
		return false
	}, func() {}, 0)
	g.Sleep(1200, 500)
	if len(sleeps) != 3 || sleeps[0] != 500 || sleeps[1] != 500 || sleeps[2] != 200 {
		t.Fatalf("sleeps=%v", sleeps)
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 guard checks, got %d", len(checks))
	}
}
