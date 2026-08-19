package ui

import "testing"

func TestOverlayScaleUsesImageBoundsNotFixedDisplay(t *testing.T) {
	sx, sy := OverlayScale(800, 450, 400, 225)
	if sx != 0.5 || sy != 0.5 {
		t.Fatalf("scale=(%v,%v)", sx, sy)
	}
	sx, sy = OverlayScale(0, 0, 400, 225)
	if sx != 1 || sy != 1 {
		t.Fatalf("empty image must not divide by zero: %v %v", sx, sy)
	}
}
