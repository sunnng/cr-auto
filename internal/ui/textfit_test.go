package ui

import (
	"strings"
	"testing"
)

func TestFitRunesEllipsizesToPixelWidth(t *testing.T) {
	measure := func(s string) float32 { return float32(len([]rune(s)) * 10) }
	got := FitRunes("abcdefghij", 50, measure)
	if got == "abcdefghij" || !strings.HasSuffix(got, "…") {
		t.Fatalf("got %q", got)
	}
	if measure(got) > 50 {
		t.Fatalf("still too wide: %q", got)
	}
}

func TestFitRunesKeepsShortText(t *testing.T) {
	measure := func(s string) float32 { return float32(len([]rune(s)) * 10) }
	if got := FitRunes("ab", 50, measure); got != "ab" {
		t.Fatalf("got %q", got)
	}
}
