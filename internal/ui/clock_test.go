package ui

import "testing"

func TestJoinSplitClockRoundTrip(t *testing.T) {
	h, m := SplitClock(90)
	if h != 1 || m != 30 {
		t.Fatalf("%d:%d", h, m)
	}
	if JoinClock(1, 30) != 90 {
		t.Fatal("join")
	}
	if JoinClock(24, 99) != 1439 {
		t.Fatalf("got %d", JoinClock(24, 99))
	}
}
