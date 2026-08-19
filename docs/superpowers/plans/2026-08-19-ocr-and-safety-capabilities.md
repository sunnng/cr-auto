# OCR 与安全守卫接入 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把设备 OCR、资源消费保护、敏感页面停机做成真实宿主能力，让 `EvaluateStart` 在真机上可以放行，而不再靠测试探针撒谎。

**Architecture:** OCR 走已有 `ocr.Engine` 接缝，设备端用 AutoGo `ppocr` 适配器在 `main.go` 注入。两道安全守卫按 `popup.UnstableNetwork` 同形：特征库 + `RegisterAll` 注册 + 命中后取消运行。`detectCapabilities` 根据 `ocr.Ready()` 和 `game.SafetyGuardsReady()` 报告，不再写死 `false`。

**Tech Stack:** `internal/lib/ocr`、`github.com/Dasongzi1366/AutoGo/ppocr`、`internal/game/popup`、`internal/core.Guard`、`main.go` / `main_host.go`

## Global Constraints

- 遵守 `docs/superpowers/plans/2026-08-19-imgui-followup-index.md` 的 Global Constraints。
- 没有真实特征点时，`SafetyGuardsReady()` 必须仍为 false；禁止用空特征把能力标成已启用。
- 安全守卫命中必须走取消，不能点「购买/支付」确认。
- 桌面测试用 `color.ScriptedScreen` 与假 OCR，不 import AutoGo。

---

## File Structure

- Create: `internal/lib/ocr/ppocr_adapter.go` — 纯转换：把 `[]ppocrResult` 变成 `ocr.Engine` 需要的 raw 字符串（为了桌面可测，定义本地 DTO，不在此文件 import AutoGo）
- Create: `internal/lib/ocr/ppocr_adapter_test.go`
- Create: `internal/game/popup/safety.go` — 资源消费 / 敏感页面特征 + `Ready()`
- Create: `internal/game/popup/safety_test.go`
- Create: `internal/game/safety_stop.go` — 守卫命中后的停机回调接缝
- Modify: `internal/game/register.go` — 注册两道守卫
- Modify: `main_host.go` — `detectCapabilities` 读 `ocr.Ready()` / `game.SafetyGuardsReady()`；`SetSafetyStop` 接到 `cancel`
- Modify: `main.go` — `ppocr.New` + `ocr.SetEngine`；Open 前注入
- Modify: `internal/game/register_guard_test.go` 或新建 `internal/game/register_safety_test.go`

### Task 1: OCR 结果串转换（桌面可测）

**Files:**
- Create: `internal/lib/ocr/ppocr_adapter.go`
- Test: `internal/lib/ocr/ppocr_adapter_test.go`

**Interfaces:**
- Consumes: 无
- Produces:

```go
type RegionText struct {
    Words string
    X, Y, W, H int
}

func FormatScan(items []RegionText, returnType string) string
func FindCenter(items []RegionText, text string) (x, y int, ok bool)
```

- [ ] **Step 1: Write the failing test**

```go
package ocr

import "testing"

func TestFormatScanTextJoinsLabels(t *testing.T) {
    raw := FormatScan([]RegionText{{Words: "配置", X: 10, Y: 20, W: 40, H: 16}}, ReturnTypeText)
    if raw != "配置" {
        t.Fatalf("text=%q", raw)
    }
}

func TestFormatScanJSONIncludesLocation(t *testing.T) {
    raw := FormatScan([]RegionText{{Words: "配置", X: 10, Y: 20, W: 40, H: 16}}, ReturnTypeJSON)
    items, text := decode(raw)
    if text != "配置" || len(items) != 1 {
        t.Fatalf("items=%+v text=%q", items, text)
    }
    if _, _, ok := localCenter(items[0].Location); !ok {
        t.Fatal("json location must decode to a center")
    }
}

func TestFindCenterMatchesSubstring(t *testing.T) {
    x, y, ok := FindCenter([]RegionText{{Words: "王国竞技场", X: 100, Y: 200, W: 80, H: 20}}, "竞技场")
    if !ok || x != 140 || y != 210 {
        t.Fatalf("center=(%d,%d) ok=%v", x, y, ok)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lib/ocr -run TestFormatScan -count=1`

