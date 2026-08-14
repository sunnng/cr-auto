package store

import (
	"os"
	"path/filepath"
	"testing"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store.json")
	return New(path)
}

func TestSetGetRoundTrip(t *testing.T) {
	s := tempStore(t)
	if err := s.Set("mine_venture_session", map[string]any{"farWaitUntil": float64(1700000000)}); err != nil {
		t.Fatal(err)
	}
	v, err := s.Get("mine_venture_session", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := v.(map[string]any)
	if got := raw["farWaitUntil"].(float64); got != 1700000000 {
		t.Fatalf("farWaitUntil=%v", got)
	}
}

func TestGetReturnsDefaultForMissingKey(t *testing.T) {
	s := tempStore(t)
	v, err := s.Get("missing", 42)
	if err != nil || v != 42 {
		t.Fatalf("got=%v err=%v", v, err)
	}
}

func TestPersistenceAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	if err := New(path).Set("k", "v1"); err != nil {
		t.Fatal(err)
	}
	v, err := New(path).Get("k", nil)
	if err != nil || v != "v1" {
		t.Fatalf("got=%v err=%v", v, err)
	}
}

func TestCorruptFileFailsLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path).Get("k", nil); err == nil {
		t.Fatal("corrupt store must surface a load error")
	}
}

func TestDelAndHas(t *testing.T) {
	s := tempStore(t)
	_ = s.Set("k", 1)
	ok, err := s.Has("k")
	if err != nil || !ok {
		t.Fatalf("Has=%v err=%v", ok, err)
	}
	if err := s.Del("k"); err != nil {
		t.Fatal(err)
	}
	ok, err = s.Has("k")
	if err != nil || ok {
		t.Fatalf("Has after Del=%v err=%v", ok, err)
	}
}

func TestIncr(t *testing.T) {
	s := tempStore(t)
	if v, err := s.Incr("count", 1); err != nil || v != 1 {
		t.Fatalf("first Incr=%d err=%v", v, err)
	}
	if v, err := s.Incr("count", 2); err != nil || v != 3 {
		t.Fatalf("second Incr=%d err=%v", v, err)
	}
}

func TestClear(t *testing.T) {
	s := tempStore(t)
	_ = s.Set("k", 1)
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	ok, _ := s.Has("k")
	if ok {
		t.Fatal("key must be gone after Clear")
	}
}

func TestDefaultStoreSettable(t *testing.T) {
	SetDefault(nil)
	if Default() == nil {
		t.Fatal("Default must never be nil")
	}
	path := filepath.Join(t.TempDir(), "store.json")
	SetDefault(New(path))
	if Default() != nil && Default().path != path {
		t.Fatalf("Default path=%q", Default().path)
	}
	SetDefault(nil)
}
