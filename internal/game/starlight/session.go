// Package starlight 对应 Lua 工程的 game/常规_梦幻繁星岛/：繁星岛坐标库、页面、会话、路由与任务。
package starlight

import (
	"time"

	"app/internal/lib/store"
)

const doneKey = "starlight_done_date"

// IsDoneToday 今日是否已执行（对应 Session.isDoneToday）。
func IsDoneToday() bool {
	done, _ := store.Default().Get(doneKey, "")
	return done == time.Now().Format("2006-01-02")
}

// MarkDoneToday 标记今日已完成（对应 Session.markDoneToday）。
func MarkDoneToday() {
	_ = store.Default().Set(doneKey, time.Now().Format("2006-01-02"))
}

// Clear 清理今日完成标记（对应 Session.clear）。
func Clear() {
	_ = store.Default().Del(doneKey)
}

// Describe 会话状态摘要（对应 Session.describe）。
func Describe() string {
	if IsDoneToday() {
		return "今日已完成"
	}
	return "今日未完成"
}
