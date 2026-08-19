// Package ui owns the self-contained ImGui control panel state and HUD
// projection. It intentionally imports no project domain package: hosts push
// data in through Publish* and consume user actions as Commands.
package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sync"
	"time"
)

type TaskDescriptor struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	MigrationStatus   string `json:"migrationStatus,omitempty"`
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
	MaxRuns           int    `json:"maxRuns"`
}

// ConfigTab identifies a configuration page. The IDs are kept stable so a
// renderer can safely recover from a stale navigation value.
type ConfigTab int32

const (
	ConfigTabOverview ConfigTab = iota
	ConfigTabTasks
	ConfigTabSafety
	ConfigTabDetection
)

type ConfigTabState struct {
	ID          ConfigTab
	Label       string
	Title       string
	Description string
	Available   bool
}

// ConfigTabs lists the four implemented configuration pages in navigation
// order. Every page is always available; there is no capability gate in this
// package. The returned slice is package data and must not be mutated.
func ConfigTabs() []ConfigTabState {
	return configTabs
}

var configTabs = []ConfigTabState{
	{ID: ConfigTabOverview, Label: "概览", Title: "运行概览", Description: "确认运行方式与环境状态，再启动安全视觉引擎。", Available: true},
	{ID: ConfigTabTasks, Label: "任务", Title: "功能任务", Description: "只有完成视觉素材和结果验证的任务才能启用。", Available: true},
	{ID: ConfigTabSafety, Label: "安全", Title: "安全策略", Description: "所有阈值直接约束引擎，强制安全锁无法关闭。", Available: true},
	{ID: ConfigTabDetection, Label: "识别", Title: "识别诊断", Description: "查看原始截图、锚点命中和场景置信度，不执行任何动作。", Available: true},
}

// NormalizeConfigTab prevents a stale navigation value from selecting a page
// that does not exist.
func NormalizeConfigTab(tab ConfigTab) ConfigTab {
	for _, state := range configTabs {
		if state.ID == tab {
			return tab
		}
	}
	return ConfigTabOverview
}

type RuntimeStatus struct {
	Phase       string `json:"phase"`
	Scene       string `json:"scene"`
	Outcome     string `json:"outcome"`
	Message     string `json:"message"`
	FrameID     uint64 `json:"frameId"`
	ActionCount int    `json:"actionCount"`
	UpdatedAt   string `json:"updatedAt"`
	RequestID   uint64 `json:"requestId,omitempty"`
}

// Frame is a host-supplied screen capture for the diagnostic page. It is the
// UI projection of a device frame, not a domain object.
type Frame struct {
	ID         uint64
	CapturedAt time.Time
	Image      *image.NRGBA
}

type DetectionPreview struct {
	Revision  uint64
	FrameID   uint64
	UpdatedAt string
	Source    string
	Error     string
	Image     *image.NRGBA
	Detection Detection
}

type Snapshot struct {
	Settings     Draft            `json:"settings"`
	Catalog      []TaskDescriptor `json:"catalog"`
	Status       RuntimeStatus    `json:"status"`
	Capabilities Capabilities     `json:"capabilities"`
	Display      DisplayProfile   `json:"display"`
}

// DisplayProfile is the host-published screen contract. The control panel
// never probes the device; it only lays out from these facts.
type DisplayProfile struct {
	Width, Height                 int
	RequiredWidth, RequiredHeight int
	DPI                           int
	SafeMinX, SafeMinY            int
	SafeMaxX, SafeMaxY            int
}

func DefaultDisplayProfile() DisplayProfile {
	return DisplayProfile{
		Width: 1600, Height: 900,
		RequiredWidth: 1600, RequiredHeight: 900,
		DPI:      240,
		SafeMinX: 0, SafeMinY: 0, SafeMaxX: 1600, SafeMaxY: 900,
	}
}

func panelWindowGeometry(profile DisplayProfile) (x, y, width, height float32) {
	width = 1160
	if max := float32(profile.Width - 80); profile.Width > 80 && max < width {
		width = max
	}
	height = 780
	if max := float32(profile.Height - 120); profile.Height > 120 && max < height {
		height = max
	}
	x = (float32(profile.Width) - width) / 2
	y = float32(profile.SafeMinY + 55)
	return
}

type CommandType string

const (
	CommandSave       CommandType = "config.save"
	CommandStart      CommandType = "engine.start"
	CommandPause      CommandType = "engine.pause"
	CommandResume     CommandType = "engine.resume"
	CommandStop       CommandType = "engine.stop"
	CommandExit       CommandType = "engine.exit"
	CommandDiagnostic CommandType = "diagnostics.snapshot"
	CommandInspect    CommandType = "diagnostics.inspect"
)

