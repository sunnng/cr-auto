//go:build android && cgo

package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dasongzi1366/AutoGo/imgui"
)

var (
	colorGold    = imgui.Vec4{X: 0.91, Y: 0.70, Z: 0.30, W: 1}
	colorCream   = imgui.Vec4{X: 0.95, Y: 0.90, Z: 0.82, W: 1}
	colorMuted   = imgui.Vec4{X: 0.65, Y: 0.59, Z: 0.53, W: 1}
	colorGreen   = imgui.Vec4{X: 0.38, Y: 0.81, Z: 0.57, W: 1}
	colorWarning = imgui.Vec4{X: 0.98, Y: 0.73, Z: 0.27, W: 1}
	colorRed     = imgui.Vec4{X: 0.96, Y: 0.43, Z: 0.43, W: 1}
	colorDarkInk = imgui.Vec4{X: 0.16, Y: 0.11, Z: 0.06, W: 1}
)

type themeColor struct {
	index imgui.Col
	value imgui.Vec4
}

const panelWindowID = "饼干王国自动化控制台###auto-cookie-shell"

const pillWindowID = "##auto-cookie-pill-raw"

var (
	detectionPreviewTexture         *imgui.Texture
	detectionPreviewTextureRevision uint64
)

func startPanelRenderer(panel *Panel) error {
	if err := imgui.Init(); err != nil {
		return err
	}
	imgui.Run(panel.renderFrame)
	return nil
}

func stopPanelRenderer() {
	if detectionPreviewTexture != nil {
		detectionPreviewTexture.Delete()
		detectionPreviewTexture = nil
	}
	detectionPreviewTextureRevision = 0
	imgui.Close()
}

func (p *Panel) renderFrame() {
	frame, ok := p.readFrame()
	if !ok {
		return
	}
	if frame.CaptureHidden {
		renderCapturePlaceholder()
		p.markCaptureReady(frame.CaptureRevision)
		return
	}

	colorCount := pushKingdomTheme()
	// Compacting transforms the existing panel window into the pill instead of
	// destroying the panel and creating an unrelated window. Keeping the same
	// ImGui ID preserves the Android overlay's pointer input channel.
	var commands []Command
	commands = renderPill(&frame)
	if !frame.Compact {
		commands = append(commands, renderConfigWindow(&frame)...)
	}
	imgui.PopStyleColorV(colorCount)

	p.writeFrame(frame)
	for _, command := range commands {
		p.emit(command)
	}
}

func renderCapturePlaceholder() {
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize |
		imgui.WindowFlagsNoMove | imgui.WindowFlagsNoScrollbar |
		imgui.WindowFlagsNoScrollWithMouse | imgui.WindowFlagsNoBackground |
		imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoInputs
	for _, windowID := range []string{panelWindowID, pillWindowID} {
		renderTransparentWindow(windowID, flags)
	}
}

func renderTransparentWindow(windowID string, flags imgui.WindowFlags) {
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 1, Y: 1}, imgui.CondAlways)
	imgui.SetNextWindowPosV(imgui.Vec2{X: -100, Y: -100}, imgui.CondAlways, imgui.Vec2{})
	imgui.SetNextWindowCollapsedV(false, imgui.CondAlways)
	imgui.SetNextWindowBgAlpha(0)
	imgui.BeginV(windowID, nil, flags)
	imgui.End()
}

func syncDetectionPreviewTexture(preview DetectionPreview) {
	if preview.Image == nil {
		if detectionPreviewTexture != nil {
			detectionPreviewTexture.Delete()
			detectionPreviewTexture = nil
		}
		detectionPreviewTextureRevision = 0
		return
	}
	if detectionPreviewTexture != nil && detectionPreviewTextureRevision == preview.Revision {
		return
	}
	if detectionPreviewTexture != nil {
		detectionPreviewTexture.Delete()
	}
	detectionPreviewTexture = imgui.CreateTextureNrgba(preview.Image)
	detectionPreviewTextureRevision = preview.Revision
}

func renderConfigWindow(frame *panelFrame) []Command {
	styleCount := pushKingdomGeometry()
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 1160, Y: 780}, imgui.CondAlways)
	imgui.SetNextWindowPosV(imgui.Vec2{X: 220, Y: 55}, imgui.CondAlways, imgui.Vec2{})
	imgui.SetNextWindowCollapsedV(false, imgui.CondAlways)

	flags := imgui.WindowFlagsNoMove | imgui.WindowFlagsNoResize |
		imgui.WindowFlagsNoScrollbar | imgui.WindowFlagsNoScrollWithMouse |
		imgui.WindowFlagsNoCollapse
	visible := imgui.BeginV(panelWindowID, nil, flags)
	var commands []Command
	if visible {
		frame.ActiveTab = int32(NormalizeConfigTab(ConfigTab(frame.ActiveTab)))
		if renderTitleBarMinimizeButton() {
			minimizeToPill(frame)
		}
		// AutoGo's PushFont wrapper cannot accept nil like native Dear ImGui can.
		// Scale the panel window instead; child windows inherit this value.
		imgui.SetWindowFontScale(0.86)
		if frame.ActiveTab == 3 {
			syncDetectionPreviewTexture(frame.Preview)
		}
		renderHeader(frame.Status)
		imgui.Separator()

		available := imgui.ContentRegionAvail()
		bodyHeight := available.Y - 92
		if bodyHeight < 360 {
			bodyHeight = 360
		}

		imgui.PushStyleColorVec4(imgui.ColChildBg, imgui.Vec4{X: 0.12, Y: 0.09, Z: 0.07, W: 1})
		if imgui.BeginChildStrV("kingdom-sidebar", imgui.Vec2{X: 210, Y: bodyHeight}, imgui.ChildFlagsBorders|imgui.ChildFlagsAlwaysUseWindowPadding, imgui.WindowFlagsNoScrollbar) {
			renderSidebar(frame)
		}
		imgui.EndChild()
		imgui.PopStyleColor()

		imgui.SameLine()
		if imgui.BeginChildStrV("kingdom-content", imgui.Vec2{X: 0, Y: bodyHeight}, imgui.ChildFlagsBorders|imgui.ChildFlagsAlwaysUseWindowPadding, 0) {
			commands = renderContent(frame)
		}
		imgui.EndChild()

		imgui.Separator()
		commands = append(commands, renderFooter(frame)...)
	}
	imgui.End()
	imgui.PopStyleVarV(styleCount)
	return commands
}

func minimizeToPill(frame *panelFrame) {
	frame.Compact = true
	collapsePill(frame)
}

