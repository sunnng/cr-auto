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

func TestCorruptFileResetsToEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(path)
	v, err := s.Get("k", 42)
	if err != nil || v != 42 {
		t.Fatalf("corrupt store must fall back to defaults: got=%v err=%v", v, err)
	}
	// 下次写入必须覆盖损坏文件。
	if err := s.Set("k", "v"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "{not json" {
		t.Fatal("corrupt store must be overwritten on save")
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
	if v, err := s.Incr("count", 1, 0); err != nil || v != 1 {
		t.Fatalf("first Incr=%d err=%v", v, err)
	}
	if v, err := s.Incr("count", 2, 0); err != nil || v != 3 {
		t.Fatalf("second Incr=%d err=%v", v, err)
	}
	// 缺省值参数：键不存在时从 def 起算。
	if v, err := s.Incr("fresh", 1, 10); err != nil || v != 11 {
		t.Fatalf("Incr with default=%d err=%v", v, err)
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

type testRaw struct {
	FarWaitUntil int64 `json:"farWaitUntil"`
}

func TestDecodeMapToStruct(t *testing.T) {
	s := tempStore(t)
	if err := s.Set("k", map[string]any{"farWaitUntil": float64(1700000000)}); err != nil {
		t.Fatal(err)
	}
	v, err := s.Get("k", nil)
	if err != nil {
		t.Fatal(err)
	}
	var raw testRaw
	if err := Decode(v, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.FarWaitUntil != 1700000000 {
		t.Fatalf("decoded=%d", raw.FarWaitUntil)
	}
}

func TestDecodeInvalidValueErrors(t *testing.T) {
	if err := Decode("not-a-map", &testRaw{}); err == nil {
		t.Fatal("invalid value must fail to decode")
	}
}

func TestLoadDecodesIntoStruct(t *testing.T) {
	s := tempStore(t)
	if err := s.Set("k", map[string]any{"farWaitUntil": float64(123)}); err != nil {
		t.Fatal(err)
	}
	var raw testRaw
	if !s.Load("k", &raw) || raw.FarWaitUntil != 123 {
		t.Fatalf("load=%+v", raw)
	}
	if s.Load("missing", &raw) {
		t.Fatal("missing key must fail to load")
	}
}
