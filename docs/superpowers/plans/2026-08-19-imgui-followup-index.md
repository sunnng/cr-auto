# 控制面板后续实施计划总序

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement each linked plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 P0/P1 正确性已经落地之后，按依赖顺序把剩余能力接入、分辨率契约、帧分配、触控命令和交互细化做成可独立验收的交付。

**Architecture:** `internal/ui` 继续不 import 领域包；宿主通过 `Capabilities` / `DisplayProfile` / `Publish*` / `Command` 接缝注入。引擎侧能力（OCR、资源消费保护、敏感页面停机）必须先变成真实适配器，启动门禁才会放行。

**Tech Stack:** Go 1.25、AutoGo ImGui/cgo（Android）、`internal/ui` 桌面 `go test`、宿主 `go test .`

## Global Constraints

- 文案与注释使用 `CONTEXT.md` 词汇：控制面板、悬浮胶囊、截图隐身、面板草稿、命令。代码标识符继续用 `pill*`。
- `internal/ui` 禁止 import `internal/game` / `internal/core` / `internal/lib/*`。
- AutoGo 依赖代码不能用桌面 `go test` 证明真机行为；桌面测试覆盖纯 Go 接缝，Android 行为用源码契约测试 + 真机清单。
- 不要编辑 `AutoGo/`、`resources/`、`build/`。
- 不要用 `go build` / `go run` 当设备验收；设备端走 AutoGo 插件 F7。
- 提交信息用 Conventional Commits（`feat(ui):`、`feat(host):`、`fix(ui):`、`test:`）。**仅在用户明确要求 commit 时执行各计划里的 Commit 步骤。**
- 每个任务先写失败测试，再写实现。
- 已完成、不要重做：能力快照、启动确认（`starting` + `RequestID`）、`writeFrame` 不覆盖宿主日志/反馈、控制面板与悬浮胶囊互斥渲染。

## 当前事实（2026-08-19）

- `EvaluateStart` 在分辨率/图色未就绪，或已启用但不可用的任务时拒绝启动。资源消费保护与敏感页面停机未采集时仍报告「等待设备验收」，但不拦截启动。
- `detectCapabilities()` 读 `ocr.Ready()` 与 `game.SafetyGuardsReady()`；安全特征库仍为空，两道锁不得画成「已启用」。
- `CommandStop` 只停引擎并展开控制面板；`CommandExit` 才结束脚本。页脚「退出脚本」需 2 秒内再点一次；胶囊停止需长按 800ms。胶囊按钮 52px，悬停会刷新自动收起。
- 控制面板位置、胶囊居中、识别覆盖层读 `DisplayProfile`；分辨率不匹配时 `DeviceProfileValid=false` 并禁止启动。
- `readFrame` 共享草稿/目录/日志底层数据，不再每帧深拷贝；`cloneDraft` 只在发出命令时发生。主题颜色表为包级静态，悬浮胶囊不走控制面板几何 Push。

## 推荐执行顺序

| 序号 | 计划 | 为什么必须先做 | 独立交付物 |
|------|------|----------------|------------|
| 1 | [ocr-and-safety-capabilities](2026-08-19-ocr-and-safety-capabilities.md) | 没有它，真机永远无法启动 | OCR 注入 + 两道守卫真正注册，`Capabilities` 变真 |
| 2 | [display-profile](2026-08-19-display-profile.md) | 布局和覆盖层都依赖分辨率契约 | `DisplayProfile` 接缝、覆盖层用图像 bounds、不匹配禁止启动 |
| 3 | [ui-frame-alloc](2026-08-19-ui-frame-alloc.md) | 纯性能，不改变命令语义 | 胶囊路径低分配、主题/目录不再每帧复制 |
| 4 | [pill-touch-and-commands](2026-08-19-pill-touch-and-commands.md) | 依赖停止/退出拆分，改 `main.go` 命令面 | 触控尺寸、停止≠退出、二次确认、悬停暂停收起 |
| 5 | [panel-interaction](2026-08-19-panel-interaction.md) | 依赖 DisplayProfile 的布局，但不依赖帧分配 | HH:mm、Dirty、持久化标注、胶囊最新消息、计划等待禁用暂停 |
| 6 | [ui-test-hardening](2026-08-19-ui-test-hardening.md) | 给 1–5 做回归网，可并行补测试 | cgo 编译检查、纹理释放、计划等待按钮、真机清单 |

不要并行做 1 和 4：它们都会改 `main.go` 命令循环。2 与 3 可在 1 之后并行。5 应在 2 和 4 之后。6 贯穿全程，每个计划自带测试；计划 6 只收口跨计划缺口。

## 文件归属（后续计划共用）

| 文件 | 职责 |
|------|------|
| `internal/ui/ui.go` | Panel 接缝：`Snapshot` / `Command` / `readFrame` / `writeFrame` / 启动确认 |
| `internal/ui/capabilities.go` | `Capabilities` / `EvaluateStart` / `TaskAvailability` |
| `internal/ui/draft.go` | 面板草稿与校验 |
| `internal/ui/pill_interaction.go` | 胶囊几何与命中（纯 Go，可测） |
| `internal/ui/panel_imgui.go` | Android ImGui 渲染器（`android && cgo`） |
| `internal/ui/panel_android_source_test.go` | 无 cgo 的渲染器源码契约 |
| `main.go` | 设备适配注入、命令队列、进程退出 |
| `main_host.go` | 命令 → 引擎；能力探测；计划等待 |
| `internal/game/register.go` | 守卫与任务注册 |
| `internal/game/popup/` | 弹窗特征库 |
| `internal/config/config.go` | `Static.Display` 1600×900 契约 |
| `CONTEXT.md` | 只在新增用户可见术语时改 |

## 验收总闸

全部计划完成后，桌面：

```
go test ./internal/ui ./internal/game ./internal/lib/ocr ./internal/lib/color . -count=1
```

真机（AutoGo F7）：分辨率匹配可启动；不匹配被拒绝并留下控制面板；OCR 任务可启用；安全页显示「已启用」而不是「等待设备验收」；停止任务不退出脚本。
