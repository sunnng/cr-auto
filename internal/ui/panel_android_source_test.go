package ui

import (
	"os"
	"strings"
	"testing"
)

// AutoGo's PushFont wrapper dereferences the Go *Font before entering cimgui.
// Unlike native Dear ImGui, nil does not mean "use the default font" here.
func TestAndroidStopRendererDeletesPreviewTexture(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	fn := sourceFuncBody(string(source), "stopPanelRenderer()")
	if fn == "" {
		t.Fatal("stopPanelRenderer missing")
	}
	if !strings.Contains(fn, "detectionPreviewTexture.Delete()") {
		t.Fatal("stop must delete the preview texture")
	}
}

func TestAndroidSyncPreviewUsesShouldRebuildTexture(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "ShouldRebuildTexture(") {
		t.Fatal("syncDetectionPreviewTexture must consult ShouldRebuildTexture")
	}
}

func TestAndroidPanelNeverPushesNilFont(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "imgui.PushFont(nil") {
		t.Fatal("panel_imgui.go calls PushFont with nil; AutoGo dereferences it and panics")
	}
}

func TestAndroidPanelRoutesAllButtonsThroughCenteredHelpers(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if strings.Contains(content, "imgui.Button(") {
		t.Fatal("panel_imgui.go contains a direct imgui.Button call")
	}
	if got := strings.Count(content, "imgui.ButtonV("); got != 2 {
		t.Fatalf("expected ButtonV only in the two centered helpers, got %d calls", got)
	}
}

func TestAndroidPanelDisablesOuterWindowScrolling(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if !strings.Contains(content, "imgui.WindowFlagsNoScrollbar | imgui.WindowFlagsNoScrollWithMouse") {
		t.Fatal("main panel window must disable its redundant outer scrollbar")
	}
}

func TestAndroidPanelIsFixedAndCanBeMinimized(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if !strings.Contains(content, "imgui.CondAlways") {
		t.Fatal("main panel position must be anchored every frame")
	}
	if !strings.Contains(content, "imgui.WindowFlagsNoMove") {
		t.Fatal("main panel must reject move gestures")
	}
	if !strings.Contains(content, "imgui.WindowFlagsNoCollapse") {
		t.Fatal("main panel must hide the native collapse arrow in favor of the title-bar minimize control")
	}
	for _, required := range []string{
		"renderTitleBarMinimizeButton()",
		"PushClipRectFullScreen()",
		"AddLineV(",
		"IsMouseClickedBoolV(imgui.MouseButtonLeft, false)",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("title-bar minimize control is missing %s", required)
		}
	}
	if strings.Contains(content, `centeredButton("收起"`) || strings.Contains(content, `"header-compact"`) {
		t.Fatal("minimize control must not remain in the content header")
	}
}

func TestAndroidPanelDoesNotForbidStartForUncapturedSafetyGuards(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if strings.Contains(content, "在能力验收完成前禁止启动") {
		t.Fatal("uncaptured resource/sensitive-page guards must not tell the operator that start is forbidden")
	}
}

func TestAndroidPanelProvidesCompactDynamicPill(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	interaction, err := os.ReadFile("pill_interaction.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source) + string(interaction)
	for _, required := range []string{
		`"##auto-cookie-pill-raw"`,
		`pillCollapsedWidth`,
		`pillCollapsedHeight`,
		`pillExpandedWidth`,
		`pillExpandedHeight`,
		`advancePillAnimation(frame, now)`,
		`pillEase(frame.PillExpansion)`,
		`interactionLayout := pillInteractionLayout(pillControlsEnabled(frame, now), visualLayout)`,
		`drawCollapsedPillContent`,
		`drawExpandedPillContent`,
		`imgui.IsMouseClickedBoolV(imgui.MouseButtonLeft, false)`,
		`imgui.MousePos()`,
		`imgui.IsMouseReleased(imgui.MouseButtonLeft)`,
		`refreshPillCollapse`,
		`FitRunes`,
		`CommandPause`,
		`CommandResume`,
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("compact pill is missing %s", required)
		}
	}
	if strings.Contains(string(source), "if hovered {\n\t\t\tframe.PillExpanded = true") {
		t.Fatal("compact pill must expand on click, not pointer hover")
	}
	if strings.Contains(string(source), "primePillInputWindow") {
		t.Fatal("pill must be fully rendered from frame one instead of using an empty priming window")
	}
	if strings.Contains(string(source), "imgui.WindowFlagsNoNav | imgui.WindowFlagsNoBackground") {
		t.Fatal("pill must use a real rounded window background so Android can hit-test it")
	}
	for _, required := range []string{
		`const panelWindowID =`,
		`if frame.Compact {`,
		`commands = renderPill(&frame)`,
		`commands = renderConfigWindow(&frame)`,
		`imgui.IsMouseDown(imgui.MouseButtonLeft)`,
		`renderDetectionPreview`,
		`drawDetectionOverlay`,
		`imgui.CreateTextureNrgba`,
		`CommandInspect`,
	} {
		if !strings.Contains(string(source), required) {
			t.Fatalf("compact mode does not preserve the panel window lifecycle: %s", required)
		}
	}
	if strings.Contains(string(source), "if !frame.Compact {\n\t\tcommands = append(commands, renderConfigWindow(&frame)...)") {
		t.Fatal("control panel and pill must not both be interactive in the same frame")
	}
	footer := string(source)
	startIdx := strings.Index(footer, `centeredButton("保存并启动"`)
	if startIdx < 0 {
		t.Fatal("save-and-start button missing")
	}
	footerSlice := footer[startIdx:]
	endFooter := strings.Index(footerSlice, "func sectionTitle")
	if endFooter < 0 {
		t.Fatal("could not isolate footer renderer")
	}
	if strings.Contains(footerSlice[:endFooter], "minimizeToPill(frame)") {
		t.Fatal("save-and-start must not collapse the panel before the host confirms running")
	}
}

