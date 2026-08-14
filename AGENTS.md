# AGENTS.md

Cookie Run: Kingdom (CRK) automation bot built on [AutoGo](https://autogo.cc/), a Go Android automation framework.

## Build & run — do not use plain `go build`

- Every package in `AutoGo/` is a **stub**: functions have empty bodies returning zero values. Real implementations are injected at compile time by the AutoGo toolchain, so local `go build`/`go run` compiles but does nothing useful.
- Real workflow: GoLand/IDEA with the AutoGo JetBrains plugin, Android device/emulator visible to adb, run/debug via **F7**. VSCode extension exists (`.vscode/settings.json` sets `AutoGo.targetPlatform: android`) but official docs use JetBrains.
- `go.mod` requires **Go 1.25+**.
- AutoGo-dependent code cannot be unit-tested locally — verify behavior only on device. `go vet`/`go build` passing proves nothing about runtime.

## Repo layout

- Root module `app`: `main.go` boots the ImGui panel, consumes its `Command`s, and registers `apkctl.RegEvent` lifecycle hooks. Engine/task domain arrives in later milestones (see `docs/adr/`).
- `internal/ui/`: self-contained ImGui control panel + floating pill HUD, migrated from auto-cookie. Imports no domain package; hosts publish state via `Publish*` and consume user actions as `Command`s. `panel_imgui.go` (`//go:build android && cgo`) is the real renderer, `panel_nocgo.go` is the desktop stub — **desktop-testable**: `go test ./internal/ui`.
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
