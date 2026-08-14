// cr-auto 是《冲呀！饼干人：王国》的 AutoGo 视觉自动化入口。
// 当前里程碑：M2a 守卫 + 矿山模块 —— 自包含 ImGui 控制面板（internal/ui）已迁移；
// 引擎/任务层（internal/core + internal/lib + internal/vision，按 ADR-0002 结构直译、
// ADR-0003 帧比色）已接入：面板命令驱动主循环（守卫→调度→空闲等待）。
// M2a 已迁移矿山模块（见 internal/game/register.go）。
package main

import (
	"context"
	"image"
	"time"

	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/store"
	"app/internal/lib/touch"
	"app/internal/ui"

	"github.com/Dasongzi1366/AutoGo/apkctl"
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
	initialStatus := ui.RuntimeStatus{
		Phase:     "idle",
		Scene:     "unknown",
		Outcome:   "configure",
		Message:   "确认配置后启动；M1 引擎底座已接入，M2a 矿山模块（勘查/开采/战斗/解除洋菜冻）已注册",
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	if err := panel.Open(ui.Snapshot{
		Settings: ui.Default(),
		Status:   initialStatus,
	}, func(command ui.Command) {
		select {
		case commands <- command:
		default:
			utils.LogE("panel", "命令队列已满，丢弃:", command.Type)
		}
	}); err != nil {
		utils.LogE("panel", "无法打开配置面板:", err)
		return
	}
	defer panel.Close()

	// ========== 设备适配注入（M1）==========
	// 状态发布 → 控制面板。
	status.SetSink(func(update status.Update) {
		phase := "running"
		if update.Phase == status.PhaseIdle {
			phase = "idle"
		}
		_ = panel.PublishPhase(phase, "running", update.Text)
	})
	// 日志 → AutoGo 控制台/日志文件。
	logger.SetSink(func(level logger.Level, tag, message string) {
		if level >= logger.LevelError {
			utils.LogE(tag, message)
		} else {
			utils.LogI(tag, message)
		}
	})
	// 触控 → AutoGo motion。
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
	// 帧来源 → 截图隐身 + CaptureScreen（ADR-0003 唯一截图出口）。
	color.SetFrameSource(&deviceFrameSource{panel: panel})
	// 会话存储路径。
	store.SetDefault(store.New(storePath()))

	host := NewHost(panel)
	stop := make(chan struct{})
	unregister := registerLifecycle(ctx, stop, host)
	defer unregister()

	for {
		select {
		case command := <-commands:
			utils.LogE("panel", "收到命令:", command.Type)
			if command.Type == ui.CommandStop {
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

// deviceFrameSource 在“截图隐身”握手后截取一帧，避免 ImGui 遮罩污染游戏截图。
type deviceFrameSource struct {
	panel *ui.Panel
}

func (d *deviceFrameSource) Capture() (*image.NRGBA, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := d.panel.HideForCapture(ctx); err != nil {
		return nil, err
	}
	defer d.panel.RestoreAfterCapture()
	return images.CaptureScreen(0, 0, 0, 0, 0), nil
}

// storePath 会话持久化路径（AutoGo 工作目录下的 data/store.json）。
func storePath() string {
	if path := files.Path("data/store.json"); path != "" {
		return path
	}
	return "data/store.json"
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
