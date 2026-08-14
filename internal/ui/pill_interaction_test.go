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
		{"config", pillPoint{X: 500, Y: 64}, pillHitConfig},
		{"primary", pillPoint{X: 540, Y: 64}, pillHitPrimary},
		{"stop", pillPoint{X: 590, Y: 64}, pillHitStop},
		{"background", pillPoint{X: 500, Y: 100}, pillHitNone},
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
	if layout.body.Max.X-layout.body.Min.X != 520 || layout.body.Max.Y-layout.body.Min.Y != 88 {
		t.Fatalf("body=%+v want 520x88", layout.body)
	}
	for name, rect := range map[string]pillRect{
		"config":  layout.config,
		"primary": layout.primary,
		"stop":    layout.stop,
	} {
		if height := rect.Max.Y - rect.Min.Y; height != 34 {
			t.Fatalf("%s height=%v want 34", name, height)
		}
	}
	if layout.config.Min.X != 485 || layout.primary.Min.X-layout.config.Max.X != 6 || layout.stop.Min.X-layout.primary.Max.X != 6 {
		t.Fatalf("controls do not match compact right-aligned 6px grid: %+v", layout)
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
	hit := pointer.update(pillPoint{X: 800, Y: 36}, false, false, true, layout)
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
	if got := opening.hit(pillPoint{X: 540, Y: 64}); got != pillHitExpand {
		t.Fatalf("opening animation hit=%v want harmless expand", got)
	}

	frame.PillExpansion = 1
	arming := pillInteractionLayout(pillControlsEnabled(&frame, now.Add(pillControlsArmDelay-time.Millisecond)), visual)
	if got := arming.hit(pillPoint{X: 540, Y: 64}); got != pillHitExpand {
		t.Fatalf("arming delay hit=%v want harmless expand", got)
	}

	ready := pillInteractionLayout(pillControlsEnabled(&frame, now.Add(pillControlsArmDelay)), visual)
	if got := ready.hit(pillPoint{X: 540, Y: 64}); got != pillHitPrimary {
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

	if got := pointer.update(pillPoint{X: 500, Y: 64}, true, true, false, layout); got != pillHitNone {
		t.Fatalf("press emitted action %v", got)
	}
	if got := pointer.update(pillPoint{X: 590, Y: 64}, false, false, true, layout); got != pillHitNone {
		t.Fatalf("release over a different target emitted action %v", got)
	}

	pointer.update(pillPoint{X: 540, Y: 64}, true, true, false, layout)
	if got := pointer.update(pillPoint{X: 540, Y: 64}, false, false, true, layout); got != pillHitPrimary {
		t.Fatalf("primary release=%v", got)
	}
}

func TestPillPointerAcceptsReleaseOnlyAutoGoEvent(t *testing.T) {
	layout := collapsedPillLayout(635, 12)
	var pointer pillPointerState
	if got := pointer.update(pillPoint{X: 800, Y: 30}, false, false, true, layout); got != pillHitExpand {
		t.Fatalf("release-only hit=%v want expand", got)
	}
}

func TestPillPointerUsesLastDownPointWhenTouchReleaseResetsMousePosition(t *testing.T) {
	layout := collapsedPillLayout(635, 12)
	var pointer pillPointerState
	pointer.update(pillPoint{X: 800, Y: 30}, true, false, false, layout)
	invalid := pillPoint{X: -3.4028235e38, Y: -3.4028235e38}
	if got := pointer.update(invalid, false, false, true, layout); got != pillHitExpand {
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
		phase string
		label string
		want  CommandType
	}{
		{"idle", "开始", CommandStart},
		{"running", "暂停", CommandPause},
		{"waiting", "暂停", CommandPause},
		{"paused", "继续", CommandResume},
	}
	for _, test := range tests {
		label, command := pillPrimaryAction(test.phase)
		if label != test.label || command != test.want {
			t.Fatalf("phase=%q got=(%q,%q) want=(%q,%q)", test.phase, label, command, test.label, test.want)
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
