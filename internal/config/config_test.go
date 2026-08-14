package config

import "testing"

func TestRuntimeDefaultsMirrorLuaConfig(t *testing.T) {
	if got := Static.Runtime.GuardIntervalMS; got != 500 {
		t.Fatalf("GuardIntervalMS=%d", got)
	}
	if Static.Runtime.StopOnError {
		t.Fatal("StopOnError must default false")
	}
	if got := Static.Runtime.StepDelayMS; got != 5000 {
		t.Fatalf("StepDelayMS=%d", got)
	}
	if got := Static.Runtime.IdleDelayMS; got != 30000 {
		t.Fatalf("IdleDelayMS=%d", got)
	}
}

func TestDisplayFixed1600x900(t *testing.T) {
	if Static.Display.Width != 1600 || Static.Display.Height != 900 {
		t.Fatalf("display=%dx%d", Static.Display.Width, Static.Display.Height)
	}
}

func TestUserDefaultsMirrorLuaConfig(t *testing.T) {
	mine := Static.User.Mine
	if !mine.SurveyEnabled || mine.MiningEnabled || mine.BattleEnabled || mine.JellyEnabled {
		t.Fatalf("mine switches mismatch: %+v", mine)
	}
	if mine.TargetFloor != 6 || mine.FarGap != 2 || mine.FarWaitSec != 600 || mine.OcrPollSec != 60 {
		t.Fatalf("mine params mismatch: %+v", mine)
	}
	if len(mine.MiningOreCards) != 6 {
		t.Fatalf("mining ore cards=%v", mine.MiningOreCards)
	}
	if !Static.User.Square.Enabled || Static.User.Square.DailyCap != 240 {
		t.Fatalf("square mismatch: %+v", Static.User.Square)
	}
	if Static.User.Starlight.Enabled || Static.User.Arena.Enabled || Static.User.SeasideMarket.Enabled || Static.User.Biscuit.Enabled {
		t.Fatal("M2+ modules must default disabled")
	}
}