func renderPill(frame *panelFrame) []Command {
	now := time.Now()
	advancePillAnimation(frame, now)
	progress := pillEase(frame.PillExpansion)
	width := pillLerp(pillCollapsedWidth, pillExpandedWidth, progress)
	height := pillLerp(pillCollapsedHeight, pillExpandedHeight, progress)
	windowX := (1600 - width) / 2
	windowY := float32(16)
	size := imgui.Vec2{X: width, Y: height}
	visualLayout := expandedPillLayoutForBounds(windowX, windowY, width, height)
	interactionLayout := pillInteractionLayout(pillControlsEnabled(frame, now), visualLayout)

	imgui.PushStyleColorVec4(imgui.ColWindowBg, imgui.Vec4{})
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{})
	imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 0)
	rounding := pillLerp(22, 24, progress)
	imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, rounding)
	imgui.SetNextWindowSizeV(size, imgui.CondAlways)
	imgui.SetNextWindowPosV(imgui.Vec2{X: windowX, Y: windowY}, imgui.CondAlways, imgui.Vec2{})
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoScrollbar | imgui.WindowFlagsNoBackground | imgui.WindowFlagsNoSavedSettings
	visible := imgui.BeginV(pillWindowID, nil, flags)
	var commands []Command
	if visible {
		imgui.SetWindowFontScale(0.60)
		drawList := imgui.WindowDrawList()
		mouse := imgui.MousePos()
		point := pillPoint{X: mouse.X, Y: mouse.Y}
		hovered := visualLayout.body.contains(point)
		hit := interactionLayout.hit(point)
		drawPillBody(drawList, visualLayout.body, rounding, hovered || frame.PillExpanded)
		collapsedAlpha := pillClamp01(1 - progress/0.58)
		expandedAlpha := pillClamp01((progress - 0.18) / 0.62)
		if collapsedAlpha > 0 {
			drawCollapsedPillContent(drawList, frame, visualLayout.body, collapsedAlpha)
		}
		if expandedAlpha > 0 {
			drawExpandedPillContent(drawList, frame, visualLayout, hit, expandedAlpha)
		}

		if frame.PillExpanded && !frame.PillCollapseAt.IsZero() && now.After(frame.PillCollapseAt) &&
			!imgui.IsMouseDown(imgui.MouseButtonLeft) {
			collapsePill(frame)
		}

		released := imgui.IsMouseReleased(imgui.MouseButtonLeft)
		action := frame.PillPointer.update(
			point,
			imgui.IsMouseDown(imgui.MouseButtonLeft),
			imgui.IsMouseClickedBoolV(imgui.MouseButtonLeft, false),
			released,
			interactionLayout,
		)
		commands = applyPillHit(frame, action, commands)
		if frame.PillExpanded && released && action == pillHitNone {
			collapsePill(frame)
		}
	}
	imgui.End()
	imgui.PopStyleVarV(3)
	imgui.PopStyleColor()
	return commands
}

func drawCollapsedPillContent(drawList *imgui.DrawList, frame *panelFrame, bounds pillRect, alpha float32) {
	centerY := bounds.Min.Y + (bounds.Max.Y-bounds.Min.Y)/2
	dotCenter := imgui.Vec2{X: bounds.Min.X + 19, Y: centerY}
	statusColor := pillStatusColor(frame.Status.Phase)
	drawList.AddCircleFilled(dotCenter, 8, imgui.ColorU32Vec4(withPillAlpha(statusColor, alpha*0.13)))
	drawList.AddCircleFilled(dotCenter, 4, imgui.ColorU32Vec4(withPillAlpha(statusColor, alpha)))

	count := fmt.Sprintf("%d/%d", frame.Status.ActionCount, frame.Draft.Safety.MaxActionsPerRun)
	countSize := imgui.CalcTextSize(count)
	textY := bounds.Min.Y + (bounds.Max.Y-bounds.Min.Y-countSize.Y)/2 - 1.5
	countX := bounds.Max.X - 15 - countSize.X
	drawList.AddTextVec2(imgui.Vec2{X: countX, Y: textY}, imgui.ColorU32Vec4(withPillAlpha(imgui.Vec4{X: 0.95, Y: 0.76, Z: 0.37, W: 1}, alpha)), count)

	logLine := compactPillLogLimit(frame, 22)
	drawList.AddTextVec2(imgui.Vec2{X: bounds.Min.X + 33, Y: textY}, imgui.ColorU32Vec4(withPillAlpha(colorCream, alpha)), logLine)
}

func drawExpandedPillContent(drawList *imgui.DrawList, frame *panelFrame, layout pillLayout, hovered pillHitTarget, alpha float32) {
	state := pillExpandedState(frame.Status.Phase)
	statusColor := pillStatusColor(frame.Status.Phase)
	infoX := layout.body.Min.X + 17
	stateY := layout.body.Min.Y + 12
	sceneY := layout.body.Min.Y + 36
	detailY := layout.body.Min.Y + 59

	// The left two-thirds is deliberately information-first. The status dot
	// makes the state scannable without adding another control-like badge.
	drawList.AddCircleFilled(
		imgui.Vec2{X: infoX + 4, Y: stateY + 8},
		4,
		imgui.ColorU32Vec4(withPillAlpha(statusColor, alpha)),
	)
	drawList.AddTextVec2(imgui.Vec2{X: infoX + 14, Y: stateY}, imgui.ColorU32Vec4(withPillAlpha(statusColor, alpha)), state)

	sceneLine := fmt.Sprintf("场景 %s  ·  %s", fallback(frame.Status.Scene, "未知"), fallback(frame.Status.Outcome, "等待"))
	drawList.AddTextVec2(imgui.Vec2{X: infoX, Y: sceneY}, imgui.ColorU32Vec4(withPillAlpha(colorMuted, alpha)), limitPillText(sceneLine, 27))
	drawList.AddTextVec2(imgui.Vec2{X: infoX, Y: detailY}, imgui.ColorU32Vec4(withPillAlpha(colorCream, alpha)), expandedPillDetail(frame))

	count := fmt.Sprintf("动作 %d/%d", frame.Status.ActionCount, frame.Draft.Safety.MaxActionsPerRun)
	countSize := imgui.CalcTextSize(count)
	countX := layout.config.Min.X - 18 - countSize.X
	drawList.AddTextVec2(imgui.Vec2{X: countX, Y: stateY}, imgui.ColorU32Vec4(withPillAlpha(colorGold, alpha)), count)

	// A subtle divider separates passive runtime information from the compact
	// controls, so the controls remain available without dominating the pill.
	dividerX := layout.config.Min.X - 12
	drawList.AddLineV(
		imgui.Vec2{X: dividerX, Y: layout.body.Min.Y + 14},
		imgui.Vec2{X: dividerX, Y: layout.body.Max.Y - 14},
		imgui.ColorU32Vec4(withPillAlpha(imgui.Vec4{X: 0.72, Y: 0.50, Z: 0.20, W: 0.42}, alpha)),
		1,
	)

	primaryIcon := pillIconPlay
	if frame.Status.Phase == "running" || frame.Status.Phase == "waiting" {
		primaryIcon = pillIconPause
	}
	drawPillButton(drawList, layout.config, pillIconConfig, hovered == pillHitConfig, pillButtonConfig, alpha)
	drawPillButton(drawList, layout.primary, primaryIcon, hovered == pillHitPrimary, pillButtonToggle, alpha)
	drawPillButton(drawList, layout.stop, pillIconStop, hovered == pillHitStop, pillButtonStop, alpha)
}

