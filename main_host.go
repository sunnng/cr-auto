package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"app/internal/core"
	"app/internal/game"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/userconfig"
	"app/internal/ui"
)

// defaultDiagnosticDir 诊断截图默认保存目录（相对 AutoGo 脚本工作目录）。
const defaultDiagnosticDir = "data/diagnostics"

// Host 面板命令与平台生命周期的宿主接线：把 ui.Command 与 apkctl 事件
// 翻译成对引擎（core.Runtime）的控制。设备适配（截图/触控/日志/状态）在
// main 启动时注入；Host 本身不依赖 AutoGo，可桌面测试。
type Host struct {
	panel       *ui.Panel
	frameSource color.FrameSource // 识别诊断的帧来源（截图隐身握手在适配器内完成）
	diagDir     string            // 诊断截图保存目录

	mu      sync.Mutex
	rt      *core.Runtime
	cancel  context.CancelFunc
	running bool
	frameID uint64
}

func NewHost(panel *ui.Panel) *Host {
	return &Host{panel: panel, diagDir: defaultDiagnosticDir}
}

// SetFrameSource 注入识别诊断帧来源（设备端为“截图隐身 + CaptureScreen”适配器）。
func (h *Host) SetFrameSource(src color.FrameSource) { h.frameSource = src }

// SetDiagnosticDir 覆盖诊断截图保存目录（默认 data/diagnostics）。
func (h *Host) SetDiagnosticDir(dir string) {
	if dir != "" {
		h.diagDir = dir
	}
}

// Handle 消费面板命令。
func (h *Host) Handle(command ui.Command) {
	switch command.Type {
	case ui.CommandStart:
		h.start(command.Settings)
	case ui.CommandPause:
		h.pause()
	case ui.CommandResume:
		h.resume()
	case ui.CommandStop:
		h.stop()
	case ui.CommandSave:
		h.saveSettings(command.Settings)
	case ui.CommandDiagnostic:
		h.diagnostic()
	case ui.CommandInspect:
		h.inspect()
	default:
		logger.Warn("[Host]", "未知命令: %s", command.Type)
	}
}

// taskDescriptors 面板任务目录：把 9 个已注册任务的元数据发布为 ui.TaskDescriptor。
func taskDescriptors() []ui.TaskDescriptor {
	metas := game.Catalog()
	descriptors := make([]ui.TaskDescriptor, 0, len(metas))
	for _, meta := range metas {
		descriptors = append(descriptors, ui.TaskDescriptor{
			ID:          meta.ID,
			Name:        meta.Name,
			Description: meta.Description,
			Available:   true,
			MaxRuns:     meta.MaxRuns,
		})
	}
	return descriptors
}

// initialSettings 面板初始草稿：任务开关从 userconfig 回填（历史保存值 + 默认值），
// 保证面板显示与引擎实际消费的配置一致。
func initialSettings() ui.Draft {
	draft := ui.Default()
	switches, err := game.LoadTaskSwitches(userconfig.Default())
	if err != nil {
		logger.Warn("[Host]", "读取任务开关失败: %v", err)
		return draft
	}
	for _, meta := range game.Catalog() {
		setting := draft.Tasks[meta.ID]
		setting.Enabled = switches[meta.ID]
		setting.MaxRuns = meta.MaxRuns
		draft.Tasks[meta.ID] = setting
	}
	return draft
}

// saveSettings 处理“保存配置”命令：面板草稿的任务开关写入 userconfig。
func (h *Host) saveSettings(settings *ui.Draft) {
	if settings == nil {
		h.panel.PublishPhase("idle", "config_error", "缺少配置内容")
		return
	}
	if !h.applySettings(*settings) {
		return
	}
	phase, message := "idle", "配置已保存"
	if h.isRunning() {
		// 引擎侧 UserConfig 缓存随 RegisterAll 加载，运行中保存需重启后生效。
		phase, message = "running", "配置已保存（引擎重启后生效）"
	}
	h.panel.PublishPhase(phase, "config_saved", message)
}

// applySettings 校验面板草稿并把任务开关合并写入 userconfig；
// 校验或落盘失败时发布错误并返回 false。
func (h *Host) applySettings(settings ui.Draft) bool {
	if err := settings.Validate(); err != nil {
		h.panel.PublishPhase("idle", "config_error", "配置校验失败："+err.Error())
		return false
	}
	switches := make(map[string]bool, len(settings.Tasks))
	for id, setting := range settings.Tasks {
		switches[id] = setting.Enabled
	}
	if err := game.ApplyTaskSwitches(userconfig.Default(), switches); err != nil {
		h.panel.PublishPhase("idle", "config_error", "保存配置失败："+err.Error())
		return false
	}
	return true
}

