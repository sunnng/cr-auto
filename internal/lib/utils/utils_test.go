package utils

import (
	"image"
	"testing"
)

func TestParseNumber(t *testing.T) {
	cases := map[string]int{
		"1234":  1234,
		"1,234": 1234,
		"奖杯 56": 56,
	}
	for in, want := range cases {
		got, ok := ParseNumber(in)
		if !ok || got != want {
			t.Fatalf("ParseNumber(%q)=%d,%v want %d", in, got, ok, want)
		}
	}
	if _, ok := ParseNumber("abc"); ok {
		t.Fatal("non-numeric must fail")
	}
	if _, ok := ParseNumber(""); ok {
		t.Fatal("empty must fail")
	}
}

func TestParseStamina(t *testing.T) {
	st, ok := ParseStamina("12/30")
	if !ok || st.Current != 12 || st.Max != 30 {
		t.Fatalf("ParseStamina(12/30)=%+v,%v", st, ok)
	}
	st, ok = ParseStamina("1,234 / 5,678")
	if !ok || st.Current != 1234 || st.Max != 5678 {
		t.Fatalf("ParseStamina(1,234/5,678)=%+v,%v", st, ok)
	}
	if _, ok := ParseStamina("无斜杠"); ok {
		t.Fatal("no slash must fail")
	}
	if _, ok := ParseStamina(""); ok {
		t.Fatal("empty must fail")
	}
}

func TestGenerateNewPos(t *testing.T) {
	rect := GenerateNewPos(700, 600, 643, 531, 664, 521, 806, 556)
	want := image.Rect(664+57, 521+69, 806+57, 556+69)
	if rect != want {
		t.Fatalf("GenerateNewPos=%v want %v", rect, want)
	}
}

func TestKeepHanAlphaNum(t *testing.T) {
	got := KeepHanAlphaNum("胜利(胜利)! 12:34abc")
	want := "胜利胜利1234abc"
	if got != want {
		t.Fatalf("KeepHanAlphaNum=%q want %q", got, want)
	}
	if KeepHanAlphaNum("") != "" {
		t.Fatal("empty must stay empty")
	}
}