func drawPillBody(drawList *imgui.DrawList, bounds pillRect, rounding float32, highlighted bool) {
	background := imgui.Vec4{X: 0.075, Y: 0.055, Z: 0.045, W: 0.96}
	if highlighted {
		background = imgui.Vec4{X: 0.095, Y: 0.068, Z: 0.048, W: 0.98}
	}
	minimum, maximum := pillRectVec(bounds)
	drawList.AddRectFilledV(minimum, maximum, imgui.ColorU32Vec4(background), rounding, imgui.DrawFlagsRoundCornersAll)
	drawList.AddRectV(minimum, maximum, imgui.ColorU32Vec4(imgui.Vec4{X: 0.72, Y: 0.50, Z: 0.20, W: 0.94}), rounding, imgui.DrawFlagsRoundCornersAll, 1.2)
	drawList.AddLineV(
		imgui.Vec2{X: minimum.X + rounding, Y: minimum.Y + 1},
		imgui.Vec2{X: maximum.X - rounding, Y: minimum.Y + 1},
		imgui.ColorU32Vec4(imgui.Vec4{X: 1, Y: 1, Z: 1, W: 0.06}),
		1,
	)
}

type pillButtonKind uint8

const (
	pillButtonConfig pillButtonKind = iota
	pillButtonToggle
	pillButtonStop
)

type pillIcon uint8

const (
	pillIconConfig pillIcon = iota
	pillIconPlay
	pillIconPause
	pillIconStop
)

func drawPillButton(drawList *imgui.DrawList, bounds pillRect, icon pillIcon, hovered bool, kind pillButtonKind, alpha float32) {
	background := imgui.Vec4{X: 0.29, Y: 0.20, Z: 0.13, W: 1}
	textColor := colorCream
	switch kind {
	case pillButtonConfig:
		background = imgui.Vec4{X: 0.24, Y: 0.18, Z: 0.12, W: 1}
		textColor = colorMuted
	case pillButtonToggle:
		background = imgui.Vec4{X: 0.87, Y: 0.63, Z: 0.20, W: 1}
		textColor = colorDarkInk
	case pillButtonStop:
		background = imgui.Vec4{X: 0.38, Y: 0.17, Z: 0.15, W: 1}
		textColor = imgui.Vec4{X: 1, Y: 0.87, Z: 0.87, W: 1}
	}
	if hovered {
		background.X = pillClamp01(background.X + 0.08)
		background.Y = pillClamp01(background.Y + 0.06)
		background.Z = pillClamp01(background.Z + 0.04)
	}
	minimum, maximum := pillRectVec(bounds)
	drawList.AddRectFilledV(minimum, maximum, imgui.ColorU32Vec4(withPillAlpha(background, alpha)), 10, imgui.DrawFlagsRoundCornersAll)
	drawPillIcon(drawList, bounds, icon, withPillAlpha(textColor, alpha), withPillAlpha(background, alpha))
}

func drawPillIcon(drawList *imgui.DrawList, bounds pillRect, icon pillIcon, color, background imgui.Vec4) {
	center := imgui.Vec2{
		X: (bounds.Min.X + bounds.Max.X) / 2,
		Y: (bounds.Min.Y + bounds.Max.Y) / 2,
	}
	iconColor := imgui.ColorU32Vec4(color)
	switch icon {
	case pillIconConfig:
		// Draw a small gear without relying on a font glyph. The fixed spokes
		// keep it visually consistent with the play/pause/stop primitives.
		spokes := [...]struct{ X, Y float32 }{
			{0, -8}, {5.7, -5.7}, {8, 0}, {5.7, 5.7},
			{0, 8}, {-5.7, 5.7}, {-8, 0}, {-5.7, -5.7},
		}
		for _, spoke := range spokes {
			start := imgui.Vec2{X: center.X + spoke.X*0.62, Y: center.Y + spoke.Y*0.62}
			end := imgui.Vec2{X: center.X + spoke.X, Y: center.Y + spoke.Y}
			drawList.AddLineV(start, end, iconColor, 2)
		}
		drawList.AddCircleV(center, 5.1, iconColor, 12, 2)
		drawList.AddCircleFilled(center, 2, imgui.ColorU32Vec4(background))
	case pillIconPlay:
		drawList.AddTriangleFilled(
			imgui.Vec2{X: center.X - 4, Y: center.Y - 7},
			imgui.Vec2{X: center.X - 4, Y: center.Y + 7},
			imgui.Vec2{X: center.X + 7, Y: center.Y},
			iconColor,
		)
	case pillIconPause:
		drawList.AddRectFilledV(
			imgui.Vec2{X: center.X - 6, Y: center.Y - 7},
			imgui.Vec2{X: center.X - 2, Y: center.Y + 7},
			iconColor,
			2,
			imgui.DrawFlagsRoundCornersAll,
		)
		drawList.AddRectFilledV(
			imgui.Vec2{X: center.X + 2, Y: center.Y - 7},
			imgui.Vec2{X: center.X + 6, Y: center.Y + 7},
			iconColor,
			2,
			imgui.DrawFlagsRoundCornersAll,
		)
	case pillIconStop:
		drawList.AddRectFilledV(
			imgui.Vec2{X: center.X - 6, Y: center.Y - 6},
			imgui.Vec2{X: center.X + 6, Y: center.Y + 6},
			iconColor,
			2,
			imgui.DrawFlagsRoundCornersAll,
		)
	}
}

func drawCenteredPillText(drawList *imgui.DrawList, bounds pillRect, label string, color imgui.Vec4) {
	textSize := imgui.CalcTextSize(label)
	position := imgui.Vec2{
		X: bounds.Min.X + (bounds.Max.X-bounds.Min.X-textSize.X)/2,
		Y: bounds.Min.Y + (bounds.Max.Y-bounds.Min.Y-textSize.Y)/2 - 1.5,
	}
	drawList.AddTextVec2(position, imgui.ColorU32Vec4(color), label)
}