Expected: FAIL `undefined: FormatScan`

- [ ] **Step 3: Write minimal implementation**

```go
package ocr

import (
    "encoding/json"
    "strings"
)

type RegionText struct {
    Words  string
    X, Y, W, H int
}

func FormatScan(items []RegionText, returnType string) string {
    switch returnType {
    case ReturnTypeText, ReturnTypeNum:
        parts := make([]string, 0, len(items))
        for _, item := range items {
            if item.Words != "" {
                parts = append(parts, item.Words)
            }
        }
        return strings.Join(parts, "")
    default:
        type locItem struct {
            Words    string  `json:"words"`
            Location [][]int `json:"location"`
        }
        out := make([]locItem, 0, len(items))
        for _, item := range items {
            x2, y2 := item.X+item.W, item.Y+item.H
            out = append(out, locItem{
                Words: item.Words,
                Location: [][]int{
                    {item.X, item.Y}, {x2, item.Y}, {x2, y2}, {item.X, y2},
                },
            })
        }
        raw, _ := json.Marshal(out)
        return string(raw)
    }
}

func FindCenter(items []RegionText, text string) (int, int, bool) {
    if text == "" {
        return 0, 0, false
    }
    for _, item := range items {
        if item.Words != "" && strings.Contains(item.Words, text) {
            return item.X + item.W/2, item.Y + item.H/2, true
        }
    }
    return 0, 0, false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/lib/ocr -count=1`

Expected: PASS

- [ ] **Step 5: Commit（仅当用户要求）**

```
feat(ocr): convert region OCR hits into the engine scan string
```

---

### Task 2: 设备 OCR 适配器接到 `ocr.SetEngine`

**Files:**
- Create: `main_ocr.go`（与 `main.go` 同包，避免把 AutoGo import 推进 `internal/lib/ocr`）
- Modify: `main.go` — Open 面板前注入引擎
- Modify: `main_host.go` — `detectCapabilities` 已读 `ocr.Ready()`，确认不要写死 OCR
- Test: `internal/lib/ocr/ocr_test.go` 已有 `Ready()`；再补 `main_host_test.go` 里真实探测路径

**Interfaces:**
- Consumes: `FormatScan` / `FindCenter` / `ocr.Engine`
- Produces:

```go
type deviceOCR struct{ scan func(x1, y1, x2, y2 int) []ocr.RegionText }

func (d *deviceOCR) Scan(rect image.Rectangle, mode int, returnType string) (string, error)
func (d *deviceOCR) FindTapPoint(text string, rect image.Rectangle) (int, int, bool)
```

`main.go` 里用 `ppocr.New("v5")`；`New` 返回 nil 则不 `SetEngine`，OCR 保持未就绪。

- [ ] **Step 1: Write the failing host test**

在 `main_host_test.go` 追加（不要走 `SetCapabilitiesProbe`）：

```go
func TestDetectCapabilitiesReportsOCRFromEngine(t *testing.T) {
    setupHostTest(t)
    ocr.SetEngine(nil)
    t.Cleanup(func() { ocr.SetEngine(nil) })
    caps := detectCapabilities()
    if caps.OCRReady {
        t.Fatal("OCR must be unread until an engine is injected")
    }
    ocr.SetEngine(&fakeHostOCR{})
    caps = detectCapabilities()
    if !caps.OCRReady {
        t.Fatal("injected OCR engine must set OCRReady")
    }
}

type fakeHostOCR struct{}

func (*fakeHostOCR) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
    return "", nil
}
func (*fakeHostOCR) FindTapPoint(string, image.Rectangle) (int, int, bool) { return 0, 0, false }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test . -run TestDetectCapabilitiesReportsOCRFromEngine -count=1`

