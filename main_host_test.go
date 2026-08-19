package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"app/internal/core"
	"app/internal/game"
	"app/internal/game/kingdom"
	libcolor "app/internal/lib/color"
	"app/internal/lib/ocr"
	"app/internal/lib/store"
	"app/internal/lib/touch"
	"app/internal/lib/userconfig"
	"app/internal/ui"
)

func openTestPanel(t *testing.T) *ui.Panel {
	t.Helper()
	panel := ui.NewPanel()
	if err := panel.Open(ui.Snapshot{Settings: ui.Default()}, func(ui.Command) {}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(panel.Close)
	return panel
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !cond() {
		t.Fatal("condition not reached in time")
	}
}

// setupHostTest 注入临时会话存储，保证 CommandSave 走 userconfig 落盘路径。
func setupHostTest(t *testing.T) {
	t.Helper()
	store.SetDefault(store.New(filepath.Join(t.TempDir(), "store.json")))
	libcolor.SetScreen(libcolor.NewScriptedScreen())
	t.Cleanup(func() {
		store.SetDefault(nil)
		libcolor.SetScreen(nil)
	})
}

func allReadyCapabilities() ui.Capabilities {
	return ui.Capabilities{
		OCRReady:                true,
		VisionReady:             true,
		ResourceGuardReady:      true,
		SensitivePageGuardReady: true,
		DeviceProfileValid:      true,
	}
}

func newReadyHost(t *testing.T, panel *ui.Panel) *Host {
	t.Helper()
	host := NewHost(panel)
	host.SetCapabilitiesProbe(allReadyCapabilities)
	return host
}

// staticFrameSource 固定帧来源（识别诊断测试用）。
type staticFrameSource struct {
	frame *image.NRGBA
}

func (s *staticFrameSource) Capture() (*image.NRGBA, error) { return s.frame, nil }

// paintPointSpecs 把特征串的色点原样画到帧上（识别诊断测试用）。
func paintPointSpecs(img *image.NRGBA, spec string) {
	for _, chunk := range strings.Split(spec, ",") {
		parts := strings.Split(chunk, "|")
		if len(parts) < 3 {
			continue
		}
		x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		hex := strings.TrimSpace(parts[2])
		if dash := strings.LastIndex(hex, "-"); dash >= 0 {
			hex = hex[:dash]
		}
		rgb, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			continue
		}
		img.SetNRGBA(x, y, color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 0xff})
	}
}

func kingdomHomeFrame() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 1600, 900))
	paintPointSpecs(img, kingdom.Home().Feature.Points)
	return img
}

func TestHostSaveAppliesTaskSwitches(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := ui.Default()
	draft.Tasks = map[string]ui.TaskSetting{
		"mine_survey":    {Enabled: false, MaxRuns: 1},
		"mine_mining":    {Enabled: true, MaxRuns: 1},
		"seaside_market": {Enabled: true, MaxRuns: 5},
	}
	host.Handle(ui.Command{Type: ui.CommandSave, Settings: &draft})

	status := panel.Status()
	if status.Outcome != "config_saved" {
		t.Fatalf("outcome=%q want config_saved", status.Outcome)
	}

	fresh := userconfig.Default()
	var mineCfg struct {
		SurveyEnabled bool
		MiningEnabled bool
	}
	if err := fresh.Get("mine", &mineCfg); err != nil {
		t.Fatal(err)
	}
	if mineCfg.SurveyEnabled || !mineCfg.MiningEnabled {
		t.Fatalf("mine switches not applied: %+v", mineCfg)
	}
	var seaside struct{ Enabled bool }
	if err := fresh.Get("seasideMarket", &seaside); err != nil {
		t.Fatal(err)
	}
	if !seaside.Enabled {
		t.Fatalf("seaside switch not applied: %+v", seaside)
	}
}

func TestHostSaveRejectsNilSettings(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	host.Handle(ui.Command{Type: ui.CommandSave})
	if status := panel.Status(); status.Outcome != "config_error" {
		t.Fatalf("nil settings must publish config_error, got %q", status.Outcome)
	}
}

