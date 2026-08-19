package ui

import "fmt"

// Capabilities is the host-projected ability snapshot. The UI package never
// probes devices or engines; it only decides what to show and whether start
// is allowed from these facts.
type Capabilities struct {
	OCRReady                bool
	VisionReady             bool
	ResourceGuardReady      bool
	SensitivePageGuardReady bool
	DeviceProfileValid      bool
}

type CapabilityStatus string

const (
	CapabilityEnabled     CapabilityStatus = "enabled"
	CapabilityPending     CapabilityStatus = "pending"
	CapabilityUnavailable CapabilityStatus = "unavailable"
)

type StartGate struct {
	Allowed bool
	Reasons []string
}

// EvaluateStart is the single start gate used by the control panel and the
// host. Resource-spend and sensitive-page guards are reported as pending until
// captured, but they do not block start.
func EvaluateStart(caps Capabilities, draft Draft, catalog []TaskDescriptor) StartGate {
	var reasons []string
	if !caps.DeviceProfileValid {
		reasons = append(reasons, "设备分辨率未通过验收（需要 1600×900）")
	}
	if !caps.VisionReady {
		reasons = append(reasons, "图色识别未就绪")
	}
	for _, task := range catalog {
		setting, ok := draft.Tasks[task.ID]
		if !ok || !setting.Enabled {
			continue
		}
		if task.Available {
			continue
		}
		reason := task.UnavailableReason
		if reason == "" {
			reason = "任务不可用"
		}
		reasons = append(reasons, fmt.Sprintf("任务「%s」：%s", fallback(task.Name, task.ID), reason))
	}
	return StartGate{Allowed: len(reasons) == 0, Reasons: reasons}
}

// TaskAvailability projects a catalog entry from host facts. OCR-backed tasks
// stay closed until device OCR is accepted; missing vision is a hard unavailability.
func TaskAvailability(caps Capabilities, needsOCR bool) (available bool, reason string) {
	if !caps.VisionReady {
		return false, "图色识别未就绪"
	}
	if needsOCR && !caps.OCRReady {
		return false, "等待设备 OCR 验收"
	}
	return true, ""
}

func (c Capabilities) OCRStatus() CapabilityStatus {
	if c.OCRReady {
		return CapabilityEnabled
	}
	return CapabilityPending
}

func (c Capabilities) VisionStatus() CapabilityStatus {
	if c.VisionReady {
		return CapabilityEnabled
	}
	return CapabilityUnavailable
}

func (c Capabilities) ResourceGuardStatus() CapabilityStatus {
	if c.ResourceGuardReady {
		return CapabilityEnabled
	}
	return CapabilityPending
}

func (c Capabilities) SensitivePageGuardStatus() CapabilityStatus {
	if c.SensitivePageGuardReady {
		return CapabilityEnabled
	}
	return CapabilityPending
}

func (c Capabilities) DeviceProfileStatus() CapabilityStatus {
	if c.DeviceProfileValid {
		return CapabilityEnabled
	}
	return CapabilityUnavailable
}

func CapabilityStatusLabel(status CapabilityStatus) string {
	switch status {
	case CapabilityEnabled:
		return "已启用"
	case CapabilityPending:
		return "等待设备验收"
	default:
		return "不可用"
	}
}