Expected: FAIL（当前 `detectCapabilities` 的 `OCRReady: ocr.Ready()` 其实已经会过。若已过，把本测试当作回归锁，不要改探测逻辑。）

若 `detectCapabilities` 仍写死 `OCRReady: false`，本步必须红。

- [ ] **Step 3: Implement `deviceOCR` in `main_ocr.go`**

```go
package main

import (
    "image"
    "app/internal/lib/ocr"
    "github.com/Dasongzi1366/AutoGo/ppocr"
)

type deviceOCR struct {
    engine *ppocr.Ppocr
}

func (d *deviceOCR) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
    if d == nil || d.engine == nil || rect.Empty() {
        return "", nil
    }
    hits := d.engine.Ocr(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, "", displayID)
    regions := make([]ocr.RegionText, 0, len(hits))
    for _, hit := range hits {
        regions = append(regions, ocr.RegionText{
            Words: hit.Label, X: hit.X, Y: hit.Y, W: hit.Width, H: hit.Height,
        })
    }
    return ocr.FormatScan(regions, returnType), nil
}

func (d *deviceOCR) FindTapPoint(text string, rect image.Rectangle) (int, int, bool) {
    if d == nil || d.engine == nil {
        return 0, 0, false
    }
    hits := d.engine.Ocr(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, "", displayID)
    regions := make([]ocr.RegionText, 0, len(hits))
    for _, hit := range hits {
        regions = append(regions, ocr.RegionText{
            Words: hit.Label, X: hit.X, Y: hit.Y, W: hit.Width, H: hit.Height,
        })
    }
    return ocr.FindCenter(regions, text)
}

func injectDeviceOCR() {
    engine := ppocr.New("v5")
    if engine == nil {
        return
    }
    ocr.SetEngine(&deviceOCR{engine: engine})
}
```

在 `main.go` 里 `color.SetScreen(...)` 之后、`panel.Open` 之前调用 `injectDeviceOCR()`。

- [ ] **Step 4: Run tests**

Run: `go test ./internal/lib/ocr . -count=1`

Expected: PASS

- [ ] **Step 5: Commit（仅当用户要求）**

```
feat(host): inject AutoGo ppocr into the OCR seam
```

---

### Task 3: 安全特征库与 Ready 门闩

**Files:**
- Create: `internal/game/popup/safety.go`
- Test: `internal/game/popup/safety_test.go`

**Interfaces:**
- Consumes: `vision.Feature`
- Produces:

```go
func ResourceSpendDef() []vision.Feature
func SensitivePageDef() []vision.Feature
func SafetyFeaturesReady() bool
```

`SafetyFeaturesReady` 为 true 当且仅当两组特征都至少有一条 `Points != ""` 且 `Sim > 0`。占位空串必须返回 false。

- [ ] **Step 1: Write the failing test**

```go
package popup

import "testing"

func TestSafetyFeaturesReadyRequiresNonEmptyPoints(t *testing.T) {
    if SafetyFeaturesReady() {
        t.Fatal("empty or placeholder features must not claim readiness")
    }
}
```

先实现空特征，测试应通过（Ready=false）。然后再写「填入测试点后 Ready=true」的表驱动：把 `SafetyFeaturesReady` 做成对传入切片的纯函数更干净：

```go
func FeaturesReady(groups ...[]vision.Feature) bool
```

生产 `SafetyFeaturesReady()` 调用 `FeaturesReady(ResourceSpendDef(), SensitivePageDef())`。

```go
func TestFeaturesReady(t *testing.T) {
    empty := []vision.Feature{{Sim: 0.9}}
    filled := []vision.Feature{{Points: "10|10|ffffff-101010", Sim: 0.9}}
    if FeaturesReady(empty, filled) {
        t.Fatal("any empty group must fail")
    }
    if !FeaturesReady(filled, filled) {
        t.Fatal("both groups filled must pass")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/game/popup -run TestFeaturesReady -count=1`

