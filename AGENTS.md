# AGENTS.md

Cookie Run: Kingdom (CRK) automation bot built on [AutoGo](https://autogo.cc/), a Go Android automation framework.

## Build & run — do not use plain `go build`

- Every package in `AutoGo/` is a **stub**: functions have empty bodies returning zero values. Real implementations are injected at compile time by the AutoGo toolchain, so local `go build`/`go run` compiles but does nothing useful.
- Real workflow: GoLand/IDEA with the AutoGo JetBrains plugin, Android device/emulator visible to adb, run/debug via **F7**. VSCode extension exists (`.vscode/settings.json` sets `AutoGo.targetPlatform: android`) but official docs use JetBrains.
- `go.mod` requires **Go 1.25+**.
- AutoGo-dependent code cannot be unit-tested locally — verify behavior only on device. `go vet`/`go build` passing proves nothing about runtime.

## Repo layout

- Root module `app`: `main.go` boots the ImGui panel, consumes its `Command`s, injects device adapters (截图隐身 frame source, AutoGo motion/log), and hosts the engine runtime (`main_host.go`, desktop-testable).
- `internal/ui/`: self-contained ImGui control panel + floating pill HUD, migrated from auto-cookie. Imports no domain package; hosts publish state via `Publish*` and consume user actions as `Command`s. `panel_imgui.go` (`//go:build android && cgo`) is the real renderer, `panel_nocgo.go` is the desktop stub — **desktop-testable**: `go test ./internal/ui`.
- `internal/vision/`: frame-based pure Go 比色/找色 (ADR-0003) — parses `"x|y|rrggbb-偏色"` 特征库 strings + sim, `FindMultiColor` region search. No AutoGo screen APIs; desktop-testable with saved frames.
- `internal/core/`: engine domain ported 结构直译 from the Lua project's `core/` — 守卫 `guard.go`, 调度器 `scheduler.go` (tasks + idle providers), 状态机 `statemachine.go` (KEEP/RETRY/DONE runner), 主循环 `runtime.go`. All desktop-testable.
- `internal/game/`: `taskbuilder.go` (standard task wrapper: 开关/就绪/让渡/离开广场) + `register.go` (M1 skeleton; M2a 注入网络联机状态不稳定守卫 + 矿山模块; M2b 全量业务模块). `catalog.go` 任务目录元数据 + 面板任务开关 ↔ userconfig 接线（`Catalog`/`ApplyTaskSwitches`/`LoadTaskSwitches`）；`detect.go` 识别诊断场景扫描（`DetectScene`，场景键与 `internal/ui` 的 SceneID 同步，由 main 包测试守护）。`kingdom/`/`popup/` 通用页（特征库+页面）；`mine/` 矿山特征库与首页；`mine/route/` 王国↔矿山路由（涉及勘查/开采页的路由因 Go 包循环限制随使用方存放）；`mine/{survey,mining,battle,jelly}/` 各矿山模块（页面/会话/任务，均桌面可测）。
- `internal/lib/`: 结构直译 from Lua `lib/` — `color.go` (frame facade over vision), `touch.go` (seam-injected AutoGo motion), `store.go` (JSON 会话存储), `userconfig.go` (defaults + persisted merge), `logger.go`/`status.go` (injectable sinks).
- `internal/config/`: 打包常量 (display/runtime/user defaults), mirrors `config.lua`.
- `AutoGo/`: local copy of the AutoGo SDK (`github.com/Dasongzi1366/AutoGo`, wired in via `replace`). Treat as third-party — don't edit; update through the plugin's SDK update feature.
- `CONTEXT.md`: UI-layer domain glossary (Chinese, with terms to avoid); `docs/adr/`: architectural decisions.
- `docs/autogo-api-2026.6.6.md`: full AutoGo API reference (Chinese) exported from autogo.cc. Read it before writing AutoGo calls.
- `resources/`: APK packaging assets — `META-INF/Android.toml` (appPackage, appName, autoRun, showFloatingBall), icons, `ui/index.html` (SDK default demo UI, not project code), per-ABI native libs (arm64-v8a / x86 / x86_64).
- `build/`: toolchain output (e.g. Android ELF binaries from quick-debug/package). Gitignored.

## Git & platform-managed files

- `AutoGo/`, `resources/`, `build/` are gitignored on purpose: initialized/synced by the AutoGo plugin (see `.gitignore`). Never edit or `git add` them.
- Commits follow Conventional Commits (`feat(ui):`, `docs:`, `chore:`) — match that style.

## Gotchas

- Only `AutoGo/imgui` uses cgo (links prebuilt `libimgui.so` from `resources/libs/<abi>/`); all other SDK packages are pure Go.
- UI docs and comments use CONTEXT.md vocabulary (控制面板, 悬浮胶囊, 截图隐身, 面板草稿, 命令) and respect its "_Avoid_" lists; code identifiers keep the `pill*` prefix from auto-cookie.
- APK mode: launch args arrive via `os.Args` (cold start through apkctl HTTP); use `apkctl.RegEvent` for pause/resume/stop; script logs go to `/sdcard/logs/<package>.log`.
