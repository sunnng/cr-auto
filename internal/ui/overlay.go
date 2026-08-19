package ui

// OverlayScale maps detection-image pixels onto the preview dest rectangle.
// Empty image bounds return 1 to avoid dividing by zero.
func OverlayScale(imageW, imageH int, destW, destH float32) (scaleX, scaleY float32) {
	if imageW <= 0 || imageH <= 0 {
		return 1, 1
	}
	return destW / float32(imageW), destH / float32(imageH)
}

// ShouldRebuildTexture decides whether the detection preview GPU texture must
// be deleted or recreated. A true result means the caller should tear down
// the current texture (if any) and, when an image is present, upload a new one.
func ShouldRebuildTexture(currentRev, incomingRev uint64, hasTexture, hasImage bool) bool {
	if !hasImage {
		return hasTexture
	}
	if !hasTexture {
		return true
	}
	return incomingRev != currentRev
}