func pillRectVec(bounds pillRect) (imgui.Vec2, imgui.Vec2) {
	return imgui.Vec2{X: bounds.Min.X, Y: bounds.Min.Y}, imgui.Vec2{X: bounds.Max.X, Y: bounds.Max.Y}
}

func compactPillLogLimit(frame *panelFrame, limit int) string {
	logLine := fmt.Sprintf("%s · %s · %d", fallback(frame.Status.Scene, "unknown"), fallback(frame.Status.Outcome, "idle"), frame.Status.ActionCount)
	if len(frame.Logs) > 0 {
		index := int(time.Now().Unix()/3) % len(frame.Logs)
		logLine = frame.Logs[index]
	}
	return limitPillText(logLine, limit)
}

func expandedPillDetail(frame *panelFrame) string {
	detail := frame.Status.Message
	if detail == "" && len(frame.Logs) > 0 {
		detail = frame.Logs[len(frame.Logs)-1]
	}
	if detail == "" {
		detail = "等待运行指令"
	}
	return limitPillText(detail, 31)
}

func limitPillText(value string, limit int) string {
	if limit < 2 {
		return ""
	}
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit-1]) + "…"
	}
	return value
}

func pillExpandedState(phase string) string {
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

func pillEase(value float32) float32 {
	value = pillClamp01(value)
	return value * value * (3 - 2*value)
}

func pillLerp(start, end, progress float32) float32 {
	return start + (end-start)*progress
}

func pillClamp01(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func withPillAlpha(color imgui.Vec4, alpha float32) imgui.Vec4 {
	color.W *= pillClamp01(alpha)
	return color
}

func pillStatusColor(phase string) imgui.Vec4 {
	switch phase {
	case "running":
		return colorGreen
	case "paused":
		return colorWarning
	case "error":
		return colorRed
	default:
		return colorGold
	}
}

func pushKingdomTheme() int32 {
	colors := []themeColor{
		{imgui.ColText, colorCream},
		{imgui.ColTextDisabled, colorMuted},
		{imgui.ColWindowBg, imgui.Vec4{X: 0.10, Y: 0.075, Z: 0.06, W: 0.98}},
		{imgui.ColChildBg, imgui.Vec4{X: 0.14, Y: 0.105, Z: 0.08, W: 1}},
		{imgui.ColPopupBg, imgui.Vec4{X: 0.14, Y: 0.105, Z: 0.08, W: 1}},
		{imgui.ColBorder, imgui.Vec4{X: 0.34, Y: 0.26, Z: 0.18, W: 1}},
		{imgui.ColBorderShadow, imgui.Vec4{X: 0, Y: 0, Z: 0, W: 0.35}},
		{imgui.ColFrameBg, imgui.Vec4{X: 0.20, Y: 0.15, Z: 0.11, W: 1}},
		{imgui.ColFrameBgHovered, imgui.Vec4{X: 0.28, Y: 0.21, Z: 0.14, W: 1}},
		{imgui.ColFrameBgActive, imgui.Vec4{X: 0.34, Y: 0.25, Z: 0.15, W: 1}},
		{imgui.ColTitleBg, imgui.Vec4{X: 0.16, Y: 0.11, Z: 0.075, W: 1}},
		{imgui.ColTitleBgActive, imgui.Vec4{X: 0.24, Y: 0.17, Z: 0.095, W: 1}},
		{imgui.ColTitleBgCollapsed, imgui.Vec4{X: 0.16, Y: 0.11, Z: 0.075, W: 1}},
		{imgui.ColScrollbarBg, imgui.Vec4{X: 0.10, Y: 0.075, Z: 0.06, W: 0.7}},
		{imgui.ColScrollbarGrab, imgui.Vec4{X: 0.39, Y: 0.29, Z: 0.18, W: 1}},
		{imgui.ColScrollbarGrabHovered, imgui.Vec4{X: 0.55, Y: 0.40, Z: 0.22, W: 1}},
		{imgui.ColScrollbarGrabActive, imgui.Vec4{X: 0.72, Y: 0.52, Z: 0.25, W: 1}},
		{imgui.ColCheckMark, colorGold},
		{imgui.ColSliderGrab, imgui.Vec4{X: 0.78, Y: 0.56, Z: 0.25, W: 1}},
		{imgui.ColSliderGrabActive, colorGold},
		{imgui.ColButton, imgui.Vec4{X: 0.24, Y: 0.18, Z: 0.12, W: 1}},
		{imgui.ColButtonHovered, imgui.Vec4{X: 0.35, Y: 0.26, Z: 0.16, W: 1}},
		{imgui.ColButtonActive, imgui.Vec4{X: 0.47, Y: 0.34, Z: 0.18, W: 1}},
		{imgui.ColHeader, imgui.Vec4{X: 0.34, Y: 0.24, Z: 0.13, W: 1}},
		{imgui.ColHeaderHovered, imgui.Vec4{X: 0.46, Y: 0.33, Z: 0.17, W: 1}},
		{imgui.ColHeaderActive, imgui.Vec4{X: 0.57, Y: 0.41, Z: 0.20, W: 1}},
		{imgui.ColSeparator, imgui.Vec4{X: 0.34, Y: 0.26, Z: 0.18, W: 0.85}},
		{imgui.ColSeparatorHovered, imgui.Vec4{X: 0.62, Y: 0.45, Z: 0.22, W: 1}},
		{imgui.ColSeparatorActive, colorGold},
	}
	for _, color := range colors {
		imgui.PushStyleColorVec4(color.index, color.value)
	}
	return int32(len(colors))
}

func pushKingdomGeometry() int32 {
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{X: 20, Y: 16})
	imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 14)
	imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 1)
	imgui.PushStyleVarFloat(imgui.StyleVarChildRounding, 10)
	imgui.PushStyleVarFloat(imgui.StyleVarChildBorderSize, 1)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{X: 12, Y: 8})
	imgui.PushStyleVarFloat(imgui.StyleVarFrameRounding, 7)
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{X: 10, Y: 9})
	imgui.PushStyleVarFloat(imgui.StyleVarScrollbarRounding, 8)
	imgui.PushStyleVarFloat(imgui.StyleVarGrabRounding, 7)
	return 10
}

const (
	panelTitleBarButtonWidth  = float32(42)
	panelTitleBarButtonHeight = float32(34)
	panelTitleBarButtonRight  = float32(10)
	panelTitleBarButtonTop    = float32(7)
)

