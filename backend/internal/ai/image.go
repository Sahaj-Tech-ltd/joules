package ai

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
)

// PrepareForOCR converts an image to grayscale and applies mild contrast
// enhancement to improve Tesseract accuracy, especially on nutrition labels.
func PrepareForOCR(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, gray, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("encode grayscale image: %w", err)
	}
	return buf.Bytes(), nil
}
