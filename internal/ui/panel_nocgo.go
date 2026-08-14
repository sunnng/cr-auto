//go:build !android || !cgo

package ui

// The desktop test build has cgo disabled. AutoGo's Android packager enables
// cgo and selects panel_imgui.go with the real renderer.
func startPanelRenderer(*Panel) error { return nil }

func stopPanelRenderer() {}
