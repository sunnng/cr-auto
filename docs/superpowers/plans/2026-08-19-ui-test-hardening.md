# 控制面板测试加固 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补上源码契约测试覆盖不到的行为网：计划等待按钮、Android/cgo 编译、纹理生命周期、分辨率不匹配、真机截图隐身清单。不替代计划 1–5 里的单测。

**Architecture:** 能桌面测的继续 `go test`。cgo 用 `GOOS=android` 语法检查或 CI 脚本，不在开发机链 `libimgui.so` 时允许 skip。真机项写成 `docs/superpowers/plans/2026-08-19-device-qa-checklist.md` 人工闸，不假装自动化已覆盖。

**Tech Stack:** `go test`、`go test -exec` 不强制；`GOOS=android CGO_ENABLED=1` 可选

## Global Constraints

- 遵守总序。已存在的测试不要删：并发 Publish、启动失败不收起、面板胶囊互斥源码契约。
- 真机清单不是代码任务的完成条件；代码任务完成 = 桌面测试绿 + 清单文件已写入。
- 不要为了测 ImGui 点击而引入假 cgo。

---

## File Structure

- Modify: 各包 `*_test.go`（只加缺口）
- Create: `docs/superpowers/plans/2026-08-19-device-qa-checklist.md`
- Create: `internal/ui/texture_lifecycle_test.go` 仅测 `syncDetectionPreviewTexture` 抽出来的纯逻辑（revision 相等则不重建）
- Optional: `.github` 或仓库脚本 `scripts/check-android-ui.sh` — 若仓库没有 CI，就只写 `go test` 注释在 `AGENTS.md` 一行（用户没要求改 AGENTS 则把命令写在清单里）

---

### Task 1: 计划等待按钮（若计划 5 已写则跳过）

确认 `TestScheduledWaitPrimaryDoesNotEmitPause` 与 `TestHostPauseDuringScheduledWaitKeepsWaiting` 存在。缺失则按计划 5 Task 5 补上。

- [ ] **Step 1: `go test ./internal/ui . -run ScheduledWait -count=1`**
- [ ] **Step 2: 无测试则补，有则标记完成**

---

### Task 2: 纹理 revision 纯逻辑

`syncDetectionPreviewTexture` 依赖 imgui，不能桌面跑。抽出：

```go
func ShouldRebuildTexture(currentRev, incomingRev uint64, hasTexture, hasImage bool) bool {
    if !hasImage {
        return hasTexture // caller deletes
    }
    if !hasTexture {
        return true
    }
    return incomingRev != currentRev
}
```

放在 `internal/ui/overlay.go` 或 `detection.go`。imgui 函数变成对它的薄包装。

- [ ] **Step 1:**

```go
func TestShouldRebuildTexture(t *testing.T) {
    if ShouldRebuildTexture(3, 3, true, true) {
        t.Fatal("same revision must keep texture")
    }
    if !ShouldRebuildTexture(3, 4, true, true) {
        t.Fatal("newer revision must rebuild")
    }
    if !ShouldRebuildTexture(0, 1, false, true) {
        t.Fatal("missing texture must build")
    }
}
```

- [ ] **Step 2: FAIL → 实现 → imgui 调用 → 源码测试 `ShouldRebuildTexture(` 出现在 `panel_imgui.go` → PASS**
- [ ] **Step 3: Commit（仅当用户要求）** `test(ui): lock detection texture rebuild on revision`

关闭后释放：现有 `stopPanelRenderer` 已 `Delete`。源码测试锁住 `detectionPreviewTexture.Delete()` 在 `stopPanelRenderer`。

```go
func TestAndroidStopRendererDeletesPreviewTexture(t *testing.T) {
    source, _ := os.ReadFile("panel_imgui.go")
    fn := source[bytes.Index(source, []byte("func stopPanelRenderer()")):]
    fn = fn[:bytes.Index(fn, []byte("\nfunc "))]
    if !strings.Contains(string(fn), "detectionPreviewTexture.Delete()") {
        t.Fatal("stop must delete the preview texture")
    }
}
```

---

### Task 3: Android 编译检查（可 skip）

新建 `scripts/check-android-ui.ps1`（本机是 Windows）：

```powershell
$env:GOOS = "android"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "1"
go test -c -o NUL ./internal/ui
```

若缺 NDK / `libimgui.so`，脚本以 exit 0 打印 `skip: android cgo toolchain not present`，不要红 CI。

`panel_android_source_test.go` 保持为无工具链时的主回归。

- [ ] **Step 1: 添加脚本**
- [ ] **Step 2: 在本机跑一次，记录 skip 或 compile ok**
- [ ] **Step 3: Commit（仅当用户要求）** `chore: add optional android cgo compile check for the panel`

---

### Task 4: 分辨率与旋转用例

依赖计划 2 的 `profileFromDevice`。

```go
func TestLandscapeMismatchAndSwapAreInvalid(t *testing.T) {
    cases := [][2]int{{1280, 720}, {900, 1600}, {0, 0}}
    for _, c := range cases {
        p := profileFromDevice(c[0], c[1], 240)
        valid := p.Width == p.RequiredWidth && p.Height == p.RequiredHeight
        if valid {
            t.Fatalf("unexpected valid %+v", c)
        }
    }
    p := profileFromDevice(1600, 900, 240)
    if p.Width != 1600 || p.Height != 900 {
        t.Fatalf("%+v", p)
    }
}
```

- [ ] **Step 1–4: TDD 若函数尚无则先做计划 2；有则本测试锁回归**

---

### Task 5: 真机清单

Create: `docs/superpowers/plans/2026-08-19-device-qa-checklist.md`

内容必须包含（每项 checkbox）：

1. 1600×900：启动成功后才出现悬浮胶囊；启动失败（存盘错误）控制面板仍在。
2. 非 1600×900 或旋转成 900×1600：启动禁用，原因含分辨率。
3. 截图隐身：识别诊断「立即识别」与引擎一轮，截图不含控制面板/胶囊像素（目视）。
4. 连续点「立即识别」10 次：无纹理泄漏、无崩溃。
5. 关闭脚本：无残留 ImGui 窗口。
6. OCR 注入成功后任务可启用；ppocr.New 失败时任务全部「等待设备 OCR 验收」。
7. 安全特征采集后：进商店购买框停机且未点确认。
8. 停止任务 ≠ 退出脚本；退出需二次确认。
9. 计划等待：胶囊显示等待，暂停无效。
10. 手指点胶囊按钮可稳定命中（≥48px）。

- [ ] **Step 1: 写下清单文件**
- [ ] **Step 2: Commit（仅当用户要求）** `docs: add device QA checklist for the control panel`

---

## 已覆盖、不要重复造轮子

| 评审建议 | 现有测试 |
|----------|----------|
| 并发 Publish 丢更新 | `TestPanelWriteFrameDoesNotClobberConcurrentPublish` |
| 启动失败不收起 | `TestPanelStartErrorKeepsPanelExpanded` |
| 面板/胶囊互斥 | `TestAndroidPanelProvidesCompactDynamicPill` 中的 `if frame.Compact` 契约 |
| 能力门禁 | `TestEvaluateStart*`、`TestHostAllowsStartWhenSafetyGuardsMissing` |

## 验收

- `go test ./internal/ui . -count=1` 绿。
- 真机清单文件存在；F7 跑清单不在本计划的自动化范围内。