Expected: FAIL `undefined: FeaturesReady`

- [ ] **Step 3: Implement**

```go
package popup

import "app/internal/vision"

func ResourceSpendDef() []vision.Feature {
    // 真机验收前保持空。填入商店/水晶购买确认框的多点比色串后，Ready 才会变真。
    return nil
}

func SensitivePageDef() []vision.Feature {
    // 账号、支付、客服等敏感页。空 = 未验收。
    return nil
}

func FeaturesReady(groups ...[]vision.Feature) bool {
    if len(groups) == 0 {
        return false
    }
    for _, group := range groups {
        ready := false
        for _, f := range group {
            if f.Points != "" && f.Sim > 0 {
                ready = true
                break
            }
        }
        if !ready {
            return false
        }
    }
    return true
}

func SafetyFeaturesReady() bool {
    return FeaturesReady(ResourceSpendDef(), SensitivePageDef())
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/game/popup -count=1`

Expected: PASS

- [ ] **Step 5: Commit（仅当用户要求）**

```
feat(game): gate safety-guard readiness on captured color features
```

---

### Task 4: 停机回调 + 守卫注册

**Files:**
- Create: `internal/game/safety_stop.go`
- Modify: `internal/game/register.go`
- Test: `internal/game/register_safety_test.go`
- Modify: `main_host.go` `startEngine`：注册 `game.SetSafetyStop`

**Interfaces:**
- Consumes: `popup.ResourceSpendDef` / `SensitivePageDef` / `core.Guard.Register`
- Produces:

```go
func SetSafetyStop(fn func(reason string))
func SafetyGuardsReady() bool
```

`SafetyGuardsReady` = `popup.SafetyFeaturesReady()`（特征未采集时注册了也算未就绪）。

守卫行为：

- 资源消费：命中后 `touch.Back()` 或点击特征附带的取消区（若还没有取消区，先 Back），然后 `SetSafetyStop("命中资源消费保护")`。第一版直接停机，比误点购买更安全。
- 敏感页面：命中后立即 `SetSafetyStop("进入敏感页面，已停止")`，不要继续点。

优先级高于「网络联机状态不稳定」（例如 20 和 19；现网不稳定是 10）。

- [ ] **Step 1: Write the failing test**

`internal/game/register_safety_test.go`：用已有 `setupRegisterTest`，临时把 `ResourceSpendDef` 换成测试不可行（不要改生产函数签名去注入）。更稳的做法：让 `RegisterSafetyGuards(g *core.Guard)` 接受特征参数。

```go
func RegisterSafetyGuards(g *core.Guard, resource, sensitive []vision.Feature)
```

`RegisterAll` 调用 `RegisterSafetyGuards(g, popup.ResourceSpendDef(), popup.SensitivePageDef())`。

测试：

```go
func TestRegisterSafetyGuardsStopsOnSensitiveMatch(t *testing.T) {
    setupRegisterTest(t)
    var reason string
    SetSafetyStop(func(r string) { reason = r })
    t.Cleanup(func() { SetSafetyStop(nil) })

    g := core.NewGuard()
    feat := vision.Feature{Points: kingdom.Home().Feature.Points, Sim: 0.9} // 用已有王国主城点，只为让 ScriptedScreen 能命中
    RegisterSafetyGuards(g, nil, []vision.Feature{feat})

    img := image.NewNRGBA(image.Rect(0, 0, 1600, 900))
    paint known points... // 复用 main_host_test 的 paintPointSpecs，或 color.ScriptedScreen 脚本
    libcolor.SetScreen(libcolor.NewScriptedScreen())
    // 按 ScriptedScreen 的 API 把该特征标为命中

    if !g.Check() {
        t.Fatal("sensitive page must hit")
    }
    if reason == "" {
        t.Fatal("safety stop callback must run")
    }
}
```

