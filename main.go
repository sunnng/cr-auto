// cr-auto 是《冲呀！饼干人：王国》的 AutoGo 视觉自动化入口。
// 当前里程碑：M2b 全量业务模块 —— 自包含 ImGui 控制面板（internal/ui）已迁移；
// 引擎/任务层（internal/core + internal/lib + internal/vision，按 ADR-0002 结构直译、
// ADR-0004 AutoGo 图色）已接入：面板命令驱动主循环（守卫→调度→空闲等待）。
// M2b 已迁移矿山模块与广场/交易所/竞技场/繁星岛/洗脆饼模块（见 internal/game/register.go）。
package main

import (
	"context"
	"image"
	"sync"
	"time"

	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/store"
	"app/internal/lib/touch"
	"app/internal/ui"

	"github.com/Dasongzi1366/AutoGo/apkctl"
	"github.com/Dasongzi1366/AutoGo/device"
	"github.com/Dasongzi1366/AutoGo/files"
	"github.com/Dasongzi1366/AutoGo/images"
	"github.com/Dasongzi1366/AutoGo/motion"
	"github.com/Dasongzi1366/AutoGo/utils"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	commands := make(chan ui.Command, 32)
	panel := ui.NewPanel()
	// 会话存储路径（面板打开前注入：initialSettings 需要回读已保存的任务开关）。
	store.SetDefault(store.New(storePath()))

	status.SetSink(func(update status.Update) {
		phase := "running"
		if update.Phase == status.PhaseIdle {
			phase = "idle"
		}
		_ = panel.PublishPhase(phase, "running", update.Text)
	})
	logger.SetSink(func(level logger.Level, tag, message string) {
		if level >= logger.LevelError {
			utils.LogE(tag, message)
		} else {
			utils.LogI(tag, message)
		}
	})
	touch.SetPerform(touch.Perform{
		Tap:       func(x, y int) { motion.Click(x, y, 1, 0) },
		TouchDown: func(id, x, y int) { motion.TouchDown(x, y, id, 0) },
		TouchMove: func(id, x, y, durationMs int) {
			utils.Sleep(durationMs)
			motion.TouchMove(x, y, id, 0)
		},
		TouchUp: func(id, x, y int) bool { motion.TouchUp(x, y, id, 0); return true },
		Back:    func() { motion.Back(0) },
		Sleep:   func(ms int) { utils.Sleep(ms) },
		Random:  func(min, max int) int { return utils.Random(min, max) },
	})
	stealth := &captureStealth{panel: panel}
	color.SetScreen(&deviceScreen{stealth: stealth})
	injectDeviceOCR()

	host := NewHost(panel)
	width, height, dpi, _ := device.GetDisplayInfo(displayID)
	display := profileFromDevice(width, height, dpi)
	host.SetDeviceProfileValid(display.Width == display.RequiredWidth && display.Height == display.RequiredHeight)
	host.SetFrameSource(&deviceFrameSource{stealth: stealth})
	host.SetDiagnosticDir(diagnosticDir())

	caps := host.CurrentCapabilities()
	initialStatus := ui.RuntimeStatus{
		Phase:     "idle",
		Scene:     "unknown",
		Outcome:   "configure",
		Message:   "确认配置后启动；M2b 全量业务模块已注册（矿山/广场/交易所/竞技场/繁星岛/洗脆饼）",
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	if err := panel.Open(ui.Snapshot{
		Settings:     initialSettings(),
		Catalog:      taskDescriptors(caps),
		Status:       initialStatus,
		Capabilities: caps,
		Display:      display,
	}, func(command ui.Command) {
		select {
		case commands <- command:
		default:
			if command.Type == ui.CommandStart {
				_ = panel.PublishCommandResult(command.RequestID, "idle", "start_error", "命令队列已满，启动未发出")
			}
			utils.LogE("panel", "命令队列已满，丢弃:", command.Type)
		}
	}); err != nil {
		utils.LogE("panel", "无法打开配置面板:", err)
		return
	}
	defer panel.Close()

	stop := make(chan struct{})
	unregister := registerLifecycle(ctx, stop, host)
	defer unregister()

	for {
		select {
		case command := <-commands:
			utils.LogE("panel", "收到命令:", command.Type)
			if command.Type == ui.CommandExit {
				host.stop()
				return
			}
			host.Handle(command)
		case <-stop:
			host.stop()
			return
		case <-ctx.Done():
			return
		}
	}
}

const displayID = 0

// captureStealth 引用计数截图隐身：嵌套的 Screen.Begin 与 Capture 共用一次握手。
type captureStealth struct {
	panel *ui.Panel
	mu    sync.Mutex
	depth int
}

func (s *captureStealth) enter() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.depth++
	if s.depth > 1 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.panel.HideForCapture(ctx); err != nil {
		s.depth--
		return err
	}
	return nil
}

