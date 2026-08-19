package popup

import (
	"testing"

	"app/internal/vision"
)

func TestSafetyFeaturesReadyRequiresNonEmptyPoints(t *testing.T) {
	if SafetyFeaturesReady() {
		t.Fatal("empty or placeholder features must not claim readiness")
	}
}

func TestFeaturesReady(t *testing.T) {
	empty := []vision.Feature{{Sim: 0.9}}
	filled := []vision.Feature{{Points: "10|10|ffffff-101010", Sim: 0.9}}
	if FeaturesReady(empty, filled) {
		t.Fatal("any empty group must fail")
	}
	if !FeaturesReady(filled, filled) {
		t.Fatal("both groups filled must pass")
	}
	if FeaturesReady() {
		t.Fatal("no groups must fail")
	}
}
