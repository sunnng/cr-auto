package game

import (
	"testing"

	"app/internal/core"
)

func TestRegisterAllClearsAndReports(t *testing.T) {
	s := core.NewScheduler()
	g := core.NewGuard()
	s.Add("stale", func() bool { return false }, func() error { return nil })
	g.Register("staleTrap", func() bool { return false }, func() {}, 1)

	RegisterAll(s, g)

	if s.Count() != 0 {
		t.Fatalf("register must clear scheduler, count=%d", s.Count())
	}
	if g.TrapCount() != 0 {
		t.Fatalf("register must clear guard, count=%d", g.TrapCount())
	}
}
