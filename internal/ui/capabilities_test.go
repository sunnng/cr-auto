package ui

import (
	"strings"
	"testing"
)

func readyCapabilities() Capabilities {
	return Capabilities{
		OCRReady:                true,
		VisionReady:             true,
		ResourceGuardReady:      true,
		SensitivePageGuardReady: true,
		DeviceProfileValid:      true,
	}
}

func TestEvaluateStartAllowsWhenCapabilitiesReady(t *testing.T) {
	draft := Default()
	draft.Tasks["mine_survey"] = TaskSetting{Enabled: true, MaxRuns: 1}
	catalog := []TaskDescriptor{{
		ID:        "mine_survey",
		Name:      "矿山勘查",
		Available: true,
	}}
	gate := EvaluateStart(readyCapabilities(), draft, catalog)
	if !gate.Allowed || len(gate.Reasons) != 0 {
		t.Fatalf("expected start allowed, got %+v", gate)
	}
}

func TestEvaluateStartAllowsWhenSafetyGuardsNotReady(t *testing.T) {
	caps := readyCapabilities()
	caps.ResourceGuardReady = false
	caps.SensitivePageGuardReady = false
	gate := EvaluateStart(caps, Default(), nil)
	if !gate.Allowed {
		t.Fatalf("uncaptured resource/sensitive-page guards must not block start, got %v", gate.Reasons)
	}
	joined := strings.Join(gate.Reasons, " ")
	if strings.Contains(joined, "资源消费保护") || strings.Contains(joined, "敏感页面停机") {
		t.Fatalf("start reasons must not mention deferred safety guards, got %v", gate.Reasons)
	}
}

func TestEvaluateStartBlocksUnavailableEnabledTasks(t *testing.T) {
	draft := Default()
	draft.Tasks["biscuit"] = TaskSetting{Enabled: true, MaxRuns: 1}
	catalog := []TaskDescriptor{{
		ID:                "biscuit",
		Name:              "洗脆饼词条",
		Available:         false,
		UnavailableReason: "等待设备 OCR 验收",
	}}
	gate := EvaluateStart(readyCapabilities(), draft, catalog)
	if gate.Allowed {
		t.Fatal("enabled unavailable tasks must block start")
	}
	joined := strings.Join(gate.Reasons, " ")
	if !strings.Contains(joined, "洗脆饼词条") || !strings.Contains(joined, "OCR") {
		t.Fatalf("blocking reason must name the task and capability, got %v", gate.Reasons)
	}
}

func TestEvaluateStartIgnoresDisabledUnavailableTasks(t *testing.T) {
	draft := Default()
	draft.Tasks["biscuit"] = TaskSetting{Enabled: false, MaxRuns: 1}
	catalog := []TaskDescriptor{{
		ID:                "biscuit",
		Name:              "洗脆饼词条",
		Available:         false,
		UnavailableReason: "等待设备 OCR 验收",
	}}
	gate := EvaluateStart(readyCapabilities(), draft, catalog)
	if !gate.Allowed {
		t.Fatalf("disabled unavailable tasks must not block observation start, got %v", gate.Reasons)
	}
}

func TestTaskAvailabilityWaitsForDeviceOCR(t *testing.T) {
	caps := readyCapabilities()
	caps.OCRReady = false
	available, reason := TaskAvailability(caps, true)
	if available {
		t.Fatal("OCR-backed tasks must not appear available before device OCR is accepted")
	}
	if reason != "等待设备 OCR 验收" {
		t.Fatalf("reason=%q", reason)
	}
	available, reason = TaskAvailability(caps, false)
	if !available || reason != "" {
		t.Fatalf("vision-only tasks stay available without OCR: available=%v reason=%q", available, reason)
	}
}

func TestCapabilityStatusLabels(t *testing.T) {
	caps := Capabilities{}
	if got := caps.OCRStatus(); got != CapabilityPending {
		t.Fatalf("missing OCR must be pending device acceptance, got %s", got)
	}
	if got := caps.VisionStatus(); got != CapabilityUnavailable {
		t.Fatalf("missing vision must be unavailable, got %s", got)
	}
	if got := caps.ResourceGuardStatus(); got != CapabilityPending {
		t.Fatalf("missing resource guard must be pending, got %s", got)
	}
	if got := caps.DeviceProfileStatus(); got != CapabilityUnavailable {
		t.Fatalf("invalid device profile must be unavailable, got %s", got)
	}
	caps = readyCapabilities()
	if got := caps.OCRStatus(); got != CapabilityEnabled {
		t.Fatalf("ready OCR must be enabled, got %s", got)
	}
	if label := CapabilityStatusLabel(CapabilityPending); label != "等待设备验收" {
		t.Fatalf("label=%q", label)
	}
}
