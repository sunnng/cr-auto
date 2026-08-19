# 悬浮胶囊触控与停止/退出拆分 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 胶囊按钮达到触屏可点尺寸；停止任务不再退出脚本；退出脚本与停止任务分成两条命令，并带误触保护；指针悬停时暂停自动收起；文本按像素宽度截断。

**Architecture:** `CommandStop` 只停引擎并展开控制面板。新增 `CommandExit` 才结束 `main` 循环。页脚「退出」发 `CommandExit`；胶囊停止发 `CommandStop`。长按 800ms 确认停止/退出。`pill_interaction.go` 继续纯 Go 可测。

**Tech Stack:** `internal/ui/pill_interaction.go`、`panel_imgui.go`、`main.go`、`main_host.go`

## Global Constraints

- 遵守总序与 `CONTEXT.md`。
- **行为变更：** 今日 `main.go` 收到 `CommandStop` 会 `host.stop(); return`，等于退出脚本。本计划之后这是错误行为。
- 触屏按钮最小边长 48 逻辑像素（推荐 52）。展开胶囊高度随之增加。
- 自动收起计时器在指针位于胶囊 `body` 内或按下时不递减。

---

## File Structure

- Modify: `internal/ui/ui.go` — `CommandExit CommandType = "engine.exit"`
- Modify: `internal/ui/pill_interaction.go` — 尺寸、长按、悬停暂停收起
- Modify: `internal/ui/pill_interaction_test.go`
- Modify: `internal/ui/panel_imgui.go` — 页脚退出发 `CommandExit`；`limitPillTextWidth`
- Modify: `main.go` — `CommandStop` 走 `host.Handle`；`CommandExit` 才 `return`
- Modify: `main_host.go` — `stop()` 成功后 `PublishPhase("idle","stopped",...)` 并让 Panel 取消 compact（已有 `finishRun`）
- Modify: `CONTEXT.md` — 命令列表补「退出脚本」
- Test: `main_host_test.go` — 停止后进程逻辑上仍可再次 `CommandStart`

---

### Task 1: 命令面拆分

**Files:** `ui.go`、`main.go`、`main_host.go`、`panel_imgui.go`、测试

**Interfaces:**

```go
const (
    CommandStop CommandType = "engine.stop"  // 停引擎，留在进程
    CommandExit CommandType = "engine.exit"  // 结束脚本
)
```

- [ ] **Step 1: 失败测试**

`main_host_test.go`：

```go
func TestHostStopDoesNotNeedToBeFollowedByProcessExit(t *testing.T) {
    setupHostTest(t)
    panel := openTestPanel(t)
    host := newReadyHost(t, panel)
    host.Handle(ui.Command{Type: ui.CommandStart})
    waitFor(t, func() bool { return host.isRunning() })
    host.Handle(ui.Command{Type: ui.CommandStop})
    waitFor(t, func() bool { return !host.isRunning() })
    host.Handle(ui.Command{Type: ui.CommandStart})
    waitFor(t, func() bool { return host.isRunning() })
}
```

这个在 `Handle(CommandStop)` 已调用 `h.stop()` 时就会过。真正锁行为的是 **源码/文档测试 + main 循环**：

`panel_android_source_test.go`：

```go
if !strings.Contains(string(source), `Command{Type: CommandExit}`) {
    t.Fatal("footer exit must emit CommandExit, not CommandStop")
}
```

以及 `main.go` 测试做不到（无导出循环）。用 `main` 包测试读源码：

新建 `main_source_test.go`：

```go
func TestMainLoopExitsOnlyOnCommandExit(t *testing.T) {
    source, err := os.ReadFile("main.go")
    if err != nil { t.Fatal(err) }
    s := string(source)
    if strings.Contains(s, "if command.Type == ui.CommandStop {\n\t\t\t\thost.stop()\n\t\t\t\treturn") {
        t.Fatal("CommandStop must not return from main")
    }
    if !strings.Contains(s, "ui.CommandExit") {
        t.Fatal("main must handle CommandExit")
    }
}
```

- [ ] **Step 2: FAIL**
- [ ] **Step 3: 实现**

`ui.go` 增加 `CommandExit`。

`panel_imgui.go` 页脚：

```go
if centeredButton("退出脚本", "footer-exit", ...) {
    commands = append(commands, Command{Type: CommandExit})
}
```

`pill_interaction.go` `pillHitStop` 仍 `CommandStop`。

`main.go`：

```go
case command := <-commands:
    if command.Type == ui.CommandExit {
        host.stop()
        return
    }
    host.Handle(command)
```

`Host.Handle` `CommandStop` → `h.stop()`（已有）。`finishRun` 发布 idle 后，Panel 需展开：在 `PublishPhase("idle","stopped",...)` 之后，若没有 pending start，设置 `compact=false`。在 `ui.go` 的 `PublishCommandResult`/`PublishPhase`：当 `phase=="idle" && outcome=="stopped"` 时 `p.compact=false`。加测试：

```go
func TestPanelExpandsWhenHostStops(t *testing.T) {
    panel := NewPanel()
    _ = panel.Open(Snapshot{Settings: Default()}, func(Command) {})
    defer panel.Close()
    panel.SetCompact(true)
    _ = panel.PublishPhase("idle", "stopped", "引擎已停止")
    frame, _ := panel.readFrame()
    if frame.Compact {
        t.Fatal("stopped engine must restore the control panel")
    }
}
```

