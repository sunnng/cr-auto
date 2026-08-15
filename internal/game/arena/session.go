// Package arena 对应 Lua 工程的 game/常规_王国竞技场/：竞技场特征库、页面、会话、路由与任务。
package arena

import (
	"fmt"
	"time"

	"app/internal/config"
	"app/internal/lib/store"
)

const sessionKey = "arena_session"

// ArenaData 竞技场会话数据（对应 Lua Session.get 返回表）。
type ArenaData struct {
	Medals            int   `json:"medals"`
	Tickets           int   `json:"tickets"`
	Trophies          int   `json:"trophies"`
	Wins              int   `json:"wins"`
	Losses            int   `json:"losses"`
	Draws             int   `json:"draws"`
	BuyCount          int   `json:"buyCount"`
	NextFreeRefreshAt int64 `json:"nextFreeRefreshAt"`
}

// Get 读取竞技场会话（对应 Session.get；无记录时返回空数据）。
func Get() ArenaData {
	var d ArenaData
	if !store.Default().Load(sessionKey, &d) {
		return ArenaData{}
	}
	return d
}

// Set 保存竞技场会话（对应 Session.set）。
func Set(d ArenaData) {
	_ = store.Default().Set(sessionKey, d)
}

// Update 合并部分字段并保存（对应 Session.update）。
func Update(patch func(*ArenaData)) {
	d := Get()
	patch(&d)
	Set(d)
}

// TotalBattles 累计战斗次数（对应 Session.totalBattles）。
func TotalBattles(d ArenaData) int {
	return d.Wins + d.Losses + d.Draws
}

// IsReachMaxBattles 是否已达到最大战斗次数（对应 Session.isReachMaxBattles）。
func IsReachMaxBattles(cfg config.ArenaConfig, d ArenaData) bool {
	return cfg.MaxBattles > 0 && TotalBattles(d) >= cfg.MaxBattles
}

// Describe 会话状态摘要（对应 Session.describe）。
func Describe(d ArenaData) string {
	total := TotalBattles(d)
	rate := 0.0
	if total > 0 {
		rate = float64(d.Wins) / float64(total) * 100
	}
	return fmt.Sprintf("战斗%d 胜%d 负%d 平%d 胜率%.1f%% 门票%d 买票%d 奖杯%d",
		total, d.Wins, d.Losses, d.Draws, rate, d.Tickets, d.BuyCount, d.Trophies)
}

// HudText 顶部 HUD 完整竞技场统计（对应 Session.hudText）。
func HudText(cfg config.ArenaConfig, d ArenaData) string {
	total := TotalBattles(d)
	cap := "∞"
	if cfg.MaxBattles > 0 {
		cap = fmt.Sprintf("%d", cfg.MaxBattles)
	}
	rate := 0.0
	if total > 0 {
		rate = float64(d.Wins) / float64(total) * 100
	}
	return fmt.Sprintf("总%d/%s 胜%d 负%d 平%d 胜率%.1f%% 门票%d 买票%d 奖杯%d",
		total, cap, d.Wins, d.Losses, d.Draws, rate, d.Tickets, d.BuyCount, d.Trophies)
}

// Clear 清空会话（对应 Session.clear）。
func Clear() {
	_ = store.Default().Set(sessionKey, map[string]any{})
}

// SetNextFreeRefreshAt 设置下次免费刷新时间戳（对应 Session.setNextFreeRefreshAt）。
func SetNextFreeRefreshAt(at int64) {
	Update(func(d *ArenaData) { d.NextFreeRefreshAt = at })
}

// GetTimeUntilRefresh 距离下次免费刷新剩余秒数（对应 Session.getTimeUntilRefresh）。
func GetTimeUntilRefresh() int {
	d := Get()
	if d.NextFreeRefreshAt <= 0 {
		return 0
	}
	remain := int(d.NextFreeRefreshAt - time.Now().Unix())
	if remain > 0 {
		return remain
	}
	return 0
}

// ClearNextFreeRefresh 清除免费刷新等待（对应 Session.clearNextFreeRefresh）。
func ClearNextFreeRefresh() {
	Update(func(d *ArenaData) { d.NextFreeRefreshAt = 0 })
}
