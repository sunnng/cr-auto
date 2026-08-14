// cr-auto 是《冲呀！饼干人：王国》的 AutoGo 视觉自动化入口。
// 当前里程碑：从 auto-cookie 迁移的自包含 ImGui 控制面板（internal/ui），
// 打开面板确认设置后由宿主消费命令；引擎与任务域在后续里程碑接入。
package main

import (
	"context"
	"time"

	"app/internal/ui"

	"github.com/Dasongzi1366/AutoGo/apkctl"
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
		Message:   "确认配置后启动；任务与引擎将在后续里程碑接入",
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

	stop := make(chan struct{})
	unregister := registerLifecycle(ctx, stop)
	defer unregister()

	for {
		select {
		case command := <-commands:
			utils.LogE("panel", "收到命令:", command.Type)
			switch command.Type {
			case ui.CommandStop:
				return
			}
		case <-stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func registerLifecycle(ctx context.Context, stop chan<- struct{}) func() {
	apkctl.RegEvent(apkctl.EventPause, func() { utils.LogE("lifecycle", "平台暂停事件") })
	apkctl.RegEvent(apkctl.EventResume, func() { utils.LogE("lifecycle", "平台恢复事件") })
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
