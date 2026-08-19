# 控制面板交互细化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 计划时段用 HH:mm 编辑；草稿有 Dirty 与持久化说明；胶囊显示最新状态而不是轮播旧日志；计划等待时禁用暂停（或让宿主真正暂停等待）。

**Architecture:** `Draft.Run.StartMinute/EndMinute` 仍是 0..1439 的单一数据源；控件只是投影。Dirty = 当前草稿相对 `lastSaved` 快照。持久化边界保持现状：`CommandSave` 只写任务开关；运行模式、安全阈值、优先级/单次上限仅随 `CommandStart` 进入 `runPolicy`。胶囊文案用 `Status.Message`。计划等待 `Outcome=="scheduled_wait"` 时 `pillPrimaryAction` 不发 `CommandPause`。

**Tech Stack:** `internal/ui/draft.go`、`panel_imgui.go`、`pill_interaction.go`、`main_host.go`

## Global Constraints

- 遵守总序。本计划在 [display-profile](2026-08-19-display-profile.md) 与 [pill-touch-and-commands](2026-08-19-pill-touch-and-commands.md) 之后执行。
- 不把运行模式/安全阈值偷偷写入 `userconfig`，除非另开需求。只把现状标清楚。
- 跨午夜窗口：当前 `Validate` 禁止 `EndMinute < StartMinute`。本计划不引入跨夜，HH:mm 控件必须遵守同一规则。

---

## File Structure

- Modify: `internal/ui/draft.go` — `func ClockMinutes(h, m int) int`、`func SplitMinutes(total int) (h, m int)`
- Modify: `internal/ui/ui.go` — `lastSaved Draft`、`Dirty() bool` 或 frame 上的 `Dirty bool` / `PersistHint`
- Modify: `internal/ui/panel_imgui.go` — 概览 HH:mm、Dirty 条、安全/任务持久化标注
- Modify: `internal/ui/pill_interaction.go` + `panel_imgui.go` — 最新消息；计划等待禁用暂停
- Modify: `main_host.go` — `pause()` 在 `rt==nil && running` 时发布说明，不静默
- Test: `draft_test.go`、`panel_test.go`、`pill_interaction_test.go`、`main_host_test.go`

---

### Task 1: 分钟 ↔ 时钟

**Files:** `internal/ui/draft.go`、`internal/ui/clock_test.go`

**Interfaces:**

```go
func SplitClock(minute int) (hour, min int)
func JoinClock(hour, min int) int
```

非法输入 clamp 到 0..23 / 0..59；`JoinClock` 结果 clamp 0..1439。

- [ ] **Step 1:**

```go
func TestJoinSplitClockRoundTrip(t *testing.T) {
    h, m := SplitClock(90)
    if h != 1 || m != 30 {
        t.Fatalf("%d:%d", h, m)
    }
    if JoinClock(1, 30) != 90 {
        t.Fatal("join")
    }
    if JoinClock(24, 99) != 1439 {
        t.Fatalf("got %d", JoinClock(24, 99))
    }
}
```

- [ ] **Step 2: FAIL → 实现 → PASS**
- [ ] **Step 3: Commit（仅当用户要求）** `feat(ui): convert schedule minutes to clock fields`

---

### Task 2: 概览页 HH:mm 控件

**Files:** `panel_imgui.go` `renderOverview`

把

```go
imgui.SliderIntV("开始分钟##schedule", &start, 0, 1439, "%d", 0)
```

换成两组 `SliderInt`：时 0..23、分 0..59，标签显示 `fmt.Sprintf("开始 %02d:%02d", h, m)`。写回 `StartMinute=JoinClock`。同一套给结束。

源码测试：`panel_imgui.go` 不得包含 `"开始分钟"` / `0, 1439`。必须包含 `JoinClock` 与 `"%02d:%02d"`。

校验失败文案已存在；保持「结束不能早于开始」。

- [ ] **Step 1: `TestAndroidScheduleUsesClockSliders` 源码契约 FAIL**
- [ ] **Step 2: 改 `renderOverview` → PASS**
- [ ] **Step 3: Commit（仅当用户要求）** `feat(ui): edit scheduled windows as HH:mm`

---

### Task 3: Dirty 与持久化标注

**Files:** `ui.go`、`draft.go`、`panel_imgui.go`、`panel_test.go`

**Interfaces:**

```go
func DraftsEqual(a, b Draft) bool
```

map 比较 Tasks；忽略未在 UI 编辑的 Production/HUD/Plans 也可，但为避免假 Dirty，比较 `Run`、`Safety`、`Tasks` 三组即可。

`Panel` 持有 `saved Draft`。`Open` 时 `saved = cloneDraft(draft)`。`PublishCommandResult` 当 `outcome=="config_saved"` 时 `saved = cloneDraft(p.draft)`（注意：保存的是已发出的草稿；用命令里的 settings 更准，但 Panel 在 emit 后草稿未变，clone 当前 draft 即可）。

`readFrame` 设 `frame.Dirty = !DraftsEqual(p.draft, p.saved)`。

页脚：

- Dirty：`有未保存修改`（colorWarning）
- 非 Dirty：`已保存`
- 另起一行灰色：`任务开关会写入本机存储。运行模式、安全阈值和任务优先级/单次上限仅本次运行生效。`

任务页卡片旁、安全页阈值旁用同一句短注。

- [ ] **Step 1:**

