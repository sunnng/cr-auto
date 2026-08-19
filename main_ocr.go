package main

import (
	"image"

	"app/internal/lib/ocr"

	"github.com/Dasongzi1366/AutoGo/ppocr"
)

type deviceOCR struct {
	engine *ppocr.Ppocr
}

func (d *deviceOCR) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
	if d == nil || d.engine == nil || rect.Empty() {
		return "", nil
	}
	hits := d.engine.Ocr(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, "", displayID)
	regions := make([]ocr.RegionText, 0, len(hits))
	for _, hit := range hits {
		regions = append(regions, ocr.RegionText{
			Words: hit.Label, X: hit.X, Y: hit.Y, W: hit.Width, H: hit.Height,
		})
	}
	return ocr.FormatScan(regions, returnType), nil
}

func (d *deviceOCR) FindTapPoint(text string, rect image.Rectangle) (int, int, bool) {
	if d == nil || d.engine == nil || rect.Empty() {
		return 0, 0, false
	}
	hits := d.engine.Ocr(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, "", displayID)
	regions := make([]ocr.RegionText, 0, len(hits))
	for _, hit := range hits {
		regions = append(regions, ocr.RegionText{
			Words: hit.Label, X: hit.X, Y: hit.Y, W: hit.Width, H: hit.Height,
		})
	}
	return ocr.FindCenter(regions, text)
}

func injectDeviceOCR() {
	engine := ppocr.New("v5")
	if engine == nil {
		return
	}
	ocr.SetEngine(&deviceOCR{engine: engine})
}
