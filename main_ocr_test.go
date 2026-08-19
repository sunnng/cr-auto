package main

import (
	"image"
	"testing"

	"app/internal/lib/ocr"
)

func TestDeviceOCRScanWithoutEngineReturnsEmpty(t *testing.T) {
	d := &deviceOCR{}
	raw, err := d.Scan(image.Rect(0, 0, 10, 10), ocr.MultiMode, ocr.ReturnTypeText)
	if err != nil || raw != "" {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
	if _, _, ok := d.FindTapPoint("配置", image.Rect(0, 0, 10, 10)); ok {
		t.Fatal("nil engine must not report a tap point")
	}
}
