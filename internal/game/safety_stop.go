package game

import (
	"app/internal/core"
	"app/internal/game/popup"
	"app/internal/lib/touch"
	"app/internal/vision"
)

var safetyStop func(string)

// SetSafetyStop 注入安全守卫命中后的停机回调（宿主接到 Runtime cancel）。
func SetSafetyStop(fn func(reason string)) { safetyStop = fn }

func requestSafetyStop(reason string) {
	if safetyStop != nil {
		safetyStop(reason)
	}
}

// RegisterSafetyGuards 注册资源消费保护与敏感页面停机。空特征组跳过，避免空守卫冒充已启用。
func RegisterSafetyGuards(g *core.Guard, resource, sensitive []vision.Feature) {
	if len(resource) > 0 {
		g.Register("资源消费保护", resource, func() {
			touch.PressBack(0)
			requestSafetyStop("命中资源消费保护，已停止")
		}, 20)
	}
	if len(sensitive) > 0 {
		g.Register("敏感页面停机", sensitive, func() {
			requestSafetyStop("进入敏感页面，已停止")
		}, 19)
	}
}

// SafetyGuardsReady 两组安全特征都已采集。未采集时即使用 RegisterAll 也不得宣称已启用。
func SafetyGuardsReady() bool { return popup.SafetyFeaturesReady() }
