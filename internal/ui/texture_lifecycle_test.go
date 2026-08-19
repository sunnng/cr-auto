package ui

import "testing"

func TestShouldRebuildTexture(t *testing.T) {
	if ShouldRebuildTexture(3, 3, true, true) {
		t.Fatal("same revision must keep texture")
	}
	if !ShouldRebuildTexture(3, 4, true, true) {
		t.Fatal("newer revision must rebuild")
	}
	if !ShouldRebuildTexture(0, 1, false, true) {
		t.Fatal("missing texture must build")
	}
}