// diagnostic 处理“诊断截图”命令：截取一帧并保存 PNG。
func (h *Host) diagnostic() {
	frame, err := h.captureFrame()
	if err != nil {
		h.panel.PublishPhase("idle", "diagnostic_error", err.Error())
		return
	}
	path, err := h.saveFrame(frame)
	if err != nil {
		h.panel.PublishPhase("idle", "diagnostic_error", "保存诊断截图失败："+err.Error())
		return
	}
	h.panel.PublishPhase("idle", "diagnostic_saved", "诊断截图已保存："+path)
}

// inspect 处理“立即识别”命令：截取一帧并发布识别诊断预览（不执行任何动作）。
func (h *Host) inspect() {
	frame, err := h.captureFrame()
	if err != nil {
		h.panel.PublishDetectionPreviewError(err.Error())
		return
	}
	h.panel.PublishDetectionPreview(ui.Frame{
		ID:         h.nextFrameID(),
		CapturedAt: time.Now(),
		Image:      frame,
	}, toUIDetection(game.DetectScene(frame)))
}

func (h *Host) captureFrame() (*image.NRGBA, error) {
	if h.frameSource == nil {
		return nil, errors.New("帧来源未注入，无法截图")
	}
	frame, err := h.frameSource.Capture()
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, errors.New("截图为空")
	}
	return frame, nil
}

// saveFrame 把帧保存为诊断 PNG，返回保存路径。
func (h *Host) saveFrame(frame *image.NRGBA) (string, error) {
	dir := h.diagDir
	if dir == "" {
		dir = defaultDiagnosticDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("diag-%d-%s.png", h.nextFrameID(), time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err := png.Encode(file, frame); err != nil {
		return "", err
	}
	return path, nil
}

// nextFrameID 单调递增的帧编号（诊断截图/识别预览共用）。
func (h *Host) nextFrameID() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.frameID++
	return h.frameID
}

// toUIDetection 把 game 的场景扫描结果投影为面板识别诊断页的 Detection。
// 场景扫描是软性置信度模型（命中率排序），没有硬约束锚点/排除锚点语义，
// 因此候选与锚点只填充真实存在的字段，Required*/Exclusion* 保持零值。
func toUIDetection(d game.SceneDetection) ui.Detection {
	detection := ui.Detection{
		Scene:      ui.SceneID(d.Best),
		Confidence: d.Confidence,
	}
	for _, candidate := range d.Candidates {
		detection.Candidates = append(detection.Candidates, ui.SceneCandidate{
			Scene:          ui.SceneID(candidate.Key),
			Score:          candidate.Score,
			MatchedAnchors: candidate.Matched,
			TotalAnchors:   candidate.Total,
		})
	}
	for _, result := range d.Anchors {
		coverage := float32(0)
		if result.Matched {
			coverage = 1
		}
		detection.Anchors = append(detection.Anchors, ui.AnchorObservation{
			X:        result.Point.X,
			Y:        result.Point.Y,
			Matched:  result.Matched,
			Coverage: coverage,
		})
	}
	return detection
}

// isRunning 引擎是否在跑（平台生命周期事件与面板命令共用）。
func (h *Host) isRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

// start 启动引擎：先应用面板草稿（若有），再重建运行时并注入业务注册。
func (h *Host) start(settings *ui.Draft) {
	if settings != nil {
		if !h.applySettings(*settings) {
			return
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt := core.NewRuntime(core.RuntimeOptions{})
	rt.Register = func() { game.RegisterAll(rt.Scheduler, rt.Guard) }

	h.rt = rt
	h.cancel = cancel
	h.running = true

	go func() {
		_ = rt.Run(ctx)
		h.mu.Lock()
		h.running = false
		h.mu.Unlock()
		h.panel.PublishPhase("idle", "stopped", "引擎已停止")
	}()
	h.panel.PublishPhase("running", "running", "引擎已启动（M2b 全量业务模块）")
}

// pause 暂停引擎（轮间生效）。
func (h *Host) pause() {
	h.mu.Lock()
	rt := h.rt
	h.mu.Unlock()
	if rt == nil {
		return
	}
	rt.Pause()
	h.panel.PublishPhase("paused", "paused", "引擎已暂停")
}

// resume 恢复引擎。
func (h *Host) resume() {
	h.mu.Lock()
	rt := h.rt
	h.mu.Unlock()
	if rt == nil {
		return
	}
	rt.Resume()
	h.panel.PublishPhase("running", "running", "引擎已恢复")
}

// stop 停止引擎；返回是否真的停止（无引擎在跑时为 false）。
func (h *Host) stop() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running {
		return false
	}
	if h.cancel != nil {
		h.cancel()
	}
	return true
}