func renderTitleBarMinimizeButton() bool {
	windowPos := imgui.WindowPos()
	windowSize := imgui.WindowSize()
	minimum := imgui.Vec2{
		X: windowPos.X + windowSize.X - panelTitleBarButtonRight - panelTitleBarButtonWidth,
		Y: windowPos.Y + panelTitleBarButtonTop,
	}
	maximum := imgui.Vec2{
		X: minimum.X + panelTitleBarButtonWidth,
		Y: minimum.Y + panelTitleBarButtonHeight,
	}

	mouse := imgui.MousePos()
	hovered := mouse.X >= minimum.X && mouse.X <= maximum.X &&
		mouse.Y >= minimum.Y && mouse.Y <= maximum.Y
	clicked := hovered && imgui.IsMouseClickedBoolV(imgui.MouseButtonLeft, false)

	drawList := imgui.WindowDrawList()
	drawList.PushClipRectFullScreen()
	if hovered {
		drawList.AddRectFilledV(
			minimum,
			maximum,
			imgui.ColorU32Vec4(imgui.Vec4{X: 0.35, Y: 0.26, Z: 0.16, W: 0.9}),
			7,
			imgui.DrawFlagsRoundCornersAll,
		)
	}
	centerY := minimum.Y + panelTitleBarButtonHeight/2
	drawList.AddLineV(
		imgui.Vec2{X: minimum.X + 12, Y: centerY},
		imgui.Vec2{X: maximum.X - 12, Y: centerY},
		imgui.ColorU32Vec4(colorCream),
		2.5,
	)
	drawList.PopClipRect()
	return clicked
}

func renderHeader(status RuntimeStatus) {
	colorText(colorGold, "CR AUTO")
	imgui.SameLine()
	imgui.TextUnformatted("饼干王国自动化")

	profile := "CN 1600×900  ·  240dpi"
	alignTextRight(profile, 104)
	colorText(colorMuted, profile)

	phase := phaseLabel(status.Phase)
	statusText := "● " + phase + "   场景 " + fallback(status.Scene, "unknown") + "   状态 " + fallback(status.Outcome, "configure")
	switch status.Phase {
	case "running":
		colorText(colorGreen, statusText)
	case "error":
		colorText(colorRed, statusText)
	case "paused":
		colorText(colorWarning, statusText)
	default:
		colorText(colorGold, statusText)
	}
	if status.Message != "" {
		disabledText(status.Message)
	}
}

func renderSidebar(frame *panelFrame) {
	colorText(colorGold, "配置中心")
	disabledText("生产级安全控制")
	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	items := ConfigTabs()
	for _, item := range items {
		active := ConfigTab(frame.ActiveTab) == item.ID
		textColor := colorCream
		if active {
			pushGoldButton()
			textColor = colorDarkInk
		}
		width := imgui.ContentRegionAvail().X
		clicked := centeredButton(item.Label, fmt.Sprintf("sidebar-%d", item.ID), imgui.Vec2{X: width, Y: 42}, textColor)
		if clicked {
			frame.ActiveTab = int32(item.ID)
		}
		if active {
			imgui.PopStyleColorV(4)
		}
	}
}

func renderContent(frame *panelFrame) []Command {
	frame.ActiveTab = int32(NormalizeConfigTab(ConfigTab(frame.ActiveTab)))
	tab := ConfigTab(frame.ActiveTab)
	var current ConfigTabState
	for _, state := range ConfigTabs() {
		if state.ID == tab {
			current = state
			break
		}
	}
	colorText(colorGold, "CR AUTO / SETTINGS")
	imgui.SameLine()
	imgui.TextUnformatted(current.Title)
	disabledText(current.Description)
	imgui.Separator()
	imgui.Spacing()

	switch tab {
	case ConfigTabTasks:
		renderTasks(frame)
	case ConfigTabSafety:
		renderSafety(frame)
	case ConfigTabDetection:
		return renderDetectionPreview(frame)
	default:
		renderOverview(frame)
	}
	return nil
}

func renderOverview(frame *panelFrame) {
	availableTasks := 0
	for _, task := range frame.Catalog {
		if task.Available {
			availableTasks++
		}
	}

	width := (imgui.ContentRegionAvail().X - 20) / 3
	renderStatCard("ability", "当前能力", fmt.Sprintf("%d / %d", availableTasks, len(frame.Catalog)), "开放任务", width, true)
	imgui.SameLine()
	renderStatCard("budget", "动作预算", fmt.Sprintf("%d", frame.Draft.Safety.MaxActionsPerRun), "单次运行上限", width, false)
	imgui.SameLine()
	renderStatCard("guard", "安全状态", "已锁定", "禁止付费资源", width, false)

	imgui.Spacing()
	sectionTitle("运行方式", "选择本次脚本的执行策略")
	mode := runModeIndex(frame.Draft.Run.Mode)
	modeWidth := (imgui.ContentRegionAvail().X - 20) / 3
	if modeButton("手动运行", "由你确认后启动", "manual", mode == 0, modeWidth) {
		mode = 0
	}
	imgui.SameLine()
	if modeButton("单次运行", "完成一轮后退出", "once", mode == 1, modeWidth) {
		mode = 1
	}
	imgui.SameLine()
	if modeButton("计划时段", "按时间窗口执行", "scheduled", mode == 2, modeWidth) {
		mode = 2
	}
	frame.Draft.Run.Mode = runModeAt(mode)

	if frame.Draft.Run.Mode == RunScheduled {
		imgui.Spacing()
		start := int32(frame.Draft.Run.StartMinute)
		end := int32(frame.Draft.Run.EndMinute)
		imgui.SetNextItemWidth(520)
		imgui.SliderIntV("开始分钟##schedule", &start, 0, 1439, "%d", 0)
		imgui.SetNextItemWidth(520)
		imgui.SliderIntV("结束分钟##schedule", &end, 0, 1439, "%d", 0)
		frame.Draft.Run.StartMinute = int(start)
		frame.Draft.Run.EndMinute = int(end)
		disabledText("分钟从当天 00:00 起计算，时区固定为 Asia/Shanghai。")
	}

	imgui.Spacing()
	if availableTasks == 0 {
		renderWarning("安全观察模式", "任务目录尚未配置，启动后不会点击游戏。")
	} else {
		renderSuccess("任务已就绪", fmt.Sprintf("当前有 %d 个任务可安全启用。", availableTasks))
	}
}

func renderStatCard(id, label, value, detail string, width float32, accent bool) {
	if accent {
		imgui.PushStyleColorVec4(imgui.ColChildBg, imgui.Vec4{X: 0.22, Y: 0.16, Z: 0.085, W: 1})
		imgui.PushStyleColorVec4(imgui.ColBorder, imgui.Vec4{X: 0.69, Y: 0.49, Z: 0.22, W: 1})
	}
	childFlags := imgui.ChildFlagsBorders | imgui.ChildFlagsAlwaysUseWindowPadding | imgui.ChildFlagsAutoResizeY
	if imgui.BeginChildStrV("stat-"+id, imgui.Vec2{X: width, Y: 0}, childFlags, imgui.WindowFlagsNoScrollbar) {
		disabledText(label)
		imgui.TextUnformatted(value)
		if accent {
			colorText(colorGold, detail)
		} else {
			disabledText(detail)
		}
	}
	imgui.EndChild()
	if accent {
		imgui.PopStyleColorV(2)
	}
}