- [ ] **Step 4: `go test ./internal/ui . -count=1` PASS**
- [ ] **Step 5: Commit（仅当用户要求）** `feat(ui): separate engine stop from script exit`

---

### Task 2: 胶囊按钮 52px 与命中区

**Files:** `pill_interaction.go`、`pill_interaction_test.go`、`panel_imgui.go` 展开高度

常量改为：

```go
pillCollapsedHeight = 56
pillExpandedHeight  = 112
pillButtonSize      = float32(52)
pillButtonGap       = float32(8)
```

`expandedPillLayoutForBounds` 用 `pillButtonSize` 代替 34/38。`TestExpandedPillPrioritizesInfoWithCompactControlGrid` 改为断言高度 ≥ 48。

- [ ] **Step 1: 改测试期望为 52 高度，旧 34 必须失败**
- [ ] **Step 2: `go test ./internal/ui -run TestExpandedPill -count=1` FAIL**
- [ ] **Step 3: 改 layout 常量与 `drawPillButton` 圆角**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit（仅当用户要求）** `feat(ui): enlarge compact pill controls for touch`

---

### Task 3: 长按确认停止/退出

**Files:** `pill_interaction.go`、页脚 imgui（退出可二次点击：第一次标 `exitArmed`，2 秒内再点才 emit）

胶囊停止：

```go
const pillConfirmHold = 800 * time.Millisecond
```

`pillPointerState` 增加 `holdSince time.Time`。`update` 在 `down && target==pillHitStop` 期间若 `now.Sub(holdSince) >= pillConfirmHold` 才返回 `pillHitStop`。短按不发命令。

桌面测试用注入 `now` 或直接测纯函数：

```go
func stopConfirmed(downFor time.Duration) bool {
    return downFor >= pillConfirmHold
}
```

页脚「退出脚本」：`panelFrame.ExitArmedUntil time.Time`。第一次点击只 arm 并显示「再点一次退出」；第二次在 2s 内才 `CommandExit`。

- [ ] **Step 1: 测试短按 stop 不产生命令；800ms 产生 CommandStop**
- [ ] **Step 2: FAIL**
- [ ] **Step 3: 实现 hold 逻辑；页脚 arm**
- [ ] **Step 4: PASS**
- [ ] **Step 5: Commit（仅当用户要求）** `feat(ui): require hold or double-tap before stop and exit`

---

### Task 4: 悬停暂停自动收起

**Files:** `pill_interaction.go` `advancePillAnimation` 附近、`panel_imgui.go` `renderPill`

当 `visualLayout.body.contains(point)` 或 `IsMouseDown` 时，把 `PillCollapseAt = now.Add(pillAutoCollapse)`（刷新deadline），不要在 hover 时 `collapsePill`。

现逻辑：`now.After(frame.PillCollapseAt)` 且未按下才收起。改为还要求 `!hovered`。

- [ ] **Step 1:**

```go
func TestHoverRefreshesAutoCollapseDeadline(t *testing.T) {
    now := time.Unix(100, 0)
    frame := &panelFrame{PillExpanded: true, PillCollapseAt: now.Add(time.Second)}
    refreshPillCollapse(frame, now.Add(500*time.Millisecond), true)
    if !frame.PillCollapseAt.After(now.Add(pillAutoCollapse-time.Second)) {
        t.Fatalf("deadline not refreshed: %v", frame.PillCollapseAt)
    }
}
```

- [ ] **Step 2: FAIL → 实现 `refreshPillCollapse` → PASS**
- [ ] **Step 3: imgui `renderPill` 每帧在 hovered 时调用**
- [ ] **Step 4: Commit（仅当用户要求）** `fix(ui): pause pill auto-collapse while the pointer is over it`

---

### Task 5: 按像素宽度截断

**Files:** 新建 `internal/ui/textfit.go`（纯函数）、`panel_imgui.go` 用 `imgui.CalcTextSize`

```go
func FitRunes(value string, maxWidth float32, measure func(string) float32) string
```

`measure` 在测试里用 `float32(len([]rune(s))*10)`；imgui 里闭包 `imgui.CalcTextSize(s).X`。

- [ ] **Step 1:**

```go
func TestFitRunesEllipsizesToPixelWidth(t *testing.T) {
    measure := func(s string) float32 { return float32(len([]rune(s)) * 10) }
    got := FitRunes("abcdefghij", 50, measure) // 5 cells * 10 = 50, need ellipsis
    if got == "abcdefghij" || !strings.HasSuffix(got, "…") {
        t.Fatalf("got %q", got)
    }
    if measure(got) > 50 {
        t.Fatalf("still too wide: %q", got)
    }
}
```

- [ ] **Step 2: FAIL → 实现二分或从尾部删 rune → PASS**
- [ ] **Step 3: `compactPillLogLimit` / `expandedPillDetail` 改为像素版；保留 rune 版给无 imgui 测试或删除调用**
- [ ] **Step 4: Commit（仅当用户要求）** `fix(ui): ellipsize pill text by measured pixel width`

---

## 验收

- 停止任务后控制面板回来，可再次启动。
- 「退出脚本」需二次确认，且才会结束 AutoGo 脚本。
- 胶囊按钮 ≥ 48px；手指按住停止约 0.8s 才生效。
- 指针停在胶囊上不会 6 秒后自己收起。
