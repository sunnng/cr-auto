# DisplayProfile 与识别覆盖层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把写死的 1600×900 收成宿主发布的 `DisplayProfile`；识别覆盖层按预览图像 bounds 换算；分辨率不匹配时 `DeviceProfileValid=false` 并禁止启动。

**Architecture:** `config.Static.Display` 仍是打包契约。宿主用 `device.GetDisplayInfo` 对照契约，把实际宽高/DPI/安全区发布到 `ui.DisplayProfile`。控制面板和悬浮胶囊只读 profile，不再出现裸 `1600`/`900` 字面量（测试常量除外）。

**Tech Stack:** `internal/ui`、`internal/config`、`main.go` `device.GetDisplayInfo`、ImGui 覆盖层

## Global Constraints

- 遵守总序 Global Constraints。
- 本项目只支持横屏 1600×900；不匹配即不可启动，不做缩放适配。
- `internal/ui` 不 import `internal/config`。契约宽高由宿主写入 `DisplayProfile.RequiredWidth/Height`。

---

## File Structure

- Modify: `internal/ui/ui.go` — `DisplayProfile` 类型、`Snapshot`、`PublishDisplay`、`panelFrame`
- Modify: `internal/ui/capabilities.go` — `EvaluateStart` 继续用 `DeviceProfileValid`（已存在）
- Modify: `internal/ui/panel_imgui.go` — 窗口位置/胶囊居中/覆盖层/字体比例使用 profile
- Modify: `internal/ui/panel_test.go` / 新建 `internal/ui/display_test.go`
- Modify: `main.go` — Open 时填入 profile
- Modify: `internal/ui/panel_android_source_test.go` — 禁止覆盖层写死 `/1600` `/900`

默认 profile（宿主未发布时）：

```go
type DisplayProfile struct {
    Width, Height           int
    RequiredWidth, RequiredHeight int
    DPI                     int
    SafeMinX, SafeMinY      int
    SafeMaxX, SafeMaxY      int
}

func DefaultDisplayProfile() DisplayProfile {
    return DisplayProfile{
        Width: 1600, Height: 900,
        RequiredWidth: 1600, RequiredHeight: 900,
        DPI: 240,
        SafeMinX: 0, SafeMinY: 0, SafeMaxX: 1600, SafeMaxY: 900,
    }
}
```

---

### Task 1: DisplayProfile 接缝

**Files:**
- Modify: `internal/ui/ui.go`
- Test: `internal/ui/display_test.go`

**Interfaces:**
- Consumes: 现有 `Snapshot` / `PublishCapabilities`
- Produces: `DisplayProfile`、`func (p *Panel) PublishDisplay(profile DisplayProfile) error`

- [ ] **Step 1: Write the failing test**

```go
package ui

import "testing"

func TestPanelProjectsDisplayProfile(t *testing.T) {
    panel := NewPanel()
    if err := panel.Open(Snapshot{
        Settings: Default(),
        Display:  DisplayProfile{Width: 1280, Height: 720, RequiredWidth: 1600, RequiredHeight: 900},
    }, func(Command) {}); err != nil {
        t.Fatal(err)
    }
    defer panel.Close()
    frame, ok := panel.readFrame()
    if !ok || frame.Display.Width != 1280 || frame.Display.Height != 720 {
        t.Fatalf("display not projected: %+v", frame.Display)
    }
    if err := panel.PublishDisplay(DefaultDisplayProfile()); err != nil {
        t.Fatal(err)
    }
    frame, ok = panel.readFrame()
    if !ok || frame.Display.Width != 1600 {
        t.Fatalf("publish display lost: %+v", frame.Display)
    }
}

func TestWriteFrameDoesNotClobberDisplayProfile(t *testing.T) {
    panel := NewPanel()
    if err := panel.Open(Snapshot{Settings: Default(), Display: DefaultDisplayProfile()}, func(Command) {}); err != nil {
        t.Fatal(err)
    }
    defer panel.Close()
    frame, _ := panel.readFrame()
    _ = panel.PublishDisplay(DisplayProfile{Width: 1, Height: 1, RequiredWidth: 1600, RequiredHeight: 900})
    panel.writeFrame(frame)
    next, _ := panel.readFrame()
    if next.Display.Width != 1 {
        t.Fatalf("writeFrame overwrote display: %+v", next.Display)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestPanelProjectsDisplayProfile -count=1`

Expected: FAIL `undefined: DisplayProfile`

- [ ] **Step 3: Add type, Snapshot field, panel field, readFrame copy, writeFrame 不回写 Display, PublishDisplay**

`Open` 时若 `snapshot.Display.Width==0`，填 `DefaultDisplayProfile()`。

- [ ] **Step 4: Run tests**

Run: `go test ./internal/ui -count=1`

Expected: PASS

- [ ] **Step 5: Commit（仅当用户要求）**

```
feat(ui): publish a display profile through the snapshot seam
```

---

### Task 2: 覆盖层用图像 bounds

**Files:**
- Modify: `internal/ui/panel_imgui.go` `drawDetectionOverlay`
- Create: `internal/ui/overlay.go`（纯函数，桌面可测）
- Test: `internal/ui/overlay_test.go`
- Modify: `internal/ui/panel_android_source_test.go`

**Interfaces:**
- Produces:

```go
func OverlayScale(imageW, imageH int, destW, destH float32) (scaleX, scaleY float32)
```

零宽高时 scale=1，避免除零。

- [ ] **Step 1: Write the failing test**