```go
func TestPanelDirtyTracksUnsavedDraftEdits(t *testing.T) {
    panel := NewPanel()
    _ = panel.Open(Snapshot{Settings: Default()}, func(Command) {})
    defer panel.Close()
    frame, _ := panel.readFrame()
    if frame.Dirty {
        t.Fatal("fresh panel must be clean")
    }
    frame.Draft.Run.Mode = RunOnce
    panel.writeFrame(frame)
    next, _ := panel.readFrame()
    if !next.Dirty {
        t.Fatal("mode change must mark dirty")
    }
    _ = panel.PublishPhase("idle", "config_saved", "配置已保存")
    next, _ = panel.readFrame()
    if next.Dirty {
        t.Fatal("config_saved must clear dirty")
    }
}
```

`config_saved` 清 Dirty 的前提是 `saved` 更新为当前 draft。实现时在 PublishPhase 里 clone。

- [ ] **Step 2: FAIL → 实现 DraftsEqual + saved + footer 文案 → PASS**
- [ ] **Step 3: Commit（仅当用户要求）** `feat(ui): show draft dirty state and persistence scope`

---

### Task 4: 胶囊显示最新消息

**Files:** `panel_imgui.go` `compactPillLogLimit`

删除按 `Unix()/3 % len(logs)` 的轮播。折叠胶囊主文案：

1. `Status.Message` 非空则用它
2. 否则最后一条 log
3. 否则 `scene · outcome · 动作 n`

展开后 `expandedPillDetail` 已优先 Message；保持。历史 logs 仅在展开区需要时再画滚动列表——第一版展开区继续单行最新消息即可，不要轮播。

源码测试：`panel_imgui.go` 不得包含 `Unix()/3`。

- [ ] **Step 1: 源码契约 + 纯函数测试**

把逻辑抽到 `pill_interaction.go`：

```go
func CompactPillHeadline(status RuntimeStatus, logs []string) string
```

```go
func TestCompactPillHeadlinePrefersLatestMessage(t *testing.T) {
    got := CompactPillHeadline(RuntimeStatus{Message: "引擎已启动", Scene: "x", Outcome: "old"}, []string{"a", "b"})
    if got != "引擎已启动" {
        t.Fatalf("%q", got)
    }
}
```

- [ ] **Step 2: FAIL → 实现 → imgui 调用 → PASS**
- [ ] **Step 3: Commit（仅当用户要求）** `fix(ui): show the latest status on the compact pill`

---

### Task 5: 计划等待禁用暂停

**Files:** `pill_interaction.go`、`main_host.go`、测试

`pillPrimaryAction`：

```go
func pillPrimaryAction(phase, outcome string) (string, CommandType, bool) // bool=enabled
```

当 `outcome=="scheduled_wait"`：标签 `"等待"`，command 仍 `CommandPause` 但 `enabled=false`。`applyPillHit` 在 `!enabled` 时不 emit。

展开态文案：`pillExpandedState` 对 scheduled_wait 返回 `"等待计划时段"`。

宿主兜底：

```go
func (h *Host) pause() {
    h.mu.Lock()
    rt := h.rt
    running := h.running
    h.mu.Unlock()
    if rt == nil {
        if running {
            h.panel.PublishPhase("running", "scheduled_wait", "计划等待中，无法暂停")
        }
        return
    }
    ...
}
```

不要在等待 goroutine 里接入 Runtime.Pause（第一版禁用即可）。若后续要真正暂停等待，再让 `waitForRunWindow` 听 `pauseCh`——超出本任务。

- [ ] **Step 1:**

```go
func TestScheduledWaitPrimaryDoesNotEmitPause(t *testing.T) {
    frame := panelFrame{Status: RuntimeStatus{Phase: "running", Outcome: "scheduled_wait"}, Compact: true, PillExpanded: true}
    commands := applyPillHit(&frame, pillHitPrimary, nil)
    if len(commands) != 0 {
        t.Fatalf("got %v", commands)
    }
}

func TestHostPauseDuringScheduledWaitKeepsWaiting(t *testing.T) {
    setupHostTest(t)
    panel := openTestPanel(t)
    host := newReadyHost(t, panel)
    host.nowFn = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local) }
    draft := ui.Default()
    draft.Run.Mode = ui.RunScheduled
    draft.Run.StartMinute = 600
    draft.Run.EndMinute = 700
    host.Handle(ui.Command{Type: ui.CommandStart, Settings: &draft})
    waitFor(t, func() bool { return panel.Status().Outcome == "scheduled_wait" })
    host.Handle(ui.Command{Type: ui.CommandPause})
    if panel.Status().Outcome != "scheduled_wait" {
        t.Fatalf("outcome=%q", panel.Status().Outcome)
    }
    if host.isRunning() != true {
        t.Fatal("wait must continue")
    }
    host.stop()
}
```

先读现有计划等待测试的时间注入方式，复用 `nowFn`，不要另造时钟。

- [ ] **Step 2: FAIL → 改 `applyPillHit` / `pause` → PASS**
- [ ] **Step 3: imgui `drawExpandedPillContent` 对 disabled 主按钮用灰色、不画 pause 图标（用时钟或空心圆）。**
- [ ] **Step 4: Commit（仅当用户要求）** `fix(ui): disable pause while waiting for a scheduled window`

---

## 验收

- 计划时段控件显示 `08:30` 而不是 `510`。
- 改运行模式后页脚出现「有未保存修改」；保存任务开关后变「已保存」，并写明仅开关持久化。
- 折叠胶囊不再每 3 秒换一条历史 log。
- 计划等待时点暂停无效果，文案为等待计划时段。
