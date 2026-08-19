package ui

import (
	"testing"
	"time"
)

func TestPillLayoutHitTargets(t *testing.T) {
	layout := expandedPillLayout(100, 20)
	tests := []struct {
		name  string
		point pillPoint
		want  pillHitTarget
	}{
		{"config", pillPoint{X: 461, Y: 76}, pillHitConfig},
		{"primary", pillPoint{X: 521, Y: 76}, pillHitPrimary},
		{"stop", pillPoint{X: 581, Y: 76}, pillHitStop},
		{"background", pillPoint{X: 500, Y: 140}, pillHitNone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := layout.hit(test.point); got != test.want {
				t.Fatalf("hit=%v want=%v", got, test.want)
			}
		})
	}
}

func TestExpandedPillPrioritizesInfoWithCompactControlGrid(t *testing.T) {
	layout := expandedPillLayout(100, 20)
	if layout.body.Max.X-layout.body.Min.X != 520 || layout.body.Max.Y-layout.body.Min.Y != 112 {
		t.Fatalf("body=%+v want 520x112", layout.body)
	}
	for name, rect := range map[string]pillRect{
		"config":  layout.config,
		"primary": layout.primary,
		"stop":    layout.stop,
	} {
		if height := rect.Max.Y - rect.Min.Y; height < 48 {
			t.Fatalf("%s height=%v want at least 48", name, height)
		}
		if width := rect.Max.X - rect.Min.X; width != 52 {
			t.Fatalf("%s width=%v want 52", name, width)
		}
	}
	if layout.primary.Min.X-layout.config.Max.X != 8 || layout.stop.Min.X-layout.primary.Max.X != 8 {
		t.Fatalf("controls do not match compact right-aligned 8px grid: %+v", layout)
	}
	if rightPadding := layout.body.Max.X - layout.stop.Max.X; rightPadding != 13 {
		t.Fatalf("right padding=%v want 13", rightPadding)
	}
}

func TestCollapsedAnimatedPillOnlyExposesExpandTarget(t *testing.T) {
	visual := expandedPillLayoutForBounds(635, 16, pillCollapsedWidth, pillCollapsedHeight)
	layout := pillInteractionLayout(false, visual)

	for _, point := range []pillPoint{
		{X: 650, Y: 36},
		{X: 740, Y: 36},
		{X: 800, Y: 36},
		{X: 930, Y: 36},
	} {
		if got := layout.hit(point); got != pillHitExpand {
			t.Fatalf("collapsed point %+v hit=%v want expand", point, got)
		}
	}

	var pointer pillPointerState
	hit := pointer.update(pillPoint{X: 800, Y: 36}, false, false, true, layout, time.Time{})
	frame := panelFrame{Compact: true, Draft: Default()}
	commands := applyPillHit(&frame, hit, nil)
	if !frame.PillExpanded || !frame.Compact || len(commands) != 0 {
		t.Fatalf("collapsed center click must only expand: frame=%+v commands=%+v", frame, commands)
	}
}

func TestPillControlsStayDisabledUntilExpansionFinishes(t *testing.T) {
	visual := expandedPillLayout(100, 20)
	now := time.Unix(100, 0)
	frame := panelFrame{PillExpanded: true, PillExpansion: 0.75, PillControlsAt: now.Add(pillControlsArmDelay)}
	opening := pillInteractionLayout(pillControlsEnabled(&frame, now), visual)
	if got := opening.hit(pillPoint{X: 521, Y: 76}); got != pillHitExpand {
		t.Fatalf("opening animation hit=%v want harmless expand", got)
	}

	frame.PillExpansion = 1
	arming := pillInteractionLayout(pillControlsEnabled(&frame, now.Add(pillControlsArmDelay-time.Millisecond)), visual)
	if got := arming.hit(pillPoint{X: 521, Y: 76}); got != pillHitExpand {
		t.Fatalf("arming delay hit=%v want harmless expand", got)
	}

	ready := pillInteractionLayout(pillControlsEnabled(&frame, now.Add(pillControlsArmDelay)), visual)
	if got := ready.hit(pillPoint{X: 521, Y: 76}); got != pillHitPrimary {
		t.Fatalf("fully expanded hit=%v want primary", got)
	}
}

