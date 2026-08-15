// Package square 对应 Lua 工程的 game/常规_布谷鸟广场/：广场特征库、页面、会话、路由与任务。
package square

import (
	"fmt"

	"app/internal/config"
	"app/internal/core"
	"app/internal/game/kingdom"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/userconfig"
)

const (
	taskTag    = "[布谷鸟广场任务]"
	minStaySec = 60
)

func squareCfg() config.SquareConfig {
	cfg, err := userconfig.Square()
	if err != nil {
		logger.Warn(taskTag, "读取广场配置失败: %v", err)
		return config.Static.User.Square
	}
	return cfg
}

// dailyCap 每日奖励领取上限（对应 Lua dailyCap：`cfg().dailyCap or 240`；
// Go 配置缺省值已由 userconfig 合入 240，此处直接返回配置值）。
func dailyCap() int {
	return squareCfg().DailyCap
}

func staySec() int {
	sec := squareCfg().CheckIntervalSec
	if sec < minStaySec {
		return minStaySec
	}
	return sec
}

func maxChunkSec() int {
	chunk := squareCfg().ChunkSec
	if chunk <= 0 {
		return 10
	}
	return chunk
}

func stayProgressText() string {
	required := staySec()
	rem := StayRemaining(required)
	if rem <= 0 {
		return "可开弹窗查看奖励"
	}
	elapsed := max(0, required-rem)
	return fmt.Sprintf("有效停留 %ds/%ds", elapsed, required)
}

func ensureSquarePage() bool {
	if IsCurrent() || IsLeaveDialog() {
		return true
	}
	if kingdom.IsKingdomHome() {
		return KingdomToSquare()
	}
	logger.Warn(taskTag, " 当前界面未知，无法进入广场")
	return false
}

func openLeaveDialog() bool {
	if IsLeaveDialog() {
		StartStay()
		return true
	}
	if !ensureSquarePage() {
		PauseStay()
		return false
	}
	StartStay()
	return OpenLeaveDialog()
}

func finishToday(reason string) bool {
	logger.Info(taskTag, " 今日广场任务结束: %s", reason)
	status.SetTask("布谷鸟广场", "今日已完成")
	PauseStay()
	if !LeaveDialogToKingdom(30000) {
		return false
	}
	MarkDoneToday()
	return true
}

func claimAndFinish() bool {
	logger.Info(taskTag, " 奖励已达标，点击一次领回")
	status.SetTask("布谷鸟广场", "一次领回…")
	if !IsLeaveDialog() && !openLeaveDialog() {
		return false
	}
	TapClaimAll(500)
	color.Sleep(1500, 500)
	TapUtilDialog()
	return finishToday("已领取奖励")
}

func waitAccumulationChunk() bool {
	if !ensureSquarePage() {
		PauseStay()
		return false
	}
	if IsLeaveDialog() {
		TapCloseDialog(1000)
		color.Sleep(800, 400)
		if !IsCurrent() {
			return false
		}
	}

	StartStay()
	remaining := StayRemaining(staySec())
	if remaining <= 0 {
		logger.Info(taskTag, " 有效停留已满 %d 秒，打开离开弹窗检查", staySec())
		status.SetTask("布谷鸟广场", "检查奖励…")
		return openLeaveDialog() && HandleLeaveDialog()
	}

	chunk := min(remaining, maxChunkSec())
	status.CountdownSleep(chunk, "布谷鸟广场", func(int) string { return stayProgressText() }, min(5, chunk))
	return true
}

// HandleLeaveDialog 处理离开弹窗：检查完成状态 / 奖励数量（对应 SquareTask.handleLeaveDialog）。
func HandleLeaveDialog() bool {
	if !IsLeaveDialog() {
		logger.Warn(taskTag, " handleLeaveDialog 调用时不在离开弹窗")
		return false
	}

	StartStay()
	color.Sleep(500, 250)

	if IsFinishOcr() {
		return finishToday("isFinishOcr=最大")
	}

	pending, total, sum, ok := ReadRewardSum()
	if !ok {
		color.Sleep(1000, 500)
		pending, total, sum, ok = ReadRewardSum()
	}
	if !ok {
		logger.Warn(taskTag, " 无法识别奖励数量")
		return false
	}

	status.SetTask("布谷鸟广场", fmt.Sprintf("%d+%d=%d / %d", pending, total, sum, dailyCap()))
	if sum >= dailyCap() {
		return claimAndFinish()
	}

	logger.Info(taskTag, " 未达领取条件 %d/%d，返回广场继续挂机", sum, dailyCap())
	TapCloseDialog(1000)
	color.Sleep(800, 400)
	MarkCheckedToday()
	ResetStayTimer()
	return waitAccumulationChunk()
}

// LeaveForOtherTask 其它任务插队：回王国主城，保留本轮有效停留进度（对应 SquareTask.leaveForOtherTask）。
func LeaveForOtherTask() bool {
	PauseStay()
	if kingdom.IsKingdomHome() {
		return true
	}
	if IsLeaveDialog() {
		return LeaveDialogToKingdom(30000)
	}
	if IsCurrent() {
		TapBack(1000)
		if WaitLeaveDialog(8000, 500) {
			return LeaveDialogToKingdom(30000)
		}
	}
	return LeaveDialogToKingdom(30000)
}

// Run 运行布谷鸟广场任务（对应 SquareTask.run）。
// 返回 nil 表示正常结束，core.ErrSkip 表示本轮未完成动作。
func Run(_ *core.Guard) error {
	if IsDoneToday() {
		logger.Info(taskTag, " 今日已完成，跳过")
		return nil
	}

	Ensure()
	status.SetTask("布谷鸟广场", "执行中…")

	if HasCheckedToday() {
		if !waitAccumulationChunk() {
			return core.ErrSkip
		}
		return nil
	}

	if !openLeaveDialog() {
		return core.ErrSkip
	}
	if !HandleLeaveDialog() {
		return core.ErrSkip
	}
	return nil
}
