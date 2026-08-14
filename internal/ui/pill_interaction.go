package ui

import "time"

const (
	pillAutoCollapse      = 6 * time.Second
	pillAnimationDuration = 220 * time.Millisecond
	pillCollapsedWidth    = float32(330)
	pillCollapsedHeight   = float32(42)
	pillExpandedWidth     = float32(520)
	pillExpandedHeight    = float32(88)
	pillControlsReadyAt   = float32(0.98)
	pillControlsArmDelay  = 420 * time.Millisecond
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
	buttonY := y + (height-34)/2
	stopMaxX := x + width - 13
	stopMinX := stopMaxX - 38
	primaryMaxX := stopMinX - 6
	primaryMinX := primaryMaxX - 38
	configMaxX := primaryMinX - 6
	configMinX := configMaxX - 34
	return pillLayout{
		body: pillRect{Min: pillPoint{X: x, Y: y}, Max: pillPoint{X: x + width, Y: y + height}},
		config: pillRect{
			Min: pillPoint{X: configMinX, Y: buttonY},
			Max: pillPoint{X: configMaxX, Y: buttonY + 34},
		},
		primary: pillRect{
			Min: pillPoint{X: primaryMinX, Y: buttonY},
			Max: pillPoint{X: primaryMaxX, Y: buttonY + 34},
		},
		stop: pillRect{
			Min: pillPoint{X: stopMinX, Y: buttonY},
			Max: pillPoint{X: stopMaxX, Y: buttonY + 34},
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
}

func (state *pillPointerState) update(point pillPoint, down, clicked, released bool, layout pillLayout) pillHitTarget {
	validPoint := point.X >= 0 && point.Y >= 0
	if down && validPoint {
		target := layout.hit(point)
		if !state.tracking && target != pillHitNone {
			state.tracking = true
			state.downTarget = target
		}
		if state.tracking {
			state.lastPoint = point
			state.hasPoint = true
		}
	} else if clicked && validPoint {
		state.tracking = true
		state.downTarget = layout.hit(point)
		state.lastPoint = point
		state.hasPoint = true
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
		_, commandType := pillPrimaryAction(frame.Status.Phase)
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

func pillPrimaryAction(phase string) (string, CommandType) {
	switch phase {
	case "running", "waiting":
		return "暂停", CommandPause
	case "paused":
		return "继续", CommandResume
	default:
		return "开始", CommandStart
	}
}
