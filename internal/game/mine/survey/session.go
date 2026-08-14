// Package survey 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山勘查/：勘查页面与会话。
package survey

import (
	"fmt"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/store"
)

const (
	sessionKey = "mine_venture_session"
	sessionTag = "[矿山勘查.会话]"
)

// surveyRaw 持久化会话结构（对应 Lua Store 存的 farWaitUntil）。
type surveyRaw struct {
	FarWaitUntil int64 `json:"farWaitUntil"`
}

func loadRaw() (surveyRaw, bool) {
	var raw surveyRaw
	if !store.Default().Load(sessionKey, &raw) || raw.FarWaitUntil <= 0 {
		return surveyRaw{}, false
	}
	return raw, true
}

// RestoreProgress 远距等待剩余秒数（0 表示无等待或已到期；对应 Session.restoreProgress）。
func RestoreProgress() int {
	raw, ok := loadRaw()
	if !ok {
		logger.Debug(sessionTag, "restoreProgress: 无会话记录")
		return 0
	}
	remain := int(raw.FarWaitUntil - time.Now().Unix())
	if remain > 0 {
		logger.Debug(sessionTag, "restoreProgress: 远距等待剩余 %ds", remain)
		return remain
	}
	logger.Debug(sessionTag, "restoreProgress: 等待已到期或无截止时间")
	return 0
}

// EnterFarWait 进入远距等待（对应 Session.enterFarWait，缺省 600s）。
func EnterFarWait(waitSec int) {
	if waitSec <= 0 {
		waitSec = 600
	}
	until := time.Now().Unix() + int64(waitSec)
	_ = store.Default().Set(sessionKey, map[string]any{"farWaitUntil": until})
	logger.Info(sessionTag, "enterFarWait: 进入远距等待 %ds（到期戳 %d）", waitSec, until)
}

// CheckFarWait 检查是否可运行（远距等待是否已到期；对应 Session.checkFarWait）。
func CheckFarWait() (bool, int) {
	raw, ok := loadRaw()
	if !ok {
		return true, 0
	}
	remain := int(raw.FarWaitUntil - time.Now().Unix())
	if remain <= 0 {
		logger.Debug(sessionTag, "checkFarWait: 等待已到期，可运行")
		return true, 0
	}
	logger.Debug(sessionTag, "checkFarWait: 等待中，剩余 %ds，本轮跳过", remain)
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
		return "无远距等待记录"
	}
	remain := int(raw.FarWaitUntil - time.Now().Unix())
	if remain > 0 {
		return fmt.Sprintf("远距等待中，剩余 %ds", remain)
	}
	return "等待已到期（记录仍在，可清除）"
}