func TestPillExpansionAnimatesAcrossPrototypeDuration(t *testing.T) {
	start := time.Unix(100, 0)
	frame := panelFrame{PillExpanded: true, PillAnimationAt: start}

	advancePillAnimation(&frame, start.Add(pillAnimationDuration/2))
	if frame.PillExpansion < 0.49 || frame.PillExpansion > 0.51 {
		t.Fatalf("half-duration expansion=%v want approximately 0.5", frame.PillExpansion)
	}
	advancePillAnimation(&frame, start.Add(pillAnimationDuration))
	if frame.PillExpansion != 1 {
		t.Fatalf("full-duration expansion=%v want 1", frame.PillExpansion)
	}

	frame.PillExpanded = false
	advancePillAnimation(&frame, start.Add(pillAnimationDuration+pillAnimationDuration/2))
	if frame.PillExpansion < 0.49 || frame.PillExpansion > 0.51 {
		t.Fatalf("closing half-duration expansion=%v want approximately 0.5", frame.PillExpansion)
	}
}

func TestPillPointerRequiresReleaseInsideOriginalTarget(t *testing.T) {
	layout := expandedPillLayout(100, 20)
	var pointer pillPointerState

	if got := pointer.update(pillPoint{X: 461, Y: 76}, true, true, false, layout, time.Time{}); got != pillHitNone {
		t.Fatalf("press emitted action %v", got)
	}
	if got := pointer.update(pillPoint{X: 581, Y: 76}, false, false, true, layout, time.Time{}); got != pillHitNone {
		t.Fatalf("release over a different target emitted action %v", got)
	}

	pointer.update(pillPoint{X: 521, Y: 76}, true, true, false, layout, time.Time{})
	if got := pointer.update(pillPoint{X: 521, Y: 76}, false, false, true, layout, time.Time{}); got != pillHitPrimary {
		t.Fatalf("primary release=%v", got)
	}
}

func TestPillPointerAcceptsReleaseOnlyAutoGoEvent(t *testing.T) {
	layout := collapsedPillLayout(635, 12)
	var pointer pillPointerState
	if got := pointer.update(pillPoint{X: 800, Y: 30}, false, false, true, layout, time.Time{}); got != pillHitExpand {
		t.Fatalf("release-only hit=%v want expand", got)
	}
}

func TestPillPointerUsesLastDownPointWhenTouchReleaseResetsMousePosition(t *testing.T) {
	layout := collapsedPillLayout(635, 12)
	var pointer pillPointerState
	pointer.update(pillPoint{X: 800, Y: 30}, true, false, false, layout, time.Time{})
	invalid := pillPoint{X: -3.4028235e38, Y: -3.4028235e38}
	if got := pointer.update(invalid, false, false, true, layout, time.Time{}); got != pillHitExpand {
		t.Fatalf("cached release hit=%v want expand", got)
	}
}

func TestPillClickExpandsForSixSeconds(t *testing.T) {
	frame := panelFrame{}
	before := time.Now()
	applyPillHit(&frame, pillHitExpand, nil)
	if !frame.PillExpanded {
		t.Fatal("click did not expand pill")
	}
	if frame.PillCollapseAt.Before(before.Add(5*time.Second)) || frame.PillCollapseAt.After(before.Add(7*time.Second)) {
		t.Fatalf("unexpected auto-collapse deadline: %v", frame.PillCollapseAt)
	}
}

func TestPillPrimaryActionTracksRuntimePhase(t *testing.T) {
	tests := []struct {
		phase, outcome, label string
		want                  CommandType
		enabled               bool
	}{
		{"idle", "", "开始", CommandStart, true},
		{"running", "", "暂停", CommandPause, true},
		{"running", "scheduled_wait", "等待", CommandPause, false},
		{"waiting", "", "暂停", CommandPause, true},
		{"paused", "", "继续", CommandResume, true},
	}
	for _, test := range tests {
		label, command, enabled := pillPrimaryAction(test.phase, test.outcome)
		if label != test.label || command != test.want || enabled != test.enabled {
			t.Fatalf("phase=%q outcome=%q got=(%q,%q,%v) want=(%q,%q,%v)", test.phase, test.outcome, label, command, enabled, test.label, test.want, test.enabled)
		}
	}
}

