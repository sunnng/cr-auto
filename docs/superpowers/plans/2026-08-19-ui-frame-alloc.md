# 控制面板帧分配 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 降低每帧 Go 堆分配与重复 cgo 样式调用，悬浮胶囊路径接近零分配；不改变命令语义和启动确认。

**Architecture:** 渲染线程独占 UI 状态；宿主 mailbox 带 revision。`readFrame` 只在 revision 变化时克隆宿主投影。主题在 `imgui.Init` 后尽量只设一次；做不到就用包级静态颜色表，避免每帧 `make([]themeColor)`。`Catalog` / `ConfigTabs` 改为包级不可变切片。`cloneDraft` 只在 `emit` 时发生。

**Tech Stack:** `internal/ui` 纯 Go + `panel_imgui.go`

## Global Constraints

- 遵守总序。行为回归：`TestPanelWriteFrameDoesNotClobberConcurrentPublish`、启动确认测试必须保持绿。
- 禁止为了少分配而重新让 `writeFrame` 覆盖 logs/feedback。
- 胶囊路径：不 `cloneDraft`、不复制 catalog、不复制 logs 切片（可共享只读视图，因为渲染线程不改 logs）。

---

## File Structure

- Modify: `internal/ui/ui.go` — revision mailbox、`configTabs` 静态表、按需 clone
- Modify: `internal/ui/panel_imgui.go` — 主题一次化 / 静态颜色表；胶囊不走完整 `cloneDraft`
- Test: `internal/ui/panel_test.go` 增补 revision 测试
- Modify: `internal/ui/panel_android_source_test.go` — `pushKingdomTheme` 不再每帧 `[]themeColor{`

---

### Task 1: ConfigTabs 静态表

**Files:** `internal/ui/ui.go`、`internal/ui/panel_test.go`

**Interfaces:** `func ConfigTabs() []ConfigTabState` 签名不变，返回包级切片（调用方不得修改）。若担心外部乱写，返回值保持只读约定，测试用 `ConfigTabs()[0].ID`。

- [ ] **Step 1: 把现有 `TestConfigTabsExposeTheFourImplementedPages` 加上稳定性断言**

```go
func TestConfigTabsReturnsStableBackingStore(t *testing.T) {
    a := ConfigTabs()
    b := ConfigTabs()
    if &a[0] != &b[0] {
        t.Fatal("ConfigTabs must not allocate a new backing array every call")
    }
}
```

（若返回的是切片头指向同一 array，`&a[0]==&b[0]`。若实现选择每次 copy，本测试会失败——那就是本任务要修的。）

- [ ] **Step 2: `go test ./internal/ui -run TestConfigTabsReturnsStableBackingStore -count=1`**

Expected: FAIL（当前每次 `[]ConfigTabState{...}`）

- [ ] **Step 3: 包级 `var configTabs = []ConfigTabState{...}`，`ConfigTabs` 直接 return `configTabs`。`NormalizeConfigTab` 遍历该表。**

- [ ] **Step 4: `go test ./internal/ui -count=1` PASS**
- [ ] **Step 5: Commit（仅当用户要求）** `perf(ui): keep config tabs as immutable package data`

---

### Task 2: 宿主投影带 revision，无更新不克隆

**Files:** `internal/ui/ui.go`、`internal/ui/panel_test.go`

**Interfaces:**

在 `Panel` 增加：

```go
hostRevision uint64
```

`Publish*` / `PublishPhase` / `PublishCommandResult` / `PublishObservation` / `PublishCapabilities` / `PublishCatalog` / `PublishDisplay` / `publishDetectionLocked` 都 `p.hostRevision++`。

`readFrame`：

- UI 字段（draft/tab/pill/compact）仍每帧读。
- Draft：渲染器已持有可变草稿，**不要每帧 `cloneDraft`**。把 `panelFrame.Draft` 改成直接赋 `p.draft`（同进程渲染线程独占）。`emit` 时再 `cloneDraft`。
- Catalog / Logs / Capabilities / Status / Display / Preview：若 `frameHostRevision == p.hostRevision` 可复用；`readFrame` 是值返回，复用需要 `Panel` 缓存上一帧投影：

```go
cachedHost panelFrame // 只填宿主字段
cachedRev  uint64
```

更简单的最小实现（本任务采用）：

```go
func (p *Panel) readFrame() (panelFrame, bool) {
    // applyStartConfirmationLocked
    frame := panelFrame{
        Draft:        p.draft, // 不 clone
        Catalog:      p.catalog,
        Status:       p.status,
        Feedback:     p.feedback,
        Logs:         p.logs,
        Capabilities: p.capabilities,
        Preview:      p.preview,
        // UI fields copied by value
    }
    return frame, true
}
```

