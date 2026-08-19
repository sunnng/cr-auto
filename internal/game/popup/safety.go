package popup

import "app/internal/vision"

// ResourceSpendDef 商店/水晶购买确认等资源消费画面。真机验收前保持空，
// 填入多点比色串后 SafetyFeaturesReady 才会为真。
func ResourceSpendDef() []vision.Feature {
	return nil
}

// SensitivePageDef 账号、支付、客服等敏感页。空 = 未验收。
func SensitivePageDef() []vision.Feature {
	return nil
}

// FeaturesReady 每一组都至少有一条 Points 非空且 Sim>0 才算就绪。
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

// SafetyFeaturesReady 资源消费与敏感页面两组特征都已采集。
func SafetyFeaturesReady() bool {
	return FeaturesReady(ResourceSpendDef(), SensitivePageDef())
}
