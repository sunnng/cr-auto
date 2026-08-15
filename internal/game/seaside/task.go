// Package seaside 对应 Lua 工程的 game/常规_海滩交易所/：交易所坐标库、页面、会话、路由与任务。
package seaside

import (
	"strings"

	"app/internal/config"
	"app/internal/core"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/userconfig"
)

const taskTag = "[海滩交易所.任务]"

func seasideCfg() config.SeasideMarketConfig {
	cfg, err := userconfig.SeasideMarket()
	if err != nil {
		logger.Warn(taskTag, "读取配置失败: %v", err)
		return config.Static.User.SeasideMarket
	}
	return cfg
}

func resolveItems() []string {
	cfg := seasideCfg()
	if len(cfg.Items) > 0 {
		return cfg.Items
	}
	return StockKeys()
}

func scheduleFromPage() bool {
	restockSec, raw, ok := ReadRestockSeconds()
	if ok && restockSec > 0 {
		ScheduleAfterRestock(restockSec)
		return true
	}
	if ok && restockSec == 0 {
		logger.Info(taskTag, " 刷新按钮仍为免费刷新，不写等待")
		return false
	}
	logger.Warn(taskTag, " 未能读取补货时间 raw=%s", raw)
	return false
}

// Run 运行海滩交易所任务（对应 Task.run）。
// 返回 nil 表示正常结束，core.ErrSkip 表示本轮未完成动作。
func Run(_ *core.Guard) error {
	cfg := seasideCfg()
	if !cfg.Enabled {
		logger.Info(taskTag, " 任务未启用，跳过")
		return core.ErrSkip
	}
	status.SetTask("海滩交易所", "进入中…")
	if !Enter() {
		return core.ErrSkip
	}
	if !EnsureItemTab() {
		logger.Warn(taskTag, " 未能确认道具交易所页")
		Leave()
		return core.ErrSkip
	}

	forceFirstRun := ConsumeStartupBypass()
	status.SetTask("海滩交易所", "检查刷新…")
	if IsFreeRefresh() {
		logger.Info(taskTag, " 可免费刷新，先刷新")
		status.SetTask("海滩交易所", "免费刷新…")
		TapRefresh()
	} else {
		remain, raw, ok := ReadRestockSeconds()
		if ok && remain > 0 {
			if forceFirstRun {
				logger.Info(taskTag, " 首轮强制扫货，忽略页面补货倒计时: %s", raw)
			} else {
				logger.Info(taskTag, " 当前冷却中，以 OCR 为准推迟: %s", raw)
				ScheduleAfterRestock(remain)
				Leave()
				return nil
			}
		}
	}

	items := resolveItems()
	logger.Info(taskTag, " 开始扫货: %s", strings.Join(items, ","))
	status.SetTask("海滩交易所", "扫货中…")
	stats := PurchaseWishlist(items)
	logger.Info(taskTag, " 扫货结束 purchased=%d soldOut=%d shortage=%d failed=%d",
		stats.Purchased, stats.Skipped.SoldOut, stats.Skipped.Shortage, stats.Skipped.Failed)

	status.SetTask("海滩交易所", "读取补货…")
	scheduleFromPage()
	Leave()
	return nil
}