func TestHostSaveRejectsInvalidDraftWithoutWriting(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := ui.Default()
	draft.Safety.MinConfidence = 0.5 // 低于 0.90 下限
	host.Handle(ui.Command{Type: ui.CommandSave, Settings: &draft})

	if status := panel.Status(); status.Outcome != "config_error" {
		t.Fatalf("invalid draft must publish config_error, got %q", status.Outcome)
	}
	// 校验失败不得落盘：mine 段仍为默认（勘查开）。
	fresh := userconfig.Default()
	var mineCfg struct{ SurveyEnabled bool }
	if err := fresh.Get("mine", &mineCfg); err != nil {
		t.Fatal(err)
	}
	if !mineCfg.SurveyEnabled {
		t.Fatal("invalid draft must not touch the store")
	}
}

func TestHostStartWithSettingsSavesThenRuns(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := ui.Default()
	draft.Tasks = map[string]ui.TaskSetting{"mine_survey": {Enabled: false, MaxRuns: 1}}
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool { return host.isRunning() })

	fresh := userconfig.Default()
	var mineCfg struct{ SurveyEnabled bool }
	if err := fresh.Get("mine", &mineCfg); err != nil {
		t.Fatal(err)
	}
	if mineCfg.SurveyEnabled {
		t.Fatal("start with settings must persist task switches before running")
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostDiagnosticSavesFrame(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	dir := filepath.Join(t.TempDir(), "diag")
	host.SetDiagnosticDir(dir)
	host.SetFrameSource(&staticFrameSource{frame: image.NewNRGBA(image.Rect(0, 0, 16, 16))})

	host.Handle(ui.Command{Type: ui.CommandDiagnostic})

	status := panel.Status()
	if status.Outcome != "diagnostic_saved" {
		t.Fatalf("outcome=%q want diagnostic_saved", status.Outcome)
	}
	if !strings.Contains(status.Message, dir) {
		t.Fatalf("message must contain the saved path: %q", status.Message)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".png") {
		t.Fatalf("expected one png in %s, got %v err=%v", dir, entries, err)
	}
}

func TestHostDiagnosticWithoutFrameSource(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	host.Handle(ui.Command{Type: ui.CommandDiagnostic})

	if status := panel.Status(); status.Outcome != "diagnostic_error" {
		t.Fatalf("outcome=%q want diagnostic_error", status.Outcome)
	}
}

func TestHostInspectPublishesDetectionPreview(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	host.SetFrameSource(&staticFrameSource{frame: kingdomHomeFrame()})
	libcolor.SetScreen(libcolor.HitFeatures(kingdom.Home().Feature))

	host.Handle(ui.Command{Type: ui.CommandInspect})

	preview := panel.DetectionPreview()
	if preview.Error != "" {
		t.Fatalf("inspect must not publish an error: %s", preview.Error)
	}
	if preview.Detection.Scene != ui.SceneKingdomHome {
		t.Fatalf("scene=%q want %q", preview.Detection.Scene, ui.SceneKingdomHome)
	}
	if preview.Detection.Confidence != 1 {
		t.Fatalf("confidence=%v want 1", preview.Detection.Confidence)
	}
	if len(preview.Detection.Candidates) == 0 || len(preview.Detection.Anchors) == 0 {
		t.Fatalf("preview must include candidates and anchors: %+v", preview.Detection)
	}
}

func TestHostInspectWithoutFrameSource(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	host.Handle(ui.Command{Type: ui.CommandInspect})

	if preview := panel.DetectionPreview(); preview.Error == "" {
		t.Fatal("inspect without frame source must publish an error")
	}
}

func TestTaskDescriptorsCatalog(t *testing.T) {
	setupHostTest(t)

	descriptors := taskDescriptors(allReadyCapabilities())
	if len(descriptors) != 9 {
		t.Fatalf("catalog must publish 9 tasks, got %d", len(descriptors))
	}
	seen := map[string]bool{}
	for _, d := range descriptors {
		if d.ID == "" || d.Name == "" || d.Description == "" {
			t.Fatalf("incomplete descriptor: %+v", d)
		}
		if seen[d.ID] {
			t.Fatalf("duplicate descriptor ID %q", d.ID)
		}
		seen[d.ID] = true
		if !d.Available {
			t.Fatalf("M2b task %q must be available when host capabilities are ready", d.ID)
		}
		if d.MaxRuns < 1 {
			t.Fatalf("descriptor %q MaxRuns=%d", d.ID, d.MaxRuns)
		}
	}
	// 名称顺序与 RegisterAll 一致。
	for i, meta := range game.Catalog() {
		if descriptors[i].Name != meta.Name {
			t.Fatalf("descriptor[%d].Name=%q want %q", i, descriptors[i].Name, meta.Name)
		}
	}
}

func TestTaskDescriptorsWaitForDeviceOCR(t *testing.T) {
	setupHostTest(t)
	caps := ui.Capabilities{VisionReady: true, DeviceProfileValid: true}
	descriptors := taskDescriptors(caps)
	if len(descriptors) == 0 {
		t.Fatal("catalog must not be empty")
	}
	for _, d := range descriptors {
		if d.Available {
			t.Fatalf("%q must stay closed until device OCR is accepted", d.ID)
		}
		if d.UnavailableReason != "等待设备 OCR 验收" {
			t.Fatalf("%q reason=%q", d.ID, d.UnavailableReason)
		}
	}
}

func TestHostAllowsStartWhenSafetyGuardsMissing(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := NewHost(panel)

	caps := host.CurrentCapabilities()
	if caps.ResourceGuardReady || caps.SensitivePageGuardReady {
		t.Fatal("empty safety features must not claim guards ready")
	}

	host.Handle(ui.Command{Type: ui.CommandStart, RequestID: 7})
	waitFor(t, func() bool { return host.isRunning() })

	status := panel.Status()
	if status.Outcome == "start_error" {
		t.Fatalf("start must not be blocked by uncaptured safety features, message=%q", status.Message)
	}
	if status.RequestID != 7 {
		t.Fatalf("request id=%d want 7", status.RequestID)
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestProfileFromDeviceMarksMismatch(t *testing.T) {
	p := profileFromDevice(1280, 720, 240)
	if p.Width != 1280 || p.RequiredWidth != 1600 {
		t.Fatalf("%+v", p)
	}
	host := NewHost(openTestPanel(t))
	host.SetDeviceProfileValid(p.Width == p.RequiredWidth && p.Height == p.RequiredHeight)
	caps := host.CurrentCapabilities()
	if caps.DeviceProfileValid {
		t.Fatal("1280x720 must not be a valid device profile")
	}
}

func TestProfileFromDeviceAcceptsContractSize(t *testing.T) {
	p := profileFromDevice(1600, 900, 240)
	if p.Width != 1600 || p.Height != 900 || p.DPI != 240 {
		t.Fatalf("%+v", p)
	}
	host := NewHost(openTestPanel(t))
	host.SetDeviceProfileValid(p.Width == p.RequiredWidth && p.Height == p.RequiredHeight)
	if !host.CurrentCapabilities().DeviceProfileValid {
		t.Fatal("1600x900 must be a valid device profile")
	}
}

func TestProfileFromDeviceRejectsRotatedPortrait(t *testing.T) {
	p := profileFromDevice(900, 1600, 240)
	if p.Width == p.RequiredWidth && p.Height == p.RequiredHeight {
		t.Fatal("rotated 900x1600 must not match the landscape contract")
	}
}

func TestLandscapeMismatchAndSwapAreInvalid(t *testing.T) {
	cases := [][2]int{{1280, 720}, {900, 1600}, {0, 0}}
	for _, c := range cases {
		p := profileFromDevice(c[0], c[1], 240)
		valid := p.Width == p.RequiredWidth && p.Height == p.RequiredHeight
		if valid {
			t.Fatalf("unexpected valid %+v", c)
		}
	}
	p := profileFromDevice(1600, 900, 240)
	if p.Width != 1600 || p.Height != 900 {
		t.Fatalf("%+v", p)
	}
}

func TestDetectCapabilitiesReportsOCRFromEngine(t *testing.T) {
	setupHostTest(t)
	ocr.SetEngine(nil)
	t.Cleanup(func() { ocr.SetEngine(nil) })
	caps := detectCapabilities()
	if caps.OCRReady {
		t.Fatal("OCR must be unread until an engine is injected")
	}
	ocr.SetEngine(&fakeHostOCR{})
	caps = detectCapabilities()
	if !caps.OCRReady {
		t.Fatal("injected OCR engine must set OCRReady")
	}
}

type fakeHostOCR struct{}

func (*fakeHostOCR) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
	return "", nil
}
func (*fakeHostOCR) FindTapPoint(string, image.Rectangle) (int, int, bool) { return 0, 0, false }

// TestGameSceneKeysHaveDisplayNames 识别诊断页的场景键（game）必须都有中文显示名
// （ui），防止 detect.go / ui/detection.go 两侧键值漂移（add scene 时三处同步）。
func TestGameSceneKeysHaveDisplayNames(t *testing.T) {
	setupHostTest(t)

	for _, key := range game.SceneKeys() {
		name := ui.SceneDisplayName(key)
		if name == key || name == "未知场景" {
			t.Fatalf("scene key %q has no display name, got %q", key, name)
		}
	}
	// 未注册的键回退为原键显示。
	if got := ui.SceneDisplayName("not-a-real-scene"); got != "not-a-real-scene" {
		t.Fatalf("unknown scene must fall back to the raw key, got %q", got)
	}
}

func TestInitialSettingsSeedsTaskSwitchesFromUserConfig(t *testing.T) {
	setupHostTest(t)

	if err := game.ApplyTaskSwitches(userconfig.Default(), map[string]bool{
		"mine_survey":    false,
		"seaside_market": true,
	}); err != nil {
		t.Fatal(err)
	}

	draft := initialSettings()
	if draft.Tasks["mine_survey"].Enabled {
		t.Fatal("mine_survey must seed disabled from userconfig")
	}
	if !draft.Tasks["seaside_market"].Enabled {
		t.Fatal("seaside_market must seed enabled from userconfig")
	}
	// 未保存过的任务回读默认值（config.Static.User：square.Enabled=true）。
	if !draft.Tasks["square"].Enabled {
		t.Fatal("square must seed default enabled")
	}
	if len(draft.Tasks) != len(game.Catalog()) {
		t.Fatalf("draft must cover all %d catalog tasks, got %d", len(game.Catalog()), len(draft.Tasks))
	}
	for _, setting := range draft.Tasks {
		if setting.MaxRuns < 1 {
			t.Fatalf("seeded task must carry MaxRuns, got %+v", setting)
		}
	}
}

func TestHostStartRunsAndStopHalts(t *testing.T) {
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	if !host.stop() {
		t.Fatal("stop must halt a running engine")
	}
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostStopDoesNotNeedToBeFollowedByProcessExit(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })
	host.Handle(ui.Command{Type: ui.CommandStop})
	waitFor(t, func() bool { return !host.isRunning() })
	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostStartIsIdempotent(t *testing.T) {
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	host.Handle(ui.Command{Type: ui.CommandStart})
	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	rts := 0
	host.mu.Lock()
	if host.rt != nil {
		rts = 1
	}
	host.mu.Unlock()
	if rts != 1 {
		t.Fatal("second start must not create another runtime")
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostPauseResume(t *testing.T) {
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	host.Handle(ui.Command{Type: ui.CommandPause})
	host.Handle(ui.Command{Type: ui.CommandResume})
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostStopWithoutEngineIsNoop(t *testing.T) {
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	if host.stop() {
		t.Fatal("stop without engine must report false")
	}
}

func TestHostEngineStartsWithRegisterInjection(t *testing.T) {
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	// 注册在 Runtime.Run 内完成，轮询等待注入结果。
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.rt != nil && host.rt.Scheduler.Count() == 9
	})
	host.mu.Lock()
	scheduler := host.rt.Scheduler
	guard := host.rt.Guard
	host.mu.Unlock()
	// M2b：守卫 1 个（网络联机状态不稳定）+ 业务任务 9 个
	// （矿山勘查/开采/战斗/解除洋菜冻 + 海滩交易所/王国竞技场/梦幻繁星岛/布谷鸟广场/洗脆饼词条）。
	if scheduler.Count() != 9 {
		t.Fatalf("M2b register must inject 9 tasks, got %d", scheduler.Count())
	}
	if guard.TrapCount() != 1 {
		t.Fatalf("M2b register must inject 1 guard trap, got %d", guard.TrapCount())
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

// allTasksDisabled 关闭全部任务的草稿（让引擎每轮快速空转）。
func allTasksDisabled() ui.Draft {
	draft := ui.Default()
	for _, meta := range game.Catalog() {
		draft.Tasks[meta.ID] = ui.TaskSetting{Enabled: false, Priority: 0, MaxRuns: meta.MaxRuns}
	}
	return draft
}

func TestHostRunOnceStopsAfterOneRound(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := allTasksDisabled()
	draft.Run.Mode = ui.RunOnce
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	// 单次运行可能在一轮内极快完成（<10ms），直接等终态：停止 + 原因消息。
	waitFor(t, func() bool {
		return !host.isRunning() && strings.Contains(panel.Status().Message, "单次运行")
	})
}

func TestHostActionBudgetStopsEngine(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := allTasksDisabled()
	draft.Safety.MaxActionsPerRun = 3
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool { return host.isRunning() })

	touch.TapR(10, 10, 0)
	touch.TapR(20, 20, 0)
	touch.TapR(30, 30, 0)

	waitFor(t, func() bool { return !host.isRunning() })
	if status := panel.Status(); !strings.Contains(status.Message, "动作预算") {
		t.Fatalf("budget stop must explain the reason: %q", status.Message)
	}
}

func TestHostActionCountPublishedToPanel(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := allTasksDisabled()
	draft.Safety.MaxActionsPerRun = 100
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool { return host.isRunning() })

	touch.TapR(10, 10, 0)
	waitFor(t, func() bool { return panel.Status().ActionCount == 1 })
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostUnknownSceneTimeoutStopsEngine(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	host.SetFrameSource(&staticFrameSource{frame: image.NewNRGBA(image.Rect(0, 0, 1600, 900))})
	host.observeInterval = 100 * time.Millisecond

	fakeNow := time.Date(2026, 8, 15, 10, 0, 0, 0, time.Local)
	host.nowFn = func() time.Time { return fakeNow }

	draft := allTasksDisabled()
	draft.Safety.UnknownTimeoutSec = 5
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool { return host.isRunning() })

	// 等首个观测 tick 记下 unknownSince，再推进时钟触发超时。
	time.Sleep(250 * time.Millisecond)
	fakeNow = fakeNow.Add(6 * time.Second)
	waitFor(t, func() bool { return !host.isRunning() })
	if status := panel.Status(); !strings.Contains(status.Message, "未知场景超时") {
		t.Fatalf("unknown-scene stop must explain the reason: %q", status.Message)
	}
}

func TestHostSceneObservedIntoStatus(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	host.SetFrameSource(&staticFrameSource{frame: kingdomHomeFrame()})
	libcolor.SetScreen(libcolor.HitFeatures(kingdom.Home().Feature))
	host.observeInterval = 100 * time.Millisecond

	draft := allTasksDisabled()
	draft.Safety.UnknownTimeoutSec = 60
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool { return host.isRunning() })

	waitFor(t, func() bool { return panel.Status().Scene == string(ui.SceneKingdomHome) })
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostScheduledWaitsOutsideWindowThenStarts(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	fakeNow := time.Date(2026, 8, 15, 10, 0, 0, 0, time.Local)
	host.nowFn = func() time.Time { return fakeNow }

	draft := allTasksDisabled()
	draft.Run.Mode = ui.RunScheduled
	draft.Run.StartMinute = 10*60 + 10 // 10:10
	draft.Run.EndMinute = 10*60 + 20   // 10:20
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})

	// 窗口未到：等待计划时段，引擎未启动。
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.running && host.rt == nil
	})
	waitFor(t, func() bool { return strings.Contains(panel.Status().Message, "计划时段") })

	// 时钟进入窗口：引擎启动。
	fakeNow = time.Date(2026, 8, 15, 10, 15, 0, 0, time.Local)
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.rt != nil
	})
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostScheduledStopsWhenWindowEnds(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	// 帧来源存在时窗口结束由观测器检测（100ms 级），与设备端一致。
	host.SetFrameSource(&staticFrameSource{frame: kingdomHomeFrame()})
	libcolor.SetScreen(libcolor.HitFeatures(kingdom.Home().Feature))
	host.observeInterval = 100 * time.Millisecond

	fakeNow := time.Date(2026, 8, 15, 10, 5, 0, 0, time.Local)
	host.nowFn = func() time.Time { return fakeNow }

	draft := allTasksDisabled()
	draft.Run.Mode = ui.RunScheduled
	draft.Run.StartMinute = 10 * 60 // 10:00
	draft.Run.EndMinute = 10*60 + 9 // 10:09
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.rt != nil
	})

	fakeNow = time.Date(2026, 8, 15, 10, 10, 0, 0, time.Local)
	waitFor(t, func() bool { return !host.isRunning() })
	if status := panel.Status(); !strings.Contains(status.Message, "计划时段") {
		t.Fatalf("window-end stop must explain the reason: %q", status.Message)
	}
}

