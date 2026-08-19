package ui

import (
	"fmt"
	"time"
)

const (
	pillAutoCollapse      = 6 * time.Second
	pillAnimationDuration = 220 * time.Millisecond
	pillCollapsedWidth    = float32(330)
	pillCollapsedHeight   = float32(56)
	pillExpandedWidth     = float32(520)
	pillExpandedHeight    = float32(112)
	pillButtonSize        = float32(52)
	pillButtonGap         = float32(8)
	pillControlsReadyAt   = float32(0.98)
	pillControlsArmDelay  = 420 * time.Millisecond
	pillConfirmHold       = 800 * time.Millisecond
	exitArmWindow         = 2 * time.Second
)

type pillPoint struct {
	X float32
	Y float32
}

type pillRect struct {
	Min pillPoint
	Max pillPoint
}

func (r pillRect) contains(point pillPoint) bool {
	return point.X >= r.Min.X && point.X <= r.Max.X && point.Y >= r.Min.Y && point.Y <= r.Max.Y
}

type pillHitTarget uint8

const (
	pillHitNone pillHitTarget = iota
	pillHitExpand
	pillHitConfig
	pillHitPrimary
	pillHitStop
)

type pillLayout struct {
	body    pillRect
	expand  pillRect
	config  pillRect
	primary pillRect
	stop    pillRect
}

func collapsedPillLayout(x, y float32) pillLayout {
	body := pillRect{Min: pillPoint{X: x, Y: y}, Max: pillPoint{X: x + pillCollapsedWidth, Y: y + pillCollapsedHeight}}
	return pillLayout{body: body, expand: body}
}

func expandedPillLayout(x, y float32) pillLayout {
	return expandedPillLayoutForBounds(x, y, pillExpandedWidth, pillExpandedHeight)
}

// expandedPillLayoutForBounds keeps the compact controls anchored to the right
// edge, leaving the larger left area available for runtime information.
func expandedPillLayoutForBounds(x, y, width, height float32) pillLayout {
	buttonY := y + (height-pillButtonSize)/2
	stopMaxX := x + width - 13
	stopMinX := stopMaxX - pillButtonSize
	primaryMaxX := stopMinX - pillButtonGap
	primaryMinX := primaryMaxX - pillButtonSize
	configMaxX := primaryMinX - pillButtonGap
	configMinX := configMaxX - pillButtonSize
	return pillLayout{
		body: pillRect{Min: pillPoint{X: x, Y: y}, Max: pillPoint{X: x + width, Y: y + height}},
		config: pillRect{
			Min: pillPoint{X: configMinX, Y: buttonY},
			Max: pillPoint{X: configMaxX, Y: buttonY + pillButtonSize},
		},
		primary: pillRect{
			Min: pillPoint{X: primaryMinX, Y: buttonY},
			Max: pillPoint{X: primaryMaxX, Y: buttonY + pillButtonSize},
		},
		stop: pillRect{
			Min: pillPoint{X: stopMinX, Y: buttonY},
			Max: pillPoint{X: stopMaxX, Y: buttonY + pillButtonSize},
		},
	}
}

// pillInteractionLayout deliberately strips the invisible expanded controls
// while the pill is collapsed. Without this separation, the button rectangles
// win hit-testing over the full-body expand target even though no buttons are
// visible yet.
func pillInteractionLayout(expanded bool, visual pillLayout) pillLayout {
	if expanded {
		return visual
	}
	return pillLayout{body: visual.body, expand: visual.body}
}

func pillControlsEnabled(frame *panelFrame, now time.Time) bool {
	return frame.PillExpanded && frame.PillExpansion >= pillControlsReadyAt &&
		!frame.PillControlsAt.IsZero() && !now.Before(frame.PillControlsAt)
}

func advancePillAnimation(frame *panelFrame, now time.Time) {
	if frame.PillAnimationAt.IsZero() {
		frame.PillAnimationAt = now
		return
	}
	elapsed := now.Sub(frame.PillAnimationAt)
	if elapsed <= 0 {
		return
	}
	frame.PillAnimationAt = now
	delta := float32(elapsed) / float32(pillAnimationDuration)
	if frame.PillExpanded {
		frame.PillExpansion += delta
		if frame.PillExpansion > 1 {
			frame.PillExpansion = 1
		}
		return
	}
	frame.PillExpansion -= delta
	if frame.PillExpansion < 0 {
		frame.PillExpansion = 0
	}
}

func (layout pillLayout) hit(point pillPoint) pillHitTarget {
	switch {
	case layout.config.contains(point):
		return pillHitConfig
	case layout.primary.contains(point):
		return pillHitPrimary
	case layout.stop.contains(point):
		return pillHitStop
	case layout.expand.contains(point):
		return pillHitExpand
	default:
		return pillHitNone
	}
}

type pillPointerState struct {
	tracking   bool
	downTarget pillHitTarget
	lastPoint  pillPoint
	hasPoint   bool
	holdSince  time.Time
}

func stopConfirmed(downFor time.Duration) bool {
	return downFor >= pillConfirmHold
}

