// Package battle 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山战斗/：战斗页面与会话。
package battle

import (
	"fmt"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/store"
)

const (
	sessionKey = "mine_battle_session"
	sessionTag = "[矿山战斗.会话]"
)

// battleRaw 持久化会话结构（对应 Lua Store 存的 lastBattleAt）。
type battleRaw struct {
	LastBattleAt int64 `json:"lastBattleAt"`
}

func loadRaw() (battleRaw, bool) {
	var raw battleRaw
	if !store.Default().Load(sessionKey, &raw) || raw.LastBattleAt <= 0 {
		return battleRaw{}, false
	}
	return raw, true
}

// GetTimeUntilNext 距离下次战斗还剩多少秒（对应 Session.getTimeUntilNext，缺省 21600）。
func GetTimeUntilNext(intervalSec int) int {
	if intervalSec <= 0 {
		intervalSec = 21600
	}
	raw, ok := loadRaw()
	if !ok {
		logger.Debug(sessionTag, "getTimeUntilNext: 无上次战斗记录")
		return 0
	}
	remain := int(raw.LastBattleAt + int64(intervalSec) - time.Now().Unix())
	if remain > 0 {
		logger.Debug(sessionTag, "getTimeUntilNext: 冷却中，剩余 %ds", remain)
		return remain
	}
	logger.Debug(sessionTag, "getTimeUntilNext: 冷却已到期")
	return 0
}

// RecordBattle 记录本次战斗开始时间（对应 Session.recordBattle）。
func RecordBattle() {
	now := time.Now()
	_ = store.Default().Set(sessionKey, map[string]any{"lastBattleAt": now.Unix()})
	logger.Info(sessionTag, "recordBattle: 已记录战斗时间 %s", now.Format("15:04:05"))
}

// Clear 清理会话（对应 Session.clear）。
func Clear() {
	_ = store.Default().Set(sessionKey, map[string]any{})
	logger.Info(sessionTag, "clear: 会话已清理")
}

// Describe 会话状态摘要（对应 Session.describe）。
func Describe(intervalSec int) string {
	if intervalSec <= 0 {
		intervalSec = 21600
	}
	raw, ok := loadRaw()
	if !ok {
		return "无战斗记录"
	}
	remain := int(raw.LastBattleAt + int64(intervalSec) - time.Now().Unix())
	if remain > 0 {
		return fmt.Sprintf("冷却中，剩余 %ds（约%.1f小时）", remain, float64(remain)/3600)
	}
	return "冷却已到期（记录仍在，可清除）"
}
