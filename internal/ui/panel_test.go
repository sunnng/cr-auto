package ui

import (
	"context"
	"image"
	"testing"
	"time"
)

func TestConfigTabsExposeTheFourImplementedPages(t *testing.T) {
	tabs := ConfigTabs()
	if len(tabs) != 4 {
		t.Fatalf("expected exactly four implemented tabs, got %d", len(tabs))
	}
	want := []ConfigTab{ConfigTabOverview, ConfigTabTasks, ConfigTabSafety, ConfigTabDetection}
	for i, tab := range tabs {
		if !tab.Available {
			t.Fatalf("implemented tab must be available: %+v", tab)
		}
		if tab.ID != want[i] {
			t.Fatalf("unexpected tab order: got=%v want=%v", tabs, want)
		}
	}
}

func TestConfigTabsReturnsStableBackingStore(t *testing.T) {
	a := ConfigTabs()
	b := ConfigTabs()
	if &a[0] != &b[0] {
		t.Fatal("ConfigTabs must not allocate a new backing array every call")
	}
}

func TestDraftValidateRejectsBadTaskPolicies(t *testing.T) {
	draft := Default()
	draft.Tasks["mine_survey"] = TaskSetting{Enabled: true, Priority: 101, MaxRuns: 1}
	if err := draft.Validate(); err == nil {
		t.Fatal("priority beyond 100 must be rejected")
	}
	draft = Default()
	draft.Tasks["mine_survey"] = TaskSetting{Enabled: true, Priority: 50, MaxRuns: 0}
	if err := draft.Validate(); err == nil {
		t.Fatal("MaxRuns below 1 must be rejected")
	}
	draft = Default()
	draft.Tasks["mine_survey"] = TaskSetting{Enabled: true, Priority: 50, MaxRuns: 101}
	if err := draft.Validate(); err == nil {
		t.Fatal("MaxRuns beyond 100 must be rejected")
	}
	draft = Default()
	draft.Tasks["mine_survey"] = TaskSetting{Enabled: true, Priority: 50, MaxRuns: 5}
	if err := draft.Validate(); err != nil {
		t.Fatalf("valid task policy must pass: %v", err)
	}
}

func TestPanelPublishObservationUpdatesSceneAndCountWithoutLogSpam(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	panel.Publish(RuntimeStatus{Phase: "running", Outcome: "running", Message: "引擎已启动"})
	if err := panel.PublishObservation("mine_home", 12); err != nil {
		t.Fatal(err)
	}
	status := panel.Status()
	if status.Scene != "mine_home" || status.ActionCount != 12 {
		t.Fatalf("observation not applied: %+v", status)
	}
	if status.Phase != "running" || status.Outcome != "running" {
		t.Fatalf("observation must preserve phase/outcome: %+v", status)
	}
	if err := panel.PublishObservation("kingdom_home", 13); err != nil {
		t.Fatal(err)
	}
	logs, _ := panel.readFrame()
	if logs.Status.ActionCount != 13 {
		t.Fatalf("second observation lost: %+v", logs.Status)
	}
}

func TestPanelOwnsDraftAndEmitsIndependentSettingsCopy(t *testing.T) {
	initial := Default()
	initial.Tasks["daily"] = TaskSetting{Enabled: true, Priority: 50, MaxRuns: 1}

	var received Command
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: initial}, func(command Command) { received = command }); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	frame, ok := panel.readFrame()
	if !ok {
		t.Fatal("opened panel did not expose a frame")
	}
	frame.Draft.Tasks["daily"] = TaskSetting{Enabled: false, Priority: 10, MaxRuns: 2}
	panel.writeFrame(frame)
	panel.emit(Command{Type: CommandSave, Settings: &frame.Draft})

	frame.Draft.Tasks["daily"] = TaskSetting{Priority: 99, MaxRuns: 3}
	if received.Type != CommandSave || received.Settings == nil {
		t.Fatalf("save command not delivered: %+v", received)
	}
	if got := received.Settings.Tasks["daily"].Priority; got != 10 {
		t.Fatalf("emitted settings were aliased to renderer draft: priority=%d", got)
	}
	if got := initial.Tasks["daily"].Priority; got != 50 {
		t.Fatalf("panel mutated caller settings: priority=%d", got)
	}
}

func TestReadFrameSharesDraftMapWithRenderer(t *testing.T) {
	panel := NewPanel()
	_ = panel.Open(Snapshot{Settings: Default()}, func(Command) {})
	defer panel.Close()
	frame, _ := panel.readFrame()
	frame.Draft.Run.Mode = RunOnce
	panel.writeFrame(frame)
	next, _ := panel.readFrame()
	if next.Draft.Run.Mode != RunOnce {
		t.Fatal("renderer draft edits must round-trip")
	}
}

