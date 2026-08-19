package ui

// FitRunes shortens value so measure(result) <= maxWidth, appending an ellipsis
// when any runes are dropped. Empty maxWidth or a nil measure returns value.
func FitRunes(value string, maxWidth float32, measure func(string) float32) string {
	if measure == nil || maxWidth <= 0 {
		return value
	}
	if measure(value) <= maxWidth {
		return value
	}
	const ellipsis = "…"
	if measure(ellipsis) > maxWidth {
		return ""
	}
	runes := []rune(value)
	lo, hi := 0, len(runes)
	best := ellipsis
	for lo < hi {
		mid := (lo + hi + 1) / 2
		candidate := string(runes[:mid]) + ellipsis
		if measure(candidate) <= maxWidth {
			lo = mid
			best = candidate
		} else {
			hi = mid - 1
		}
	}
	return best
}
