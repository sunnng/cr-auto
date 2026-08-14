package ui

import (
	"os"
	"strings"
	"testing"
)

// AutoGo's PushFont wrapper dereferences the Go *Font before entering cimgui.
// Unlike native Dear ImGui, nil does not mean "use the default font" here.
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
	if strings.Contains(content, "imgui.WindowFlagsNoCollapse") {
		t.Fatal("main panel must expose ImGui's native minimize/restore control")
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
		`commands = renderPill(&frame)`,
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