func TestReadFrameDoesNotCloneCatalogWhenHostQuiet(t *testing.T) {
	panel := NewPanel()
	catalog := []TaskDescriptor{{ID: "a", Name: "A", Available: true, MaxRuns: 1}}
	_ = panel.Open(Snapshot{Settings: Default(), Catalog: catalog}, func(Command) {})
	defer panel.Close()
	a, _ := panel.readFrame()
	b, _ := panel.readFrame()
	if &a.Catalog[0] != &b.Catalog[0] {
		t.Fatal("quiet host must not recopy catalog each frame")
	}
}

func TestPanelPublishUpdatesVisibleStatus(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	want := RuntimeStatus{Phase: "error", Outcome: "config_rejected", Message: "bad config"}
	if err := panel.Publish(want); err != nil {
		t.Fatal(err)
	}
	frame, ok := panel.readFrame()
	if !ok || frame.Status != want || frame.Feedback != want.Message {
		t.Fatalf("status not projected into panel: %+v", frame)
	}
}

func TestPanelKeepsSelectedNavigationTabBetweenFrames(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	frame, ok := panel.readFrame()
	if !ok {
		t.Fatal("opened panel did not expose a frame")
	}
	frame.ActiveTab = 3
	panel.writeFrame(frame)

	next, ok := panel.readFrame()
	if !ok || next.ActiveTab != 3 {
		t.Fatalf("navigation state was not retained: %+v", next)
	}
}

func TestPanelCompactsWithoutClosingAndKeepsRecentRuntimeLogs(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	panel.SetCompact(true)
	for i := 0; i < 10; i++ {
		status := RuntimeStatus{Phase: "running", Scene: "kingdom_home", Outcome: "observed", ActionCount: i}
		if err := panel.Publish(status); err != nil {
			t.Fatal(err)
		}
	}
	frame, ok := panel.readFrame()
	if !ok || !frame.Compact {
		t.Fatalf("panel did not enter compact mode: %+v", frame)
	}
	if len(frame.Logs) != 8 {
		t.Fatalf("expected bounded eight-line runtime history, got %d", len(frame.Logs))
	}
	if frame.Logs[len(frame.Logs)-1] != "kingdom_home · observed · 动作 9" {
		t.Fatalf("unexpected latest log: %q", frame.Logs[len(frame.Logs)-1])
	}

	frame.Compact = false
	panel.writeFrame(frame)
	reopened, ok := panel.readFrame()
	if !ok || reopened.Compact {
		t.Fatalf("compact panel could not return to configuration: %+v", reopened)
	}
}

