# AGENTS.md

Cookie Run: Kingdom (CRK) automation bot built on [AutoGo](https://autogo.cc/), a Go Android automation framework.

## Build & run — do not use plain `go build`

- Every package in `AutoGo/` is a **stub**: functions have empty bodies returning zero values. Real implementations are injected at compile time by the AutoGo toolchain, so local `go build`/`go run` compiles but does nothing useful.
- Real workflow: GoLand/IDEA with the AutoGo JetBrains plugin, Android device/emulator visible to adb, run/debug via **F7**. VSCode extension exists (`.vscode/settings.json` sets `AutoGo.targetPlatform: android`) but official docs use JetBrains.
- `go.mod` requires **Go 1.25+**.
- AutoGo-dependent code cannot be unit-tested locally — verify behavior only on device. `go vet`/`go build` passing proves nothing about runtime.

## Repo layout

- Root module `app`: your script code lives here; `main.go` is the entrypoint (boots the ImGui panel and consumes its commands).
- `internal/ui/`: self-contained ImGui control panel + floating pill HUD, migrated from auto-cookie. Imports no domain package; hosts publish state via `Publish*` and consume user actions as `Command`s. `panel_imgui.go` (`//go:build android && cgo`) is the real renderer, `panel_nocgo.go` is the desktop stub — **the package is unit-testable on desktop** (`go test ./internal/ui`).
- `AutoGo/`: local copy of the AutoGo SDK (`github.com/Dasongzi1366/AutoGo`, wired in via `replace`). Treat as third-party — don't edit; update through the plugin's SDK update feature.
- `CONTEXT.md`: UI-layer domain glossary; `docs/adr/`: architectural decisions.
- `docs/autogo-api-2026.6.6.md`: full AutoGo API reference (Chinese) exported from autogo.cc. Read it before writing AutoGo calls.
- `resources/`: APK packaging assets — `META-INF/Android.toml` (appPackage, appName, autoRun, showFloatingBall), icons, `ui/index.html` (SDK default demo UI, not project code), per-ABI native libs (arm64-v8a / x86 / x86_64: yolo, ppocr, opencv, ncnn, imgui, goeval, dotocr).

## Gotchas

- Only `AutoGo/imgui` uses cgo (links prebuilt `libimgui.so` from `resources/libs/<abi>/`); all other SDK packages are pure Go.
- APK mode: launch args arrive via `os.Args` (cold start through apkctl HTTP); use `apkctl.RegEvent` for pause/resume/stop; script logs go to `/sdcard/logs/<package>.log`.