风险：测试里改 `frame.Draft` 再 `writeFrame` 是故意的；若跳过 clone，`frame.Draft.Tasks` 与 `p.draft.Tasks` 共享 map。当前 `writeFrame` 会 `cloneDraft(frame.Draft)` 写回，渲染器改 map 会直接改 Panel。这是可接受的，因为渲染线程独占。

**必须保留：** `emit` 对 `command.Settings` 的 `cloneDraft`（`TestPanelOwnsDraftAndEmitsIndependentSettingsCopy`）。

- [ ] **Step 1: 在 `TestPanelOwnsDraftAndEmitsIndependentSettingsCopy` 旁加**

```go
func TestReadFrameSharesDraftMapWithRenderer(t *testing.T) {
    panel := NewPanel()
    _ = panel.Open(Snapshot{Settings: Default()}, func(Command) {})
    defer panel.Close()
    frame, _ := panel.readFrame()
    frame.Draft.Run.Mode = RunOnce
    panel.writeFrame(frame)
    next, _ := panel.readFrame()
    if next.Draft.Run.Mode != RunOnce {
        t.Fatal("renderer draft edits must round-trip")
    }
}
```

此测试现在就会过。真正要加的是并发测试已存在。再加：

```go
func TestReadFrameDoesNotCloneCatalogWhenHostQuiet(t *testing.T) {
    panel := NewPanel()
    catalog := []TaskDescriptor{{ID: "a", Name: "A", Available: true, MaxRuns: 1}}
    _ = panel.Open(Snapshot{Settings: Default(), Catalog: catalog}, func(Command) {})
    defer panel.Close()
    a, _ := panel.readFrame()
    b, _ := panel.readFrame()
    if &a.Catalog[0] != &b.Catalog[0] {
        t.Fatal("quiet host must not recopy catalog each frame")
    }
}
```

当前 `append([]TaskDescriptor(nil), p.catalog...)` 会让它失败。

- [ ] **Step 2: Run FAIL**
- [ ] **Step 3: `readFrame` 对 catalog/logs 直接引用 `p.catalog`/`p.logs`。`PublishCatalog` 仍 copy 入站。`appendLogLocked` 仍 copy-on-write 截断 8 条。**
- [ ] **Step 4: `go test ./internal/ui -count=1` PASS**
- [ ] **Step 5: Commit（仅当用户要求）** `perf(ui): stop cloning host snapshots on quiet frames`

---

### Task 3: 主题静态化

**Files:** `internal/ui/panel_imgui.go`、`panel_android_source_test.go`

AutoGo imgui 若无 `Style()` 可变 Colors，则：

```go
var kingdomTheme = []themeColor{ ... } // 包级，不再函数内字面量

func pushKingdomTheme() int32 {
    for i := range kingdomTheme {
        imgui.PushStyleColorVec4(kingdomTheme[i].index, kingdomTheme[i].value)
    }
    return int32(len(kingdomTheme))
}
```

几何 `pushKingdomGeometry` 同理：胶囊路径 **不要** 调 `pushKingdomGeometry`（当前只有 config window 调）。确认 `renderPill` 不走 10 次 geometry Push——已经没有的话源码测试锁住。

若 SDK 有 `imgui.GetStyle()`：在 `startPanelRenderer` 的 `imgui.Init()` 成功后写 Colors，`renderFrame` 不再 `pushKingdomTheme`；局部金色按钮仍 Push/Pop。

- [ ] **Step 1: 源码测试**

```go
func TestAndroidThemeColorsArePackageLevel(t *testing.T) {
    source, _ := os.ReadFile("panel_imgui.go")
    if strings.Contains(string(source), "func pushKingdomTheme() int32 {\n\tcolors := []themeColor{") {
        t.Fatal("theme color table must not be allocated inside pushKingdomTheme")
    }
}
```

- [ ] **Step 2: FAIL then move table to var**
- [ ] **Step 3: `go test ./internal/ui -count=1` PASS**
- [ ] **Step 4: Commit（仅当用户要求）** `perf(ui): reuse the kingdom theme color table`

---

### Task 4: 胶囊路径零草稿拷贝

**Files:** `panel_imgui.go` `renderFrame`

当前 `readFrame` 已不 clone 的话本任务只是确认：`renderPill` 不调用 `cloneDraft`，除非 `applyPillHit` 发出 `CommandStart`（`pill_interaction.go` 已有 `cloneDraft`，保留）。

- [ ] **Step 1: 源码测试 `renderPill` 函数体内不含 `cloneDraft`（`applyPillHit` 在 `pill_interaction.go` 允许）**
- [ ] **Step 2: 若 imgui 里误调用则移走**
- [ ] **Step 3: PASS + Commit（仅当用户要求）** `perf(ui): keep compact pill frames off the draft clone path`

---

## 验收

- `go test ./internal/ui -count=1` 全绿。
- 安静宿主下连续两次 `readFrame` 的 catalog 底层数组相同。
- 启动确认与并发 Publish 测试不被破坏。