func TestApplyPillHitReopensConfigAndBuildsStartCommand(t *testing.T) {
	frame := panelFrame{Compact: true, PillExpanded: true, Draft: Default()}
	commands := applyPillHit(&frame, pillHitConfig, nil)
	if frame.Compact || frame.PillExpanded || !frame.PillCollapseAt.IsZero() || len(commands) != 0 {
		t.Fatalf("config hit did not reopen panel: frame=%+v commands=%+v", frame, commands)
	}

	frame.Compact = true
	commands = applyPillHit(&frame, pillHitPrimary, nil)
	if len(commands) != 1 || commands[0].Type != CommandStart || commands[0].Settings == nil {
		t.Fatalf("idle primary did not create start command: %+v", commands)
	}
}

func TestHoverRefreshesAutoCollapseDeadline(t *testing.T) {
	now := time.Unix(100, 0)
	frame := &panelFrame{PillExpanded: true, PillCollapseAt: now.Add(time.Second)}
	refreshPillCollapse(frame, now.Add(500*time.Millisecond), true)
	if !frame.PillCollapseAt.After(now.Add(pillAutoCollapse - time.Second)) {
		t.Fatalf("deadline not refreshed: %v", frame.PillCollapseAt)
	}
}

func TestStopConfirmedRequiresHold(t *testing.T) {
	if stopConfirmed(799 * time.Millisecond) {
		t.Fatal("short press must not confirm stop")
	}
	if !stopConfirmed(pillConfirmHold) {
		t.Fatal("800ms hold must confirm stop")
	}
}

func TestShortStopPressDoesNotEmitCommand(t *testing.T) {
	layout := expandedPillLayout(100, 20)
	var pointer pillPointerState
	now := time.Unix(100, 0)
	stop := pillPoint{X: 581, Y: 76}
	pointer.update(stop, true, true, false, layout, now)
	hit := pointer.update(stop, false, false, true, layout, now.Add(100*time.Millisecond))
	commands := applyPillHit(&panelFrame{PillExpanded: true}, hit, nil)
	if len(commands) != 0 {
		t.Fatalf("short stop press emitted %+v", commands)
	}
}

func TestStopHoldEmitsCommandStop(t *testing.T) {
	layout := expandedPillLayout(100, 20)
	var pointer pillPointerState
	now := time.Unix(100, 0)
	stop := pillPoint{X: 581, Y: 76}
	if got := pointer.update(stop, true, true, false, layout, now); got != pillHitNone {
		t.Fatalf("press=%v", got)
	}
	hit := pointer.update(stop, true, false, false, layout, now.Add(pillConfirmHold))
	if hit != pillHitStop {
		t.Fatalf("hold hit=%v want stop", hit)
	}
	commands := applyPillHit(&panelFrame{PillExpanded: true}, hit, nil)
	if len(commands) != 1 || commands[0].Type != CommandStop {
		t.Fatalf("hold must emit CommandStop: %+v", commands)
	}
}

func TestScheduledWaitPrimaryDoesNotEmitPause(t *testing.T) {
	frame := panelFrame{Status: RuntimeStatus{Phase: "running", Outcome: "scheduled_wait"}, Compact: true, PillExpanded: true}
	commands := applyPillHit(&frame, pillHitPrimary, nil)
	if len(commands) != 0 {
		t.Fatalf("got %v", commands)
	}
}

func TestPillExpandedStateShowsScheduledWait(t *testing.T) {
	if got := pillExpandedState("running", "scheduled_wait"); got != "等待计划时段" {
		t.Fatalf("%q", got)
	}
}

func TestCompactPillHeadlinePrefersLatestMessage(t *testing.T) {
	got := CompactPillHeadline(RuntimeStatus{Message: "引擎已启动", Scene: "x", Outcome: "old"}, []string{"a", "b"})
	if got != "引擎已启动" {
		t.Fatalf("%q", got)
	}
}

func TestConfirmExitRequiresSecondClick(t *testing.T) {
	frame := &panelFrame{}
	now := time.Unix(100, 0)
	if confirmExit(frame, now, true) {
		t.Fatal("first click must only arm")
	}
	if frame.ExitArmedUntil.IsZero() {
		t.Fatal("first click must arm exit")
	}
	if !confirmExit(frame, now.Add(time.Second), true) {
		t.Fatal("second click within 2s must confirm")
	}
}

func TestConfirmExitExpires(t *testing.T) {
	frame := &panelFrame{}
	now := time.Unix(100, 0)
	confirmExit(frame, now, true)
	if confirmExit(frame, now.Add(3*time.Second), true) {
		t.Fatal("expired arm must not exit")
	}
}