type Command struct {
	Type      CommandType
	RequestID uint64
	Settings  *Draft
}

// Panel contains renderer-independent state. The AutoGo ImGui renderer lives
// behind build-specific functions so pure Go tests do not require Android cgo.
type Panel struct {
	mu        sync.Mutex
	captureMu sync.Mutex
	opened    bool
	handler   func(Command)
	draft     Draft
	catalog   []TaskDescriptor
	status    RuntimeStatus
	feedback  string
	compact   bool
	logs      []string

	capabilities   Capabilities
	display        DisplayProfile
	requestSeq     uint64
	pendingStartID uint64

	activeTab          int32
	pillExpanded       bool
	pillExpansion      float32
	pillAnimationAt    time.Time
	pillControlsAt     time.Time
	pillCollapseAt     time.Time
	pillPointer        pillPointerState
	preview            DetectionPreview
	previewRevision    uint64
	captureHidden      bool
	captureRevision    uint64
	captureReady       chan struct{}
	captureError       error
	captureAcked       bool
	captureReadyFrames int
	stopExpandGen      uint64
	appliedStopExpand  uint64
	exitArmedUntil     time.Time
	saved              Draft
}

type panelFrame struct {
	Draft           Draft
	Catalog         []TaskDescriptor
	Status          RuntimeStatus
	Feedback        string
	ActiveTab       int32
	Compact         bool
	Starting        bool
	Logs            []string
	Capabilities    Capabilities
	Display         DisplayProfile
	PillExpanded    bool
	PillExpansion   float32
	PillAnimationAt time.Time
	PillControlsAt  time.Time
	PillCollapseAt  time.Time
	PillPointer     pillPointerState
	ExitArmedUntil  time.Time
	Preview         DetectionPreview
	CaptureHidden   bool
	CaptureRevision uint64
	Dirty           bool
}

func NewPanel() *Panel {
	return &Panel{}
}

func (p *Panel) Open(snapshot Snapshot, handler func(Command)) error {
	if handler == nil {
		return errors.New("ui: panel handler is required")
	}

	p.mu.Lock()
	if p.opened {
		p.mu.Unlock()
		return errors.New("ui: panel is already open")
	}
	p.opened = true
	p.handler = handler
	p.draft = cloneDraft(snapshot.Settings)
	p.saved = cloneDraft(p.draft)
	p.catalog = append([]TaskDescriptor(nil), snapshot.Catalog...)
	p.status = snapshot.Status
	p.capabilities = snapshot.Capabilities
	p.display = snapshot.Display
	if p.display.Width == 0 {
		p.display = DefaultDisplayProfile()
	}
	p.requestSeq = 0
	p.pendingStartID = 0
	p.feedback = ""
	p.compact = false
	p.logs = nil
	p.pillExpanded = false
	p.pillExpansion = 0
	p.pillAnimationAt = time.Time{}
	p.pillControlsAt = time.Time{}
	p.pillCollapseAt = time.Time{}
	p.pillPointer = pillPointerState{}
	p.preview = DetectionPreview{}
	p.previewRevision = 0
	p.captureHidden = false
	p.captureRevision = 0
	p.captureReady = nil
	p.captureError = nil
	p.captureAcked = false
	p.captureReadyFrames = 0
	p.appendLogLocked(snapshot.Status)
	p.activeTab = 0
	p.mu.Unlock()

	if err := startPanelRenderer(p); err != nil {
		p.mu.Lock()
		p.opened = false
		p.handler = nil
		p.mu.Unlock()
		return fmt.Errorf("ui: initialize ImGui: %w", err)
	}
	return nil
}

func (p *Panel) Publish(status RuntimeStatus) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	p.publishStatusLocked(status)
	return nil
}

// PublishCapabilities updates the host ability snapshot without changing the
// runtime phase. Catalog availability is published separately so the host can
// keep task descriptors as the source of task metadata.
func (p *Panel) PublishCapabilities(caps Capabilities) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	p.capabilities = caps
	return nil
}

// PublishDisplay updates the screen contract without changing the runtime phase.
func (p *Panel) PublishDisplay(profile DisplayProfile) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	p.display = profile
	return nil
}

// PublishCatalog replaces the task directory projection.
func (p *Panel) PublishCatalog(catalog []TaskDescriptor) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	p.catalog = append([]TaskDescriptor(nil), catalog...)
	return nil
}

// PublishDetectionPreview updates the diagnostic page without changing the
// runtime phase. Manual inspection never performs an action.
func (p *Panel) PublishDetectionPreview(frame Frame, detection Detection) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	p.publishDetectionLocked(DetectionPreview{
		FrameID:   frame.ID,
		UpdatedAt: frame.CapturedAt.Format(time.RFC3339),
		Source:    "manual",
		Image:     frame.Image,
		Detection: cloneDetection(detection),
	})
	return nil
}