func TestAndroidPanelHidesBothOverlayWindowsDuringCapture(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	for _, required := range []string{
		"if frame.CaptureHidden",
		"renderCapturePlaceholder()",
		"pillWindowID",
		"for _, windowID := range []string{panelWindowID, pillWindowID}",
		"renderTransparentWindow(windowID string",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("capture path is missing %s", required)
		}
	}
}

func TestAndroidPanelUsesVectorIconsForPillControls(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	for _, required := range []string{
		`type pillIcon uint8`,
		`drawPillIcon`,
		`pillIconConfig`,
		`pillIconPlay`,
		`pillIconPause`,
		`pillIconStop`,
		`AddTriangleFilled`,
		`AddCircleV`,
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("pill controls are missing vector icon primitive %s", required)
		}
	}
	if strings.Contains(content, `"⚙"`) || strings.Contains(content, `"Ⅱ"`) {
		t.Fatal("pill control icons must not depend on unsupported font glyphs")
	}
}

func TestAndroidPanelOverlayUsesImageBounds(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if strings.Contains(content, "size.X/1600") || strings.Contains(content, "size.Y/900") {
		t.Fatal("detection overlay must not hardcode 1600×900")
	}
	if strings.Contains(content, "/1600") || strings.Contains(content, "/900") {
		t.Fatal("detection overlay must not divide by a fixed display size")
	}
}

func TestAndroidPanelLayoutsFromDisplayProfile(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if !strings.Contains(content, "frame.Display.Width") {
		t.Fatal("panel/pill layout must read DisplayProfile")
	}
	if strings.Contains(content, "windowX := (1600 - width) / 2") {
		t.Fatal("pill must not hardcode 1600 for centering")
	}
}

func TestAndroidPanelFooterEmitsCommandExit(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if !strings.Contains(content, `Command{Type: CommandExit}`) {
		t.Fatal("footer exit must emit CommandExit, not CommandStop")
	}
	if strings.Contains(content, `centeredButton("退出", "footer-exit"`) {
		t.Fatal("footer must label the control as 退出脚本")
	}
}

func TestAndroidScheduledWaitPrimaryIsDisabled(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	for _, required := range []string{
		"pillPrimaryAction(frame.Status.Phase, frame.Status.Outcome)",
		"pillIconWait",
		`pillExpandedState(frame.Status.Phase, frame.Status.Outcome)`,
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("scheduled-wait pill is missing %s", required)
		}
	}
}

func TestAndroidPillDoesNotRotateLogs(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "Unix()/3") {
		t.Fatal("compact pill must not rotate logs on a 3-second clock")
	}
}

func TestAndroidPanelShowsDirtyAndPersistHints(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	for _, required := range []string{
		"有未保存修改",
		`"已保存"`,
		"任务开关会写入本机存储。运行模式、安全阈值和任务优先级/单次上限仅本次运行生效。",
		"frame.Dirty",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("dirty/persist footer is missing %s", required)
		}
	}
}

func TestAndroidScheduleUsesClockSliders(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if strings.Contains(content, `"开始分钟"`) || strings.Contains(content, "0, 1439") {
		t.Fatal("schedule editors must not use raw minute sliders")
	}
	if !strings.Contains(content, "JoinClock") || !strings.Contains(content, `"%02d:%02d"`) {
		t.Fatal("schedule editors must project HH:mm from JoinClock")
	}
}

func TestAndroidPanelGatesStartOnHostCapabilities(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	for _, required := range []string{
		"renderCapabilityRow",
		"EvaluateStart",
		"frame.Starting",
		"正在启动",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("capability-aware panel is missing %s", required)
		}
	}
}

func TestAndroidThemeColorsArePackageLevel(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "func pushKingdomTheme() int32 {\n\tcolors := []themeColor{") {
		t.Fatal("theme color table must not be allocated inside pushKingdomTheme")
	}
}

func TestAndroidPillSkipsConfigWindowGeometry(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	body := sourceFuncBody(string(source), "renderPill(")
	if body == "" {
		t.Fatal("renderPill missing")
	}
	if strings.Contains(body, "pushKingdomGeometry") {
		t.Fatal("compact pill must not push control-panel geometry styles")
	}
}

func TestAndroidPillDoesNotCloneDraft(t *testing.T) {
	source, err := os.ReadFile("panel_imgui.go")
	if err != nil {
		t.Fatal(err)
	}
	body := sourceFuncBody(string(source), "renderPill(")
	if body == "" {
		t.Fatal("renderPill missing")
	}
	if strings.Contains(body, "cloneDraft") {
		t.Fatal("compact pill frames must not clone the settings draft")
	}
}

func sourceFuncBody(src, name string) string {
	start := strings.Index(src, "func "+name)
	if start < 0 {
		return ""
	}
	rest := src[start:]
	next := strings.Index(rest[1:], "\nfunc ")
	if next < 0 {
		return rest
	}
	return rest[:next+1]
}