func modeButton(title, detail, id string, active bool, width float32) bool {
	textColor := colorCream
	if active {
		pushGoldButton()
		textColor = colorDarkInk
	}
	clicked := centeredTwoLineButton(title, detail, "mode-"+id, imgui.Vec2{X: width, Y: 88}, textColor)
	if active {
		imgui.PopStyleColorV(4)
	}
	return clicked
}

func renderTasks(frame *panelFrame) {
	renderWarning("任务准入规则", "只有经过验证的任务才会出现在目录中。")
	imgui.Spacing()
	for _, descriptor := range frame.Catalog {
		task := frame.Draft.Tasks[descriptor.ID]
		maxAllowedRuns := descriptor.MaxRuns
		if maxAllowedRuns < 1 {
			maxAllowedRuns = 100
		}
		if task.MaxRuns == 0 {
			task.MaxRuns = 1
		}
		if task.MaxRuns > maxAllowedRuns {
			task.MaxRuns = maxAllowedRuns
		}
		if !descriptor.Available {
			task.Enabled = false
		}

		imgui.PushIDStr(descriptor.ID)
		childFlags := imgui.ChildFlagsBorders | imgui.ChildFlagsAlwaysUseWindowPadding | imgui.ChildFlagsAutoResizeY
		if imgui.BeginChildStrV("task-card", imgui.Vec2{X: 0, Y: 0}, childFlags, imgui.WindowFlagsNoScrollbar) {
			imgui.TextUnformatted(descriptor.Name)
			if descriptor.Available {
				imgui.SameLine()
				colorText(colorGreen, "● 已开放")
			} else {
				imgui.SameLine()
				colorText(colorWarning, "● 未开放")
			}
			if descriptor.MigrationStatus != "" {
				imgui.SameLine()
				colorText(colorMuted, "· "+descriptor.MigrationStatus)
			}
			wrappedMutedText(descriptor.Description)
			if !descriptor.Available && descriptor.UnavailableReason != "" {
				disabledText(descriptor.UnavailableReason)
			}

			if !descriptor.Available {
				imgui.BeginDisabledV(true)
			}
			imgui.Checkbox("启用此任务", &task.Enabled)

			priority := int32(task.Priority)
			maxRuns := int32(task.MaxRuns)
			imgui.SetNextItemWidth(480)
			imgui.SliderIntV("优先级", &priority, 0, 100, "%d", 0)
			if maxAllowedRuns == 1 {
				disabledText("单次上限：1（安全策略固定）")
				maxRuns = 1
			} else {
				imgui.SetNextItemWidth(480)
				imgui.SliderIntV("单次上限", &maxRuns, 1, int32(maxAllowedRuns), "%d", 0)
			}
			task.Priority = int(priority)
			task.MaxRuns = int(maxRuns)
			if !descriptor.Available {
				imgui.EndDisabled()
			}
		}
		imgui.EndChild()
		imgui.PopID()
		frame.Draft.Tasks[descriptor.ID] = task
		imgui.Spacing()
	}
}

func renderSafety(frame *panelFrame) {
	renderSuccess("强制安全锁已启用", "资源消费拦截与敏感页面停机不可关闭。")
	imgui.Spacing()
	sectionTitle("识别与动作限制", "降低阈值会增加误操作风险")

	confidence := frame.Draft.Safety.MinConfidence
	maxActions := int32(frame.Draft.Safety.MaxActionsPerRun)
	unknownTimeout := int32(frame.Draft.Safety.UnknownTimeoutSec)
	imgui.SetNextItemWidth(560)
	imgui.SliderFloatV("最低视觉置信度", &confidence, 0.90, 0.99, "%.2f", 0)
	imgui.SetNextItemWidth(560)
	imgui.SliderIntV("单次动作预算", &maxActions, 1, 1000, "%d", 0)
	imgui.SetNextItemWidth(560)
	imgui.SliderIntV("未知场景超时（秒）", &unknownTimeout, 5, 300, "%d", 0)
	frame.Draft.Safety.MinConfidence = confidence
	frame.Draft.Safety.MaxActionsPerRun = int(maxActions)
	frame.Draft.Safety.UnknownTimeoutSec = int(unknownTimeout)

	imgui.Spacing()
	sectionTitle("不可关闭的保护", "首个生产版本固定启用")
	frame.Draft.Safety.BlockResourceSpend = true
	frame.Draft.Safety.StopOnSensitivePage = true
	imgui.BeginDisabledV(true)
	imgui.Checkbox("禁止消耗水晶、现金与付费资源", &frame.Draft.Safety.BlockResourceSpend)
	imgui.Checkbox("进入账号、支付等敏感页面立即停止", &frame.Draft.Safety.StopOnSensitivePage)
	imgui.EndDisabled()
}