func (p *Panel) PublishDetectionPreviewError(message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	p.publishDetectionLocked(DetectionPreview{
		UpdatedAt: time.Now().Format(time.RFC3339),
		Source:    "manual",
		Error:     message,
	})
	return nil
}

// PublishPhase updates operator control state without discarding the latest
// observed scene, frame, or action count.
func (p *Panel) PublishPhase(phase, outcome, message string) error {
	return p.PublishCommandResult(0, phase, outcome, message)
}

// PublishCommandResult is PublishPhase tagged with the command request the
// host is answering. A matching start request collapses to the pill only after
// a successful running result; errors keep the control panel open.
func (p *Panel) PublishCommandResult(requestID uint64, phase, outcome, message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	p.status.Phase = phase
	p.status.Outcome = outcome
	p.status.Message = message
	p.status.RequestID = requestID
	p.status.UpdatedAt = time.Now().Format(time.RFC3339)
	p.appendLogLocked(p.status)
	if message != "" {
		p.feedback = message
	}
	if phase == "idle" && outcome == "stopped" {
		p.stopExpandGen++
	}
	if outcome == "config_saved" {
		p.saved = cloneDraft(p.draft)
	}
	return nil
}

// PublishObservation 更新运行中的场景与动作计数（识别观测/动作预算 HUD），
// 不追加日志、不改变 phase/outcome/message（避免每帧刷屏）。
func (p *Panel) PublishObservation(scene string, actionCount int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return nil
	}
	if scene != "" {
		p.status.Scene = scene
	}
	p.status.ActionCount = actionCount
	p.status.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

// Status returns the latest published runtime status (host diagnostics/tests).
func (p *Panel) Status() RuntimeStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

// DetectionPreview returns the latest published detection preview (host
// diagnostics/tests; the renderer consumes it through the frame instead).
func (p *Panel) DetectionPreview() DetectionPreview {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneDetectionPreview(p.preview)
}

func (p *Panel) SetCompact(compact bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return
	}
	p.compact = compact
	if !compact {
		p.pillExpanded = false
		p.pillExpansion = 0
		p.pillAnimationAt = time.Time{}
		p.pillControlsAt = time.Time{}
		p.pillCollapseAt = time.Time{}
		p.pillPointer = pillPointerState{}
	}
}

// HideForCapture asks the renderer to present fully transparent frames and
// waits until two frames have completed their ImGui lifecycle. The extra frame
// lets the Android compositor present the hidden overlay before the device
// screenshot is taken. The lock keeps a manual diagnostic capture from racing
// an engine capture.
func (p *Panel) HideForCapture(ctx context.Context) error {
	p.captureMu.Lock()

	p.mu.Lock()
	if !p.opened {
		p.mu.Unlock()
		p.captureMu.Unlock()
		return errors.New("ui: panel is not open")
	}
	p.captureHidden = true
	p.captureRevision++
	p.captureReady = make(chan struct{})
	p.captureError = nil
	p.captureAcked = false
	p.captureReadyFrames = 0
	ready := p.captureReady
	p.mu.Unlock()

	select {
	case <-ready:
		p.mu.Lock()
		err := p.captureError
		p.mu.Unlock()
		if err != nil {
			p.RestoreAfterCapture()
			return err
		}
		// ImGui has finished building the hidden frame at this point. Leave a
		// short scheduling window for the platform compositor to present it.
		time.Sleep(50 * time.Millisecond)
		return nil
	case <-ctx.Done():
		p.RestoreAfterCapture()
		return ctx.Err()
	}
}

// RestoreAfterCapture makes the next renderer frame visible again. It must be
// called exactly once after a successful HideForCapture call.
func (p *Panel) RestoreAfterCapture() {
	p.mu.Lock()
	p.captureHidden = false
	p.captureReady = nil
	p.captureError = nil
	p.captureAcked = false
	p.captureReadyFrames = 0
	p.mu.Unlock()
	p.captureMu.Unlock()
}

func (p *Panel) markCaptureReady(revision uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened || !p.captureHidden || revision != p.captureRevision || p.captureReady == nil || p.captureAcked {
		return
	}
	p.captureReadyFrames++
	if p.captureReadyFrames < capturePresentationFrames {
		return
	}
	p.captureAcked = true
	close(p.captureReady)
}

func (p *Panel) Close() {
	p.mu.Lock()
	if !p.opened {
		p.mu.Unlock()
		return
	}
	p.opened = false
	p.handler = nil
	if p.captureHidden && p.captureReady != nil && !p.captureAcked {
		p.captureError = errors.New("ui: panel closed during capture preparation")
		p.captureAcked = true
		close(p.captureReady)
	}
	p.captureHidden = false
	p.mu.Unlock()
	stopPanelRenderer()
}

