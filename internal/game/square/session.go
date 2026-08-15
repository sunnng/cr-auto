// Package square 对应 Lua 工程的 game/常规_布谷鸟广场/：广场特征库、页面、会话、路由与任务。
package square

import (
	"fmt"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/store"
)

const (
	doneKey   = "cuckoo_square_done_date"
	activeKey = "cuckoo_square_active"
	sessTag   = "[布谷鸟广场.会话]"
)

var nowFn = time.Now

// SetNow 注入时钟（测试用）。
func SetNow(fn func() time.Time) { nowFn = fn }

// squareActive 有效停留会话（对应 Lua Session.getActive 返回表）。
type squareActive struct {
	StartedAt      int64  `json:"startedAt"`
	AccumulatedSec int    `json:"accumulatedSec"`
	LastEnterAt    int64  `json:"lastEnterAt"` // 0 表示未在计时
	CheckedDate    string `json:"checkedDate"`
}

func nowUnix() int64 { return nowFn().Unix() }

// Today 今日日期串（YYYY-MM-DD，对应 Lua today）。
func Today() string { return nowFn().Format("2006-01-02") }

// IsDoneToday 今日广场任务是否已完成（对应 Session.isDoneToday）。
func IsDoneToday() bool {
	done, _ := store.Default().Get(doneKey, "")
	return done == Today()
}

// MarkDoneToday 标记今日已完成并清理会话（对应 Session.markDoneToday）。
func MarkDoneToday() {
	_ = store.Default().Set(doneKey, Today())
	Clear()
}

// GetActive 读取有效停留会话；无记录时返回 ok=false（对应 Session.getActive）。
func GetActive() (squareActive, bool) {
	var active squareActive
	if !store.Default().Load(activeKey, &active) {
		return squareActive{}, false
	}
	return active, true
}

// IsActive 是否已有停留会话（对应 Session.isActive）。
func IsActive() bool {
	_, ok := GetActive()
	return ok
}

// SaveActive 保存停留会话（对应 Session.save）。
func SaveActive(active squareActive) {
	_ = store.Default().Set(activeKey, active)
}

// Clear 清理停留会话（对应 Session.clear）。
func Clear() {
	_ = store.Default().Del(activeKey)
	logger.Info(sessTag, "clear: 停留会话已清理")
}

// ClearAll 清理今日完成标记与会话（对应 Session.clearAll）。
func ClearAll() {
	_ = store.Default().Del(doneKey)
	_ = store.Default().Del(activeKey)
}

// Ensure 获取或初始化停留会话（对应 Session.ensure）。
func Ensure() squareActive {
	if active, ok := GetActive(); ok {
		return active
	}
	active := squareActive{
		StartedAt:      nowUnix(),
		AccumulatedSec: 0,
		LastEnterAt:    0,
		CheckedDate:    "",
	}
	SaveActive(active)
	return active
}

// MarkCheckedToday 标记今日已初检（对应 Session.markCheckedToday）。
func MarkCheckedToday() {
	active := Ensure()
	active.CheckedDate = Today()
	SaveActive(active)
}

// HasCheckedToday 今日是否已初检（对应 Session.hasCheckedToday）。
func HasCheckedToday() bool {
	active, ok := GetActive()
	return ok && active.CheckedDate == Today()
}

// StartStay 开始/恢复广场有效停留计时（对应 Session.startStay）。
func StartStay() {
	active := Ensure()
	if active.LastEnterAt == 0 {
		active.LastEnterAt = nowUnix()
		SaveActive(active)
	}
}

// PauseStay 暂停计时并结算已在广场内停留的秒数（对应 Session.pauseStay）。
func PauseStay() {
	active, ok := GetActive()
	if !ok || active.LastEnterAt == 0 {
		return
	}
	active.AccumulatedSec += max(0, int(nowUnix()-active.LastEnterAt))
	active.LastEnterAt = 0
	SaveActive(active)
}

// ResetStayTimer 重置一轮奖励结算所需的有效停留计时（对应 Session.resetStayTimer）。
func ResetStayTimer() {
	active := Ensure()
	active.AccumulatedSec = 0
	active.LastEnterAt = nowUnix()
	SaveActive(active)
}

// StayElapsed 当前累计有效停留秒数（对应 Session.stayElapsed）。
func StayElapsed() int {
	active, ok := GetActive()
	if !ok {
		return 0
	}
	elapsed := active.AccumulatedSec
	if active.LastEnterAt > 0 {
		elapsed += max(0, int(nowUnix()-active.LastEnterAt))
	}
	return elapsed
}

// StayRemaining 距离 requiredSec 还剩多少秒（对应 Session.stayRemaining，缺省 60s）。
func StayRemaining(requiredSec int) int {
	if requiredSec <= 0 {
		requiredSec = 60
	}
	return max(0, requiredSec-StayElapsed())
}

// Describe 会话状态摘要（对应 Session.describe）。
func Describe() string {
	if IsDoneToday() {
		return "今日已完成"
	}
	active, ok := GetActive()
	if !ok {
		return "今日未完成，无挂机会话"
	}
	checked := "未初检"
	if active.CheckedDate == Today() {
		checked = "已初检"
	}
	return fmt.Sprintf("今日未完成，%s，有效停留 %ds", checked, StayElapsed())
}