func renderDetectionPreview(frame *panelFrame) []Command {
	preview := frame.Preview
	if preview.Error != "" {
		renderWarning("识别预览失败", preview.Error)
		imgui.Spacing()
	}

	available := imgui.ContentRegionAvail()
	imageWidth := available.X * 0.62
	if imageWidth > 560 {
		imageWidth = 560
	}
	if imageWidth < 340 {
		imageWidth = 340
	}
	imageSize := imgui.Vec2{X: imageWidth - 24, Y: (imageWidth - 24) * 9 / 16}
	cardHeight := imageSize.Y + 34
	childFlags := imgui.ChildFlagsBorders | imgui.ChildFlagsAlwaysUseWindowPadding
	if imgui.BeginChildStrV("detection-preview-image", imgui.Vec2{X: imageWidth, Y: cardHeight}, childFlags, imgui.WindowFlagsNoScrollbar) {
		if detectionPreviewTexture != nil && preview.Image != nil {
			imagePos := imgui.CursorScreenPos()
			imgui.ImageV(
				detectionPreviewTexture.ID,
				imageSize,
				imgui.Vec2{X: 0, Y: 0},
				imgui.Vec2{X: 1, Y: 1},
			)
			drawDetectionOverlay(imgui.WindowDrawList(), imagePos, imageSize, preview.Detection)
		} else {
			renderWarning("暂无识别帧", "点击右侧“立即识别”采集一张只读截图。")
		}
	}
	imgui.EndChild()

	imgui.SameLine()
	// The evidence list is intentionally independently scrollable. Its height
	// is tied to the image card, while candidates, slot stats and the refresh
	// action may legitimately exceed that height on a small viewport.
	if imgui.BeginChildStrV("detection-preview-details", imgui.Vec2{X: 0, Y: cardHeight}, childFlags, 0) {
		colorText(colorGold, "识别结果")
		if preview.Image == nil {
			disabledText("尚未采集截图")
		} else {
			colorText(pillStatusColor(string(preview.Detection.Scene)), "场景："+SceneDisplayName(string(preview.Detection.Scene)))
			colorText(colorCream, fmt.Sprintf("置信度：%.1f%%", preview.Detection.Confidence*100))
			threshold := frame.Draft.Safety.MinConfidence
			if preview.Detection.Scene == SceneUnknown || preview.Detection.Confidence < threshold {
				renderWarning("不满足动作门槛", fmt.Sprintf("当前 %.2f，要求 %.2f；不会执行点击。", preview.Detection.Confidence, threshold))
			} else {
				renderSuccess("识别通过", fmt.Sprintf("当前置信度达到 %.2f 门槛。", threshold))
			}

			imgui.Spacing()
			sectionTitle("候选场景", "按颜色锚点命中率排序")
			for i, candidate := range preview.Detection.Candidates {
				if i >= 4 {
					break
				}
				TextColor := colorMuted
				if SceneID(candidate.Scene) == preview.Detection.Scene {
					TextColor = colorCream
				}
				constraint := ""
				if candidate.RequiredAnchors > 0 || candidate.ExclusionAnchors > 0 {
					constraint = fmt.Sprintf("  必需 %d/%d · 排除命中 %d", candidate.RequiredMatched, candidate.RequiredAnchors, candidate.ExclusionMatches)
					if candidate.ConstraintsPassed {
						constraint += " · 硬约束通过"
					} else {
						constraint += " · 硬约束拒绝"
					}
				}
				colorText(TextColor, fmt.Sprintf("%d. %s  %.1f%%  (%d/%d)%s", i+1, SceneDisplayName(string(candidate.Scene)), candidate.Score*100, candidate.MatchedAnchors, candidate.TotalAnchors, constraint))
			}

			matchedAnchors := 0
			positiveAnchors := 0
			requiredMatched := 0
			requiredAnchors := 0
			exclusionMatches := 0
			exclusionAnchors := 0
			for _, anchor := range preview.Detection.Anchors {
				switch anchor.Role {
				case AnchorExclusion:
					exclusionAnchors++
					if anchor.Matched {
						exclusionMatches++
					}
				default:
					positiveAnchors++
					if anchor.Matched {
						matchedAnchors++
					}
					if anchor.Role == AnchorRequired {
						requiredAnchors++
						if anchor.Matched {
							requiredMatched++
						}
					}
				}
			}
			disabledText(fmt.Sprintf("正向锚点：%d/%d · 必需：%d/%d · 排除命中：%d/%d", matchedAnchors, positiveAnchors, requiredMatched, requiredAnchors, exclusionMatches, exclusionAnchors))
			if preview.Detection.TotalSlots > 0 {
				disabledText(fmt.Sprintf("生产槽位：%d/%d（填满阈值 %d）", preview.Detection.OccupiedSlots, preview.Detection.TotalSlots, preview.Detection.SlotThreshold))
			}
			if preview.Detection.TextError != "" {
				renderWarning("文字识别不可用", limitPillText(preview.Detection.TextError, 34))
			}
			if len(preview.Detection.Text) > 0 {
				sectionTitle("文字证据", "AutoGo PPOCR · 标题区域")
				for _, text := range preview.Detection.Text {
					colorText(colorCream, fmt.Sprintf("%s  %.1f%%", text.Text, text.Confidence*100))
				}
			}
			disabledText(limitPillText(fmt.Sprintf("帧 #%d · %s · %s", preview.FrameID, preview.Source, preview.UpdatedAt), 28))
		}

		imgui.Spacing()
		pushGoldButton()
		inspect := centeredButton("立即识别", "detection-inspect", imgui.Vec2{X: 150, Y: 42}, colorDarkInk)
		imgui.PopStyleColorV(4)
		if inspect {
			imgui.EndChild()
			return []Command{{Type: CommandInspect}}
		}
	}
	imgui.EndChild()
	return nil
}

func drawDetectionOverlay(drawList *imgui.DrawList, origin, size imgui.Vec2, detection Detection) {
	toScreen := func(x, y int) imgui.Vec2 {
		return imgui.Vec2{
			X: origin.X + float32(x)*size.X/1600,
			Y: origin.Y + float32(y)*size.Y/900,
		}
	}
	for _, anchor := range detection.Anchors {
		color := colorWarning
		switch anchor.Role {
		case AnchorExclusion:
			color = colorGreen
			if anchor.Matched {
				color = colorRed
			}
		case AnchorRequired:
			color = colorRed
			if anchor.Matched {
				color = colorGreen
			}
		default:
			if anchor.Matched {
				color = colorGreen
			}
		}
		radius := float32(3)
		if anchor.Role == AnchorRequired {
			radius = 4
		}
		drawList.AddCircleFilled(toScreen(anchor.X, anchor.Y), radius, imgui.ColorU32Vec4(color))
	}
	for _, slot := range detection.Slots {
		color := colorRed
		if slot.Occupied {
			color = colorGreen
		}
		center := toScreen(slot.X, slot.Y)
		width := size.X * 42 / 1600
		height := size.Y * 42 / 900
		drawList.AddRectV(
			imgui.Vec2{X: center.X - width/2, Y: center.Y - height/2},
			imgui.Vec2{X: center.X + width/2, Y: center.Y + height/2},
			imgui.ColorU32Vec4(color),
			3,
			imgui.DrawFlagsRoundCornersAll,
			1.5,
		)
	}
}