func TestHostScheduledFullDayWindowRunsLikeManual(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := allTasksDisabled()
	draft.Run.Mode = ui.RunScheduled
	draft.Run.StartMinute = 0
	draft.Run.EndMinute = 1439
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool { return host.isRunning() })
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.rt != nil
	})
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostPauseDuringScheduledWaitKeepsWaiting(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)
	host.nowFn = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local) }
	draft := ui.Default()
	draft.Run.Mode = ui.RunScheduled
	draft.Run.StartMinute = 600
	draft.Run.EndMinute = 700
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool { return panel.Status().Outcome == "scheduled_wait" })
	host.Handle(ui.Command{Type: ui.CommandPause})
	if panel.Status().Outcome != "scheduled_wait" {
		t.Fatalf("outcome=%q", panel.Status().Outcome)
	}
	if host.isRunning() != true {
		t.Fatal("wait must continue")
	}
	if !strings.Contains(panel.Status().Message, "无法暂停") {
		t.Fatalf("pause during wait must explain itself: %q", panel.Status().Message)
	}
	host.stop()
}

func TestHostStartAppliesSessionTaskPolicies(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	draft := allTasksDisabled()
	draft.Tasks["square"] = ui.TaskSetting{Enabled: false, Priority: 100, MaxRuns: 1}
	host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.rt != nil && host.rt.Scheduler.Count() == 9
	})
	host.mu.Lock()
	tasks := host.rt.Scheduler.Tasks()
	host.mu.Unlock()
	var square *core.Task
	for i := range tasks {
		if tasks[i].Name == "布谷鸟广场" {
			square = &tasks[i]
		}
	}
	if square == nil {
		t.Fatal("square task not registered")
	}
	if square.Priority != 100 || square.MaxRuns != 1 {
		t.Fatalf("session policy not applied: %+v", square)
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostPillStartWithoutSettingsRunsManual(t *testing.T) {
	setupHostTest(t)
	panel := openTestPanel(t)
	host := newReadyHost(t, panel)

	// 悬浮胶囊启动：无草稿 → 手动模式、无预算/超时限制。
	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.rt != nil
	})
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}