func confirmExit(frame *panelFrame, now time.Time, clicked bool) bool {
	if !clicked {
		if !frame.ExitArmedUntil.IsZero() && !now.Before(frame.ExitArmedUntil) {
			frame.ExitArmedUntil = time.Time{}
		}
		return false
	}
	if !frame.ExitArmedUntil.IsZero() && now.Before(frame.ExitArmedUntil) {
		frame.ExitArmedUntil = time.Time{}
		return true
	}
	frame.ExitArmedUntil = now.Add(exitArmWindow)
	return false
}

func (state *pillPointerState) update(point pillPoint, down, clicked, released bool, layout pillLayout, now time.Time) pillHitTarget {
	validPoint := point.X >= 0 && point.Y >= 0
	if down && validPoint {
		target := layout.hit(point)
		if !state.tracking && target != pillHitNone {
			state.tracking = true
			state.downTarget = target
			if target == pillHitStop {
				state.holdSince = now
			}
		}
		if state.tracking {
			state.lastPoint = point
			state.hasPoint = true
			if state.downTarget == pillHitStop && target == pillHitStop && stopConfirmed(now.Sub(state.holdSince)) {
				state.tracking = false
				state.downTarget = pillHitNone
				state.holdSince = time.Time{}
				state.lastPoint = pillPoint{}
				state.hasPoint = false
				return pillHitStop
			}
		}
	} else if clicked && validPoint {
		state.tracking = true
		state.downTarget = layout.hit(point)
		state.lastPoint = point
		state.hasPoint = true
		if state.downTarget == pillHitStop {
			state.holdSince = now
		}
	}
	if !released {
		return pillHitNone
	}

	releasePoint := point
	if !validPoint && state.hasPoint {
		releasePoint = state.lastPoint
	}
	releasedTarget := layout.hit(releasePoint)
	pressedTarget := state.downTarget
	wasTracking := state.tracking
	state.tracking = false
	state.downTarget = pillHitNone
	state.lastPoint = pillPoint{}
	state.hasPoint = false
	state.holdSince = time.Time{}
	if pressedTarget == pillHitStop || releasedTarget == pillHitStop {
		return pillHitNone
	}
	if !wasTracking {
		// AutoGo can expose only the release frame for a short emulator tap.
		return releasedTarget
	}
	if pressedTarget == releasedTarget {
		return releasedTarget
	}
	return pillHitNone
}

func applyPillHit(frame *panelFrame, hit pillHitTarget, commands []Command) []Command {
	switch hit {
	case pillHitExpand:
		now := time.Now()
		frame.PillExpanded = true
		frame.PillControlsAt = now.Add(pillControlsArmDelay)
		frame.PillCollapseAt = now.Add(pillAutoCollapse)
	case pillHitConfig:
		frame.Compact = false
		collapsePill(frame)
	case pillHitPrimary:
		_, commandType, enabled := pillPrimaryAction(frame.Status.Phase, frame.Status.Outcome)
		if !enabled {
			return commands
		}
		command := Command{Type: commandType}
		if commandType == CommandStart {
			cfg := cloneDraft(frame.Draft)
			command.Settings = &cfg
		}
		commands = append(commands, command)
		collapsePill(frame)
	case pillHitStop:
		commands = append(commands, Command{Type: CommandStop})
		collapsePill(frame)
	}
	return commands
}

func collapsePill(frame *panelFrame) {
	frame.PillExpanded = false
	frame.PillControlsAt = time.Time{}
	frame.PillCollapseAt = time.Time{}
	frame.PillPointer = pillPointerState{}
}

func refreshPillCollapse(frame *panelFrame, now time.Time, pointerOverBody bool) {
	if !frame.PillExpanded || !pointerOverBody {
		return
	}
	frame.PillCollapseAt = now.Add(pillAutoCollapse)
}

func CompactPillHeadline(status RuntimeStatus, logs []string) string {
	if status.Message != "" {
		return status.Message
	}
	if len(logs) > 0 {
		return logs[len(logs)-1]
	}
	scene := status.Scene
	if scene == "" {
		scene = "unknown"
	}
	outcome := status.Outcome
	if outcome == "" {
		outcome = "idle"
	}
	return fmt.Sprintf("%s · %s · 动作 %d", scene, outcome, status.ActionCount)
}

func pillPrimaryAction(phase, outcome string) (string, CommandType, bool) {
	if outcome == "scheduled_wait" {
		return "等待", CommandPause, false
	}
	switch phase {
	case "running", "waiting":
		return "暂停", CommandPause, true
	case "paused":
		return "继续", CommandResume, true
	default:
		return "开始", CommandStart, true
	}
}

func pillExpandedState(phase, outcome string) string {
	if outcome == "scheduled_wait" {
		return "等待计划时段"
	}
	switch phase {
	case "running", "waiting":
		return "自动生产运行中"
	case "paused":
		return "脚本已暂停"
	case "error":
		return "脚本运行异常"
	default:
		return "脚本已停止"
	}
}
