// Package jelly 对应 Lua 工程的 game/常规_未知的地底矿山/模块_解除洋菜冻/：解除洋菜冻页面与会话。
package jelly

import (
	"fmt"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/store"
	"app/internal/lib/userconfig"
)

const (
	sessionKey     = "mine_jelly_session"
	sessionTag     = "[解除洋菜冻.会话]"
	defaultWaitSec = 3600
)

// jellyRaw 持久化会话结构（对应 Lua Store 存的 waitUntil）。
type jellyRaw struct {
	WaitUntil int64 `json:"waitUntil"`
}

func loadRaw() (jellyRaw, bool) {
	var raw jellyRaw
	if !store.Default().Load(sessionKey, &raw) || raw.WaitUntil <= 0 {
		return jellyRaw{}, false
	}
	return raw, true
}

// resolveWaitSec 解析等待秒数（对应 Lua resolveWaitSec：自定义 > 配置 > 缺省）。
func resolveWaitSec(customSec int) int {
	if customSec > 0 {
		return customSec
	}
	cfg, err := userconfig.Mine()
	if err == nil && cfg.JellyIntervalSec > 0 {
		return cfg.JellyIntervalSec
	}
	return defaultWaitSec
}

// EnterWait 记录等待截止时间（对应 Session.enterWait）。
func EnterWait(waitSec int) {
	waitSec = resolveWaitSec(waitSec)
	until := time.Now().Unix() + int64(waitSec)
	_ = store.Default().Set(sessionKey, map[string]any{
		"waitUntil":  until,
		"recordedAt": time.Now().Unix(),
	})
	logger.Info(sessionTag, "enterWait: 等待 %ds（到期戳 %d）", waitSec, until)
}

// CheckReady 检查是否可运行（对应 Session.checkReady）。
func CheckReady() (bool, int) {
	raw, ok := loadRaw()
	if !ok {
		return true, 0
	}
	remain := int(raw.WaitUntil - time.Now().Unix())
	if remain <= 0 {
		logger.Debug(sessionTag, "checkReady: 等待已到期，可运行")
		return true, 0
	}
	logger.Debug(sessionTag, "checkReady: 等待中，剩余 %ds", remain)
	return false, remain
}

// RestoreProgress 剩余等待秒数（对应 Session.restoreProgress）。
func RestoreProgress() int {
	_, remain := CheckReady()
	return remain
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
		return "无等待记录"
	}
	remain := int(raw.WaitUntil - time.Now().Unix())
	if remain > 0 {
		return fmt.Sprintf("冷却中，剩余 %ds", remain)
	}
	return "冷却已到期"
}
