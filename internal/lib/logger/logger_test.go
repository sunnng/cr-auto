package logger

import "testing"

func recorder(t *testing.T) (func() []string, func()) {
	t.Helper()
	var lines []string
	prev := sink
	prevLevel := now
	SetSink(func(level Level, tag, message string) {
		lines = append(lines, message)
	})
	restore := func() {
		SetSink(prev)
		SetLevel(prevLevel)
	}
	return func() []string { return lines }, restore
}

func TestSinkReceivesOnlyAtOrAboveLevel(t *testing.T) {
	snapshot, restore := recorder(t)
	defer restore()
	SetLevel(LevelWarn)
	Debug("T", "debug %d", 1)
	Info("T", "info %d", 2)
	Warn("T", "warn %d", 3)
	Error("T", "error %d", 4)
	lines := snapshot()
	if len(lines) != 2 || lines[0] != "warn 3" || lines[1] != "error 4" {
		t.Fatalf("lines=%v", lines)
	}
}

func TestNilSinkDiscards(t *testing.T) {
	SetSink(nil)
	Debug("T", "dropped")
}

func TestAllLevels(t *testing.T) {
	snapshot, restore := recorder(t)
	defer restore()
	SetLevel(LevelDebug)
	Debug("T", "d")
	Info("T", "i")
	Warn("T", "w")
	Error("T", "e")
	lines := snapshot()
	if len(lines) != 4 {
		t.Fatalf("expected all four levels, got %v", lines)
	}
}