读 `internal/lib/color/scripted.go` 的现有 API，按它的命中方式写，不要发明新 Screen。

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/game -run TestRegisterSafetyGuardsStopsOnSensitiveMatch -count=1`

Expected: FAIL `undefined: RegisterSafetyGuards`

- [ ] **Step 3: Implement `SetSafetyStop` + `RegisterSafetyGuards`**

```go
var safetyStop func(string)

func SetSafetyStop(fn func(reason string)) { safetyStop = fn }

func requestSafetyStop(reason string) {
    if safetyStop != nil {
        safetyStop(reason)
    }
}

func RegisterSafetyGuards(g *core.Guard, resource, sensitive []vision.Feature) {
    if len(resource) > 0 {
        g.Register("资源消费保护", resource, func() {
            touch.Back()
            requestSafetyStop("命中资源消费保护，已停止")
        }, 20)
    }
    if len(sensitive) > 0 {
        g.Register("敏感页面停机", sensitive, func() {
            requestSafetyStop("进入敏感页面，已停止")
        }, 19)
    }
}

func SafetyGuardsReady() bool { return popup.SafetyFeaturesReady() }
```

`core.Guard.Register` 的 `feature` 类型是 `any`，已支持 `[]vision.Feature`（见 `guard.go` 注释）。实现前读 `Guard.Check` 对 slice 的处理，必要时把 slice 拆成多条同优先级守卫。

- [ ] **Step 4: Wire host cancel**

在 `startEngine` 里 `rt.Register` 之前：

```go
game.SetSafetyStop(func(reason string) {
    h.setStopReason(reason)
    cancel()
})
```

`finishRun` 里 `game.SetSafetyStop(nil)`。

`detectCapabilities`：

```go
ResourceGuardReady:      game.SafetyGuardsReady(),
SensitivePageGuardReady: game.SafetyGuardsReady(),
```

两道守卫共用一个 Ready：任一特征组空则两道都不宣称已启用。

- [ ] **Step 5: Run tests**

Run: `go test ./internal/game . -count=1`

Expected: PASS。`TestHostRejectsStartWhenSafetyGuardsMissing` 在 `SafetyGuardsReady()==false` 时仍应拒绝启动。

- [ ] **Step 6: Commit（仅当用户要求）**

```
feat(game): register resource and sensitive-page safety guards
```

---

### Task 5: 真机采集特征（人工，阻塞 Ready）

**Files:**
- Modify: `internal/game/popup/safety.go` 填入 `Points` 串
- Test: `internal/game/popup/safety_test.go` 增加 `TestSafetyFeaturesReadyAfterCapture`（Points 非空才 Enable）

没有 1600×900 真机截图时，**不要编造色点**。本任务的完成标准是：

1. 在商店水晶购买确认框、账号/支付页各采集一组 `"x|y|rrggbb-偏色"`，Sim=0.9。
2. 填入 `ResourceSpendDef` / `SensitivePageDef`。
3. `go test ./internal/game/popup -run TestSafetyFeaturesReady -count=1` 变为 Ready=true。
4. 真机 F7：安全页两道锁显示「已启用」；故意走进商店购买框时引擎停止且不点确认。

- [ ] **Step 1: Capture on device**（人 + 识别诊断页「立即识别」）
- [ ] **Step 2: Paste points into `safety.go`**
- [ ] **Step 3: `go test ./internal/game ./internal/ui . -count=1`**
- [ ] **Step 4: Commit（仅当用户要求）**

```
feat(game): fill safety-guard color features from device capture
```

---

## 验收

- `ocr.Ready()` 在 `ppocr.New("v5")` 成功后为 true，任务目录不再全部「等待设备 OCR 验收」。
- 特征未采集时启动仍被拒绝（诚实）。
- 特征采集后 `EvaluateStart` 不再因两道锁拦截；命中敏感页/购买框会停机。
