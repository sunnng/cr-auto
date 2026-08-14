// Package mining 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山开采/：开采页面与会话。
package mining

import (
	"fmt"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/store"
	"app/internal/lib/userconfig"
)

const (
	sessionKey     = "mine_mining_session"
	sessionTag     = "[矿山开采.会话]"
	defaultBusySec = 6 * 3600
)

// miningRaw 持久化会话结构（对应 Lua Store 存的 allBusyUntil）。
type miningRaw struct {
	AllBusyUntil int64 `json:"allBusyUntil"`
}

func loadRaw() (miningRaw, bool) {
	var raw miningRaw
	if !store.Default().Load(sessionKey, &raw) || raw.AllBusyUntil <= 0 {
		return miningRaw{}, false
	}
	return raw, true
}

// resolveBusySec 读取用户配置的开采间隔（对应 Lua resolveBusySec）。
func resolveBusySec() int {
	cfg, err := userconfig.Mine()
	if err == nil && cfg.MiningIntervalSec > 0 {
		return cfg.MiningIntervalSec
	}
	return defaultBusySec
}

// RestoreProgress 所有栏位 busy 剩余秒数（0 表示无等待或已到期；对应 Session.restoreProgress）。
func RestoreProgress() int {
	raw, ok := loadRaw()
	if !ok {
		logger.Debug(sessionTag, "restoreProgress: 无 busy 记录")
		return 0
	}
	remain := int(raw.AllBusyUntil - time.Now().Unix())
	if remain > 0 {
		logger.Debug(sessionTag, "restoreProgress: busy 剩余 %ds", remain)
		return remain
	}
	logger.Debug(sessionTag, "restoreProgress: busy 已到期")
	return 0
}

// EnterBusyWait 进入 busy 等待（对应 Session.enterBusyWait；waitSec<=0 时读配置）。
func EnterBusyWait(waitSec int) {
	if waitSec <= 0 {
		waitSec = resolveBusySec()
	}
	until := time.Now().Unix() + int64(waitSec)
	_ = store.Default().Set(sessionKey, map[string]any{
		"allBusyUntil": until,
		"recordedAt":   time.Now().Unix(),
	})
	logger.Info(sessionTag, "enterBusyWait: busy %ds（到期戳 %d）", waitSec, until)
}

// CheckReady 检查是否可运行（busy 等待是否已到期；对应 Session.checkReady）。
func CheckReady() (bool, int) {
	raw, ok := loadRaw()
	if !ok {
		return true, 0
	}
	remain := int(raw.AllBusyUntil - time.Now().Unix())
	if remain <= 0 {
		logger.Debug(sessionTag, "checkReady: busy 已到期，可运行")
		return true, 0
	}
	logger.Debug(sessionTag, "checkReady: busy 中，剩余 %ds，本轮跳过", remain)
	return false, remain
}

// Clear 清理会话（对应 Session.clear）。
func Clear() {
	_ = store.Default().Set(sessionKey, map[string]any{})
	logger.Info(sessionTag, "clear: 会话已清理")
}

// Describe 会话状态摘要（对应 Session.describe）。
func Describe() string {
	raw, ok := loadRaw()
	if !ok {
		return "无 busy 记录"
	}
	remain := int(raw.AllBusyUntil - time.Now().Unix())
	if remain > 0 {
		return fmt.Sprintf("矿卡开采中，剩余 %ds", remain)
	}
	return "busy 已到期（记录仍在，可清除）"
}