func TestPanelCaptureHandshakeWaitsForHiddenFrame(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- panel.HideForCapture(ctx) }()

	var revision uint64
	deadline := time.After(time.Second)
	for {
		frame, ok := panel.readFrame()
		if ok && frame.CaptureHidden {
			revision = frame.CaptureRevision
			break
		}
		select {
		case <-deadline:
			t.Fatal("capture request did not reach renderer state")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	select {
	case err := <-result:
		t.Fatalf("capture returned before hidden frame was acknowledged: %v", err)
	default:
	}
	panel.markCaptureReady(revision)
	panel.markCaptureReady(revision)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	frame, ok := panel.readFrame()
	if !ok || !frame.CaptureHidden {
		t.Fatalf("panel became visible before restore: %+v", frame)
	}
	panel.RestoreAfterCapture()
	frame, ok = panel.readFrame()
	if !ok || frame.CaptureHidden {
		t.Fatalf("panel did not restore after capture: %+v", frame)
	}
}

func TestPanelCaptureHandshakeWaitsForTwoHiddenFrames(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- panel.HideForCapture(ctx) }()

	var revision uint64
	deadline := time.After(time.Second)
	for {
		frame, ok := panel.readFrame()
		if ok && frame.CaptureHidden {
			revision = frame.CaptureRevision
			break
		}
		select {
		case <-deadline:
			t.Fatal("capture request did not reach renderer state")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	panel.markCaptureReady(revision)
	select {
	case err := <-result:
		t.Fatalf("capture returned after only one hidden frame: %v", err)
	default:
	}
	select {
	case err := <-result:
		t.Fatalf("capture returned after only one hidden frame: %v", err)
	case <-time.After(75 * time.Millisecond):
	}
	panel.markCaptureReady(revision)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	panel.RestoreAfterCapture()
}

func TestPanelPublishesRuntimeDetectionPreviewWithoutAliasingEvidence(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	img := image.NewNRGBA(image.Rect(0, 0, 1600, 900))
	preview := Detection{
		Scene:      "kingdom_home",
		Confidence: 0.97,
		Candidates: []SceneCandidate{{Scene: "kingdom_home", Score: 0.97}},
	}
	if err := panel.PublishDetectionPreview(Frame{ID: 42, Image: img}, preview); err != nil {
		t.Fatal(err)
	}
	preview.Candidates[0].Score = 0.1
	frame, ok := panel.readFrame()
	if !ok || frame.Preview.FrameID != 42 || frame.Preview.Image != img {
		t.Fatalf("preview was not published: %+v", frame.Preview)
	}
	if got := frame.Preview.Detection.Candidates[0].Score; got != 0.97 {
		t.Fatalf("preview evidence was aliased: score=%v", got)
	}
}

func TestPanelWriteFrameDoesNotClobberConcurrentPublish(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	if err := panel.PublishPhase("idle", "configure", "初始反馈"); err != nil {
		t.Fatal(err)
	}
	frame, ok := panel.readFrame()
	if !ok {
		t.Fatal("opened panel did not expose a frame")
	}

	if err := panel.PublishPhase("running", "running", "最新反馈"); err != nil {
		t.Fatal(err)
	}
	frame.ActiveTab = 2
	panel.writeFrame(frame)

	next, ok := panel.readFrame()
	if !ok {
		t.Fatal("panel closed unexpectedly")
	}
	if next.Feedback != "最新反馈" {
		t.Fatalf("writeFrame overwrote host feedback: %q", next.Feedback)
	}
	if next.Status.Phase != "running" || next.Status.Outcome != "running" {
		t.Fatalf("writeFrame overwrote host status: %+v", next.Status)
	}
	if len(next.Logs) == 0 || next.Logs[len(next.Logs)-1] != "unknown · running · 动作 0" {
		t.Fatalf("writeFrame overwrote host logs: %v", next.Logs)
	}
	if next.ActiveTab != 2 {
		t.Fatalf("renderer tab write was dropped: %d", next.ActiveTab)
	}
}

func TestPanelStartWaitsForMatchingRunningBeforeCompact(t *testing.T) {
	var started Command
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(command Command) { started = command }); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	draft := Default()
	panel.emit(Command{Type: CommandStart, Settings: &draft})
	if started.Type != CommandStart || started.RequestID == 0 {
		t.Fatalf("start command must carry a request id: %+v", started)
	}

	frame, ok := panel.readFrame()
	if !ok || frame.Compact || !frame.Starting {
		t.Fatalf("start must stay on the panel until the host confirms: %+v", frame)
	}

	if err := panel.PublishCommandResult(started.RequestID+1, "idle", "config_error", "旧请求失败"); err != nil {
		t.Fatal(err)
	}
	frame, ok = panel.readFrame()
	if !ok || frame.Compact || !frame.Starting {
		t.Fatalf("stale start error must not finish the in-flight request: %+v", frame)
	}

	if err := panel.PublishCommandResult(started.RequestID, "running", "running", "引擎已启动"); err != nil {
		t.Fatal(err)
	}
	frame, ok = panel.readFrame()
	if !ok || !frame.Compact || frame.Starting {
		t.Fatalf("matching running result must collapse to the pill: %+v", frame)
	}
}

func TestPanelStartErrorKeepsPanelExpanded(t *testing.T) {
	var started Command
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default()}, func(command Command) { started = command }); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	panel.SetCompact(true)
	draft := Default()
	panel.emit(Command{Type: CommandStart, Settings: &draft})

	if err := panel.PublishCommandResult(started.RequestID, "idle", "config_error", "保存配置失败"); err != nil {
		t.Fatal(err)
	}
	frame, ok := panel.readFrame()
	if !ok || frame.Compact || frame.Starting {
		t.Fatalf("start error must reopen the panel: %+v", frame)
	}
	if frame.Feedback != "保存配置失败" {
		t.Fatalf("start error feedback=%q", frame.Feedback)
	}
}

func TestPanelDirtyTracksUnsavedDraftEdits(t *testing.T) {
	panel := NewPanel()
	_ = panel.Open(Snapshot{Settings: Default()}, func(Command) {})
	defer panel.Close()
	frame, _ := panel.readFrame()
	if frame.Dirty {
		t.Fatal("fresh panel must be clean")
	}
	frame.Draft.Run.Mode = RunOnce
	panel.writeFrame(frame)
	next, _ := panel.readFrame()
	if !next.Dirty {
		t.Fatal("mode change must mark dirty")
	}
	_ = panel.PublishPhase("idle", "config_saved", "配置已保存")
	next, _ = panel.readFrame()
	if next.Dirty {
		t.Fatal("config_saved must clear dirty")
	}
}

func TestPanelPublishCapabilitiesProjectsIntoFrame(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default(), Capabilities: Capabilities{VisionReady: true}}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()

	frame, ok := panel.readFrame()
	if !ok || !frame.Capabilities.VisionReady || frame.Capabilities.OCRReady {
		t.Fatalf("open snapshot capabilities not projected: %+v", frame.Capabilities)
	}
	want := readyCapabilities()
	if err := panel.PublishCapabilities(want); err != nil {
		t.Fatal(err)
	}
	frame, ok = panel.readFrame()
	if !ok || frame.Capabilities != want {
		t.Fatalf("published capabilities not projected: %+v", frame.Capabilities)
	}
}

func TestPanelExpandsWhenHostStops(t *testing.T) {
	panel := NewPanel()
	_ = panel.Open(Snapshot{Settings: Default()}, func(Command) {})
	defer panel.Close()
	panel.SetCompact(true)
	_ = panel.PublishPhase("idle", "stopped", "引擎已停止")
	frame, _ := panel.readFrame()
	if frame.Compact {
		t.Fatal("stopped engine must restore the control panel")
	}
}