func renderFooter(frame *panelFrame) []Command {
	var commands []Command
	if err := frame.Draft.Validate(); err != nil {
		colorText(colorRed, "配置校验失败："+err.Error())
	} else if frame.Feedback != "" {
		colorText(colorGreen, frame.Feedback)
	} else {
		disabledText("修改保存在草稿中；启动前会再次校验并写入本机存储。")
	}

	buttonRowX := imgui.WindowWidth() - 610
	if buttonRowX > imgui.CursorPosX() {
		imgui.SetCursorPosX(buttonRowX)
	}
	if centeredButton("退出", "footer-exit", imgui.Vec2{X: 92, Y: 42}, colorCream) {
		commands = append(commands, Command{Type: CommandStop})
	}
	imgui.SameLine()
	if centeredButton("诊断截图", "footer-diagnostic", imgui.Vec2{X: 128, Y: 42}, colorCream) {
		commands = append(commands, Command{Type: CommandDiagnostic})
	}
	imgui.SameLine()
	if centeredButton("保存配置", "footer-save", imgui.Vec2{X: 128, Y: 42}, colorCream) {
		cfg := cloneDraft(frame.Draft)
		commands = append(commands, Command{Type: CommandSave, Settings: &cfg})
	}
	imgui.SameLine()
	valid := frame.Draft.Validate() == nil
	imgui.BeginDisabledV(!valid)
	pushGoldButton()
	if centeredButton("保存并启动", "footer-save-start", imgui.Vec2{X: 190, Y: 42}, colorDarkInk) {
		cfg := cloneDraft(frame.Draft)
		commands = append(commands, Command{Type: CommandStart, Settings: &cfg})
		minimizeToPill(frame)
	}
	imgui.PopStyleColorV(4)
	imgui.EndDisabled()
	return commands
}

func sectionTitle(title, description string) {
	imgui.TextUnformatted(title)
	imgui.SameLine()
	disabledText(" · " + description)
}

func renderWarning(title, message string) {
	imgui.PushStyleColorVec4(imgui.ColChildBg, imgui.Vec4{X: 0.24, Y: 0.17, Z: 0.065, W: 1})
	imgui.PushStyleColorVec4(imgui.ColBorder, imgui.Vec4{X: 0.60, Y: 0.42, Z: 0.15, W: 1})
	childFlags := imgui.ChildFlagsBorders | imgui.ChildFlagsAlwaysUseWindowPadding | imgui.ChildFlagsAutoResizeY
	if imgui.BeginChildStrV("warning-"+title, imgui.Vec2{X: 0, Y: 0}, childFlags, imgui.WindowFlagsNoScrollbar) {
		colorText(colorWarning, "!  "+title)
		disabledText(message)
	}
	imgui.EndChild()
	imgui.PopStyleColorV(2)
}

func renderSuccess(title, message string) {
	imgui.PushStyleColorVec4(imgui.ColChildBg, imgui.Vec4{X: 0.10, Y: 0.20, Z: 0.14, W: 1})
	imgui.PushStyleColorVec4(imgui.ColBorder, imgui.Vec4{X: 0.25, Y: 0.56, Z: 0.37, W: 1})
	childFlags := imgui.ChildFlagsBorders | imgui.ChildFlagsAlwaysUseWindowPadding | imgui.ChildFlagsAutoResizeY
	if imgui.BeginChildStrV("success-"+title, imgui.Vec2{X: 0, Y: 0}, childFlags, imgui.WindowFlagsNoScrollbar) {
		colorText(colorGreen, "●  "+title)
		disabledText(message)
	}
	imgui.EndChild()
	imgui.PopStyleColorV(2)
}

// centeredButton keeps button interaction and styling in ImGui while drawing
// its label independently. The bundled Android Chinese font is taller than the
// inner clipping area of several production-sized buttons, making ImGui's
// ButtonTextAlign ineffective for vertical centering.
func centeredButton(label, id string, size imgui.Vec2, textColor imgui.Vec4) bool {
	clicked := imgui.ButtonV("##"+id, size)
	drawCenteredButtonLine(label, imgui.ItemRectMin(), imgui.ItemRectMax(), textColor)
	return clicked
}

func centeredTwoLineButton(title, detail, id string, size imgui.Vec2, textColor imgui.Vec4) bool {
	clicked := imgui.ButtonV("##"+id, size)
	minimum := imgui.ItemRectMin()
	maximum := imgui.ItemRectMax()
	titleSize := imgui.CalcTextSize(title)
	detailSize := imgui.CalcTextSize(detail)
	totalHeight := titleSize.Y + detailSize.Y
	startY := minimum.Y + (maximum.Y-minimum.Y-totalHeight)/2 - 1.5
	drawList := imgui.WindowDrawList()
	color := imgui.ColorU32Vec4(textColor)
	drawList.AddTextVec2(imgui.Vec2{
		X: minimum.X + (maximum.X-minimum.X-titleSize.X)/2,
		Y: startY,
	}, color, title)
	drawList.AddTextVec2(imgui.Vec2{
		X: minimum.X + (maximum.X-minimum.X-detailSize.X)/2,
		Y: startY + titleSize.Y,
	}, color, detail)
	return clicked
}

func drawCenteredButtonLine(label string, minimum, maximum imgui.Vec2, textColor imgui.Vec4) {
	textSize := imgui.CalcTextSize(label)
	position := imgui.Vec2{
		X: minimum.X + (maximum.X-minimum.X-textSize.X)/2,
		Y: minimum.Y + (maximum.Y-minimum.Y-textSize.Y)/2 - 1.5,
	}
	imgui.WindowDrawList().AddTextVec2(position, imgui.ColorU32Vec4(textColor), label)
}

func pushGoldButton() {
	imgui.PushStyleColorVec4(imgui.ColButton, imgui.Vec4{X: 0.82, Y: 0.58, Z: 0.20, W: 1})
	imgui.PushStyleColorVec4(imgui.ColButtonHovered, imgui.Vec4{X: 0.94, Y: 0.70, Z: 0.29, W: 1})
	imgui.PushStyleColorVec4(imgui.ColButtonActive, imgui.Vec4{X: 0.72, Y: 0.47, Z: 0.13, W: 1})
	imgui.PushStyleColorVec4(imgui.ColText, colorDarkInk)
}

func alignTextRight(text string, padding float32) {
	imgui.SameLine()
	target := imgui.WindowWidth() - imgui.CalcTextSize(text).X - padding
	if target > imgui.CursorPosX() {
		imgui.SetCursorPosX(target)
	}
}

func colorText(color imgui.Vec4, text string) {
	imgui.PushStyleColorVec4(imgui.ColText, color)
	imgui.TextUnformatted(text)
	imgui.PopStyleColor()
}

func wrappedText(text string) {
	imgui.TextWrapped(strings.ReplaceAll(text, "%", "%%"))
}

func disabledText(text string) {
	imgui.PushStyleColorVec4(imgui.ColText, colorMuted)
	imgui.TextUnformatted(text)
	imgui.PopStyleColor()
}

func wrappedMutedText(text string) {
	imgui.PushStyleColorVec4(imgui.ColText, colorMuted)
	wrappedText(text)
	imgui.PopStyleColor()
}

func runModeIndex(mode RunMode) int32 {
	switch mode {
	case RunOnce:
		return 1
	case RunScheduled:
		return 2
	default:
		return 0
	}
}

func runModeAt(index int32) RunMode {
	switch index {
	case 1:
		return RunOnce
	case 2:
		return RunScheduled
	default:
		return RunManual
	}
}
