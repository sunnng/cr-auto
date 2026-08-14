package userconfig

import (
	"path/filepath"
	"testing"

	"app/internal/config"
	"app/internal/lib/store"
)

func newUserConfig(t *testing.T) *UserConfig {
	t.Helper()
	return New(store.New(filepath.Join(t.TempDir(), "store.json")))
}

func TestGetReturnsDefaults(t *testing.T) {
	u := newUserConfig(t)
	var mine config.MineConfig
	if err := u.Get("mine", &mine); err != nil {
		t.Fatal(err)
	}
	if !mine.SurveyEnabled || mine.TargetFloor != 6 {
		t.Fatalf("mine defaults mismatch: %+v", mine)
	}
}

func TestSetPersistsOverrides(t *testing.T) {
	u := newUserConfig(t)
	if err := u.Set("mine", map[string]any{"targetFloor": 9, "surveyEnabled": false}); err != nil {
		t.Fatal(err)
	}
	// 未覆盖字段保持默认值。
	var mine config.MineConfig
	if err := u.Get("mine", &mine); err != nil {
		t.Fatal(err)
	}
	if mine.TargetFloor != 9 || mine.SurveyEnabled {
		t.Fatalf("overrides not applied: %+v", mine)
	}
	if mine.FarWaitSec != 600 {
		t.Fatalf("untouched field lost default: %+v", mine)
	}
}

func TestUnknownSectionRejected(t *testing.T) {
	u := newUserConfig(t)
	var out any
	if err := u.Get("no_such_section", &out); err != errUnknownSection {
		t.Fatalf("expected errUnknownSection, got %v", err)
	}
	if err := u.Set("no_such_section", map[string]any{}); err != errUnknownSection {
		t.Fatalf("Set must reject unknown section, got %v", err)
	}
}

func TestReloadFromStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	u := New(store.New(path))
	if err := u.Set("arena", map[string]any{"enabled": true, "autoBuyCount": 3}); err != nil {
		t.Fatal(err)
	}
	// 新实例从持久化恢复。
	u2 := New(store.New(path))
	var arena config.ArenaConfig
	if err := u2.Get("arena", &arena); err != nil {
		t.Fatal(err)
	}
	if !arena.Enabled || arena.AutoBuyCount != 3 {
		t.Fatalf("reloaded arena mismatch: %+v", arena)
	}
	if arena.TrophyDiff != 0 {
		t.Fatalf("defaults lost on reload: %+v", arena)
	}
}

func TestLoadReturnsCopies(t *testing.T) {
	u := newUserConfig(t)
	all, err := u.Load()
	if err != nil {
		t.Fatal(err)
	}
	all["mine"]["targetFloor"] = 999
	var mine config.MineConfig
	if err := u.Get("mine", &mine); err != nil {
		t.Fatal(err)
	}
	if mine.TargetFloor != 6 {
		t.Fatalf("Load copy mutated cache: %+v", mine)
	}
}