func (p *Panel) readFrame() (panelFrame, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return panelFrame{}, false
	}
	p.applyStartConfirmationLocked()
	p.applyStoppedExpansionLocked()
	return panelFrame{
		Draft:           p.draft,
		Catalog:         p.catalog,
		Status:          p.status,
		Feedback:        p.feedback,
		ActiveTab:       p.activeTab,
		Compact:         p.compact,
		Starting:        p.pendingStartID != 0,
		Logs:            p.logs,
		Capabilities:    p.capabilities,
		Display:         p.display,
		PillExpanded:    p.pillExpanded,
		PillExpansion:   p.pillExpansion,
		PillAnimationAt: p.pillAnimationAt,
		PillControlsAt:  p.pillControlsAt,
		PillCollapseAt:  p.pillCollapseAt,
		PillPointer:     p.pillPointer,
		ExitArmedUntil:  p.exitArmedUntil,
		Preview:         p.preview,
		CaptureHidden:   p.captureHidden,
		CaptureRevision: p.captureRevision,
		Dirty:           !DraftsEqual(p.draft, p.saved),
	}, true
}

func (p *Panel) writeFrame(frame panelFrame) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.opened {
		return
	}
	p.draft = frame.Draft
	p.activeTab = frame.ActiveTab
	p.compact = frame.Compact
	p.pillExpanded = frame.PillExpanded
	p.pillExpansion = frame.PillExpansion
	p.pillAnimationAt = frame.PillAnimationAt
	p.pillControlsAt = frame.PillControlsAt
	p.pillCollapseAt = frame.PillCollapseAt
	p.pillPointer = frame.PillPointer
	p.exitArmedUntil = frame.ExitArmedUntil
}

func (p *Panel) publishStatusLocked(status RuntimeStatus) {
	p.status = status
	p.appendLogLocked(status)
	if status.Message != "" {
		p.feedback = status.Message
	}
}

func (p *Panel) publishDetectionLocked(preview DetectionPreview) {
	p.previewRevision++
	preview.Revision = p.previewRevision
	p.preview = preview
}

func cloneDetectionPreview(src DetectionPreview) DetectionPreview {
	src.Detection = cloneDetection(src.Detection)
	return src
}

func (p *Panel) applyStoppedExpansionLocked() {
	if p.stopExpandGen == p.appliedStopExpand {
		return
	}
	p.appliedStopExpand = p.stopExpandGen
	p.compact = false
	p.resetPillLocked()
}

func (p *Panel) applyStartConfirmationLocked() {
	if p.pendingStartID == 0 || p.status.RequestID != p.pendingStartID {
		return
	}
	switch p.status.Outcome {
	case "config_error", "start_error":
		p.pendingStartID = 0
		p.compact = false
		p.resetPillLocked()
	case "running", "scheduled_wait", "already_running":
		if p.status.Phase != "running" && p.status.Phase != "paused" {
			return
		}
		p.pendingStartID = 0
		p.compact = true
		p.resetPillLocked()
	}
}

func (p *Panel) resetPillLocked() {
	p.pillExpanded = false
	p.pillExpansion = 0
	p.pillAnimationAt = time.Time{}
	p.pillControlsAt = time.Time{}
	p.pillCollapseAt = time.Time{}
	p.pillPointer = pillPointerState{}
}

func (p *Panel) appendLogLocked(status RuntimeStatus) {
	line := fmt.Sprintf("%s · %s · 动作 %d", fallback(status.Scene, "unknown"), fallback(status.Outcome, "idle"), status.ActionCount)
	if len(p.logs) > 0 && p.logs[len(p.logs)-1] == line {
		return
	}
	p.logs = append(p.logs, line)
	if len(p.logs) > 8 {
		p.logs = append([]string(nil), p.logs[len(p.logs)-8:]...)
	}
}

func (p *Panel) emit(command Command) {
	p.mu.Lock()
	if !p.opened {
		p.mu.Unlock()
		return
	}
	handler := p.handler
	if command.RequestID == 0 {
		p.requestSeq++
		command.RequestID = p.requestSeq
	}
	if command.Type == CommandStart {
		p.pendingStartID = command.RequestID
	}
	if command.Settings != nil {
		copied := cloneDraft(*command.Settings)
		command.Settings = &copied
	}
	p.mu.Unlock()
	if handler != nil {
		handler(command)
	}
}

func phaseLabel(phase string) string {
	switch phase {
	case "running":
		return "运行中"
	case "paused":
		return "已暂停"
	case "error":
		return "异常"
	case "starting":
		return "启动中"
	default:
		return "等待中"
	}
}

func fallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

const capturePresentationFrames = 2