```go
func TestOverlayScaleUsesImageBoundsNotFixedDisplay(t *testing.T) {
    sx, sy := OverlayScale(800, 450, 400, 225)
    if sx != 0.5 || sy != 0.5 {
        t.Fatalf("scale=(%v,%v)", sx, sy)
    }
    sx, sy = OverlayScale(0, 0, 400, 225)
    if sx != 1 || sy != 1 {
        t.Fatalf("empty image must not divide by zero: %v %v", sx, sy)
    }
}
```

- [ ] **Step 2: Run to fail**

Run: `go test ./internal/ui -run TestOverlayScale -count=1`

Expected: FAIL `undefined: OverlayScale`

- [ ] **Step 3: Implement OverlayScale; imgui `toScreen` 改为**

```go
bounds := image.Rect(0, 0, 1600, 900)
if preview.Image != nil {
    bounds = preview.Image.Bounds()
}
sx, sy := OverlayScale(bounds.Dx(), bounds.Dy(), size.X, size.Y)
// X: origin.X + float32(x)*sx
```

`drawDetectionOverlay` 增加 `imageW, imageH int` 参数，从 `preview.Image` 传入。槽位 42px 框同样用 `42*sx` 而不是 `size.X*42/1600`。

源码测试：`panel_imgui.go` 不得包含 `size.X/1600` 或 `size.Y/900`。

- [ ] **Step 4: Run**

Run: `go test ./internal/ui -count=1`

Expected: PASS

- [ ] **Step 5: Commit（仅当用户要求）**

```
fix(ui): map detection overlay from preview image bounds
```

---

### Task 3: 控制面板与胶囊布局改读 profile

**Files:**
- Modify: `internal/ui/panel_imgui.go`：`SetNextWindowPos/Size`、胶囊 `windowX := (display.Width-width)/2`、页眉 `fmt.Sprintf("CN %d×%d  ·  %ddpi", ...)`
- Modify: `internal/ui/pill_interaction.go` 不硬编码屏幕宽；居中在 imgui 侧完成
- Test: `panel_android_source_test.go` 要求 `frame.Display.Width` 出现在胶囊定位处

面板几何保持相对 profile：

- 窗口宽 = `min(1160, profile.Width-80)`
- 窗口高 = `min(780, profile.Height-120)`
- 位置 X = `(profile.Width - width) / 2`，Y = `profile.SafeMinY + 55`（若 SafeMinY=0 则 55）

不匹配时页眉用 `colorRed` 显示实际分辨率。

- [ ] **Step 1: 源码契约测试**

```go
func TestAndroidPanelLayoutsFromDisplayProfile(t *testing.T) {
    source, err := os.ReadFile("panel_imgui.go")
    if err != nil { t.Fatal(err) }
    content := string(source)
    if !strings.Contains(content, "frame.Display.Width") {
        t.Fatal("panel/pill layout must read DisplayProfile")
    }
    if strings.Contains(content, "windowX := (1600 - width) / 2") {
        t.Fatal("pill must not hardcode 1600 for centering")
    }
}
```

- [ ] **Step 2: Run to fail**

Run: `go test ./internal/ui -run TestAndroidPanelLayoutsFromDisplayProfile -count=1`

Expected: FAIL

- [ ] **Step 3: Replace literals; keep 1600 only inside `DefaultDisplayProfile`**

- [ ] **Step 4: Run `go test ./internal/ui -count=1`**

Expected: PASS

- [ ] **Step 5: Commit（仅当用户要求）**

```
feat(ui): layout the control panel from the host display profile
```

---

### Task 4: 宿主对照契约并拒绝不匹配设备

**Files:**
- Modify: `main.go`（已有 `SetDeviceProfileValid`）
- Modify: `main.go` `panel.Open` 传入 `Display:`
- Test: `main_host_test.go`

`DeviceProfileValid = width==RequiredWidth && height==RequiredHeight`。旋转后宽高对调视为不匹配。

```go
func profileFromDevice(width, height, dpi int) ui.DisplayProfile {
    p := ui.DefaultDisplayProfile()
    p.Width, p.Height, p.DPI = width, height, dpi
    if p.Width <= 0 || p.Height <= 0 {
        p.Width, p.Height = p.RequiredWidth, p.RequiredHeight
    }
    p.SafeMaxX, p.SafeMaxY = p.Width, p.Height
    return p
}
```

- [ ] **Step 1: Test**

```go
func TestProfileFromDeviceMarksMismatch(t *testing.T) {
    p := profileFromDevice(1280, 720, 240)
    if p.Width != 1280 || p.RequiredWidth != 1600 {
        t.Fatalf("%+v", p)
    }
    host := NewHost(openTestPanel(t))
    host.SetDeviceProfileValid(p.Width == p.RequiredWidth && p.Height == p.RequiredHeight)
    caps := host.CurrentCapabilities()
    if caps.DeviceProfileValid {
        t.Fatal("1280x720 must not be a valid device profile")
    }
}
```

`profileFromDevice` 若只在 `main.go`，把函数放到 `main_host.go` 以便测试。

- [ ] **Step 2: Fail then implement**
- [ ] **Step 3: `go test . -count=1` PASS**
- [ ] **Step 4: Commit（仅当用户要求）**

```
feat(host): reject start when the device is not 1600x900
```

---

## 验收

- 覆盖层在非 1600 宽的诊断图上锚点仍落在对应像素。
- 1280×720 设备：安全页「设备分辨率 / 不可用」，启动按钮禁用。
- 1600×900：页眉显示契约分辨率，可启动（还取决于计划 1 的 OCR/守卫）。