func (s *captureStealth) leave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.depth == 0 {
		return
	}
	s.depth--
	if s.depth == 0 {
		s.panel.RestoreAfterCapture()
	}
}

type deviceFrameSource struct {
	stealth *captureStealth
}

func (d *deviceFrameSource) Capture() (*image.NRGBA, error) {
	if err := d.stealth.enter(); err != nil {
		return nil, err
	}
	defer d.stealth.leave()
	return images.CaptureScreen(0, 0, 0, 0, displayID), nil
}

type deviceScreen struct {
	stealth *captureStealth
}

func (d *deviceScreen) Begin() { _ = d.stealth.enter() }
func (d *deviceScreen) End()   { d.stealth.leave() }

func (d *deviceScreen) DetectsMultiColors(colors string, sim float32) bool {
	return images.DetectsMultiColors(colors, sim, displayID)
}

func (d *deviceScreen) CmpColor(x, y int, colorStr string, sim float32) bool {
	return images.CmpColor(x, y, colorStr, sim, displayID)
}

func (d *deviceScreen) FindMultiColors(x1, y1, x2, y2 int, colors string, sim float32, dir int) (int, int) {
	return images.FindMultiColors(x1, y1, x2, y2, colors, sim, dir, displayID)
}

func (d *deviceScreen) FindMultiColorsAll(x1, y1, x2, y2 int, colors string, sim float32, dir int) []image.Point {
	raw := images.FindMultiColorsAll(x1, y1, x2, y2, colors, sim, dir, displayID)
	out := make([]image.Point, 0, len(raw))
	for _, p := range raw {
		out = append(out, image.Point{X: p.X, Y: p.Y})
	}
	return out
}

// storePath 会话持久化路径（AutoGo 工作目录下的 data/store.json）。
func storePath() string {
	if path := files.Path("data/store.json"); path != "" {
		return path
	}
	return "data/store.json"
}

// diagnosticDir 诊断截图保存目录（AutoGo 工作目录下的 data/diagnostics）。
func diagnosticDir() string {
	if path := files.Path("data/diagnostics"); path != "" {
		return path
	}
	return defaultDiagnosticDir
}

func registerLifecycle(ctx context.Context, stop chan<- struct{}, host *Host) func() {
	apkctl.RegEvent(apkctl.EventPause, func() {
		utils.LogE("lifecycle", "平台暂停事件")
		host.pause()
	})
	apkctl.RegEvent(apkctl.EventResume, func() {
		utils.LogE("lifecycle", "平台恢复事件")
		host.resume()
	})
	apkctl.RegEvent(apkctl.EventStop, func() {
		select {
		case stop <- struct{}{}:
		case <-ctx.Done():
		}
	})
	return func() {
		apkctl.RegEvent(apkctl.EventPause, nil)
		apkctl.RegEvent(apkctl.EventResume, nil)
		apkctl.RegEvent(apkctl.EventStop, nil)
	}
}
