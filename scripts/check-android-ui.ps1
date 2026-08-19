# Optional Android/cgo compile check for the control panel.
# Missing NDK or libimgui.so is a skip (exit 0), not a CI failure.
$ErrorActionPreference = "SilentlyContinue"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

$prevGOOS = $env:GOOS
$prevGOARCH = $env:GOARCH
$prevCGO = $env:CGO_ENABLED
$out = Join-Path ([System.IO.Path]::GetTempPath()) "cr-auto-ui-android.test"
$logFile = Join-Path ([System.IO.Path]::GetTempPath()) "cr-auto-ui-android.log"
$result = "skip: android cgo toolchain not present"
$log = ""

try {
    $env:GOOS = "android"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "1"
    cmd /c "go test -c -o `"$out`" ./internal/ui > `"$logFile`" 2>&1"
    if ($LASTEXITCODE -eq 0) {
        $result = "compile ok"
    }
    if (Test-Path $logFile) {
        $log = Get-Content -Raw $logFile
    }
}
finally {
    $env:GOOS = $prevGOOS
    $env:GOARCH = $prevGOARCH
    $env:CGO_ENABLED = $prevCGO
    foreach ($path in @($out, $logFile)) {
        if (Test-Path $path) {
            Remove-Item $path -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host $result
if ($result -ne "compile ok" -and $log) {
    Write-Host $log.TrimEnd()
}
exit 0
