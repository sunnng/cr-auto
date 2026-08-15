// Package seaside 对应 Lua 工程的 game/常规_海滩交易所/：交易所坐标库、页面、会话、路由与任务。
package seaside

import (
	"fmt"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/store"
)

const (
	sessionKey        = "seaside_market_session"
	sessionTag        = "[海滩交易所.会话]"
	defaultRestockSec = 6 * 3600
)

// startupBypassPending/startupBypassActive 本次脚本启动首轮强制执行（对应 Lua 模块级变量）。
var (
	startupBypassPending = true
	startupBypassActive  = false
)

// seasideRaw 持久化会话结构（对应 Lua Store 存的 nextRunAt 等）。
type seasideRaw struct {
	NextRunAt  int64 `json:"nextRunAt"`
	RestockSec int   `json:"restockSec"`
	BufferSec  int   `json:"bufferSec"`
	RecordedAt int64 `json:"recordedAt"`
}

func bufferSec() int {
	sec := seasideCfg().RestockBufferSec
	if sec >= 0 {
		return sec
	}
	return 30
}

func loadRaw() (seasideRaw, bool) {
	var raw seasideRaw
	if !store.Default().Load(sessionKey, &raw) || raw.NextRunAt <= 0 {
		return seasideRaw{}, false
	}
	return raw, true
}

// RestoreProgress 剩余补货等待秒数（对应 Session.restoreProgress）。
func RestoreProgress() int {
	raw, ok := loadRaw()
	if !ok {
		return 0
	}
	remain := int(raw.NextRunAt - time.Now().Unix())
	if remain > 0 {
		return remain
	}
	return 0
}

// ScheduleAfterRestock 记录下次补货调度（对应 Session.scheduleAfterRestock）。
func ScheduleAfterRestock(restockSec int) {
	if restockSec < 0 {
		restockSec = defaultRestockSec
	}
	waitSec := restockSec + bufferSec()
	nextRunAt := time.Now().Unix() + int64(waitSec)
	_ = store.Default().Set(sessionKey, map[string]any{
		"nextRunAt":  nextRunAt,
		"restockSec": restockSec,
		"bufferSec":  bufferSec(),
		"recordedAt": time.Now().Unix(),
	})
	logger.Info(sessionTag, " 下次补货调度 %ds 后（到期戳 %d）", waitSec, nextRunAt)
}

// CheckReady 检查是否可运行（对应 Session.checkReady；含启动首轮强制执行）。
func CheckReady() (bool, int) {
	if startupBypassPending {
		startupBypassPending = false
		startupBypassActive = true
		logger.Info(sessionTag, " 本次脚本启动首轮强制执行，忽略补货等待")
		return true, 0
	}
	raw, ok := loadRaw()
	if !ok {
		return true, 0
	}
	remain := int(raw.NextRunAt - time.Now().Unix())
	if remain <= 0 {
		return true, 0
	}
	logger.Debug(sessionTag, " 补货等待中，剩余 %ds", remain)
	return false, remain
}

// ConsumeStartupBypass 消费首轮强制标志（对应 Session.consumeStartupBypass）。
func ConsumeStartupBypass() bool {
	if startupBypassActive {
		startupBypassActive = false
		return true
	}
	return false
}

// Clear 清理会话（对应 Session.clear）。
func Clear() {
	_ = store.Default().Set(sessionKey, map[string]any{})
	logger.Info(sessionTag, " 会话已清理")
}

// Describe 会话状态摘要（对应 Session.describe）。
func Describe() string {
	raw, ok := loadRaw()
	if !ok {
		return "无补货等待"
	}
	remain := int(raw.NextRunAt - time.Now().Unix())
	if remain > 0 {
		return fmt.Sprintf("补货等待中，剩余 %ds", remain)
	}
	return "补货已到期"
}
