package ocr

import (
	"encoding/json"
	"strings"
)

// RegionText is a device OCR hit in screen space. The AutoGo ppocr adapter
// maps into this DTO so FormatScan/FindCenter stay desktop-testable.
type RegionText struct {
	Words      string
	X, Y, W, H int
}

// FormatScan turns region hits into the raw string ocr.Engine.Scan returns
// ("text"/"num" as concatenated labels, otherwise JSON words/location).
func FormatScan(items []RegionText, returnType string) string {
	switch returnType {
	case ReturnTypeText, ReturnTypeNum:
		parts := make([]string, 0, len(items))
		for _, item := range items {
			if item.Words != "" {
				parts = append(parts, item.Words)
			}
		}
		return strings.Join(parts, "")
	default:
		type locItem struct {
			Words    string  `json:"words"`
			Location [][]int `json:"location"`
		}
		out := make([]locItem, 0, len(items))
		for _, item := range items {
			x2, y2 := item.X+item.W, item.Y+item.H
			out = append(out, locItem{
				Words: item.Words,
				Location: [][]int{
					{item.X, item.Y}, {x2, item.Y}, {x2, y2}, {item.X, y2},
				},
			})
		}
		raw, err := json.Marshal(out)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

// FindCenter returns the center of the first region whose words contain text.
func FindCenter(items []RegionText, text string) (int, int, bool) {
	if text == "" {
		return 0, 0, false
	}
	for _, item := range items {
		if item.Words != "" && strings.Contains(item.Words, text) {
			return item.X + item.W/2, item.Y + item.H/2, true
		}
	}
	return 0, 0, false
}
