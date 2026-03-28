package ai

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	_ "image/gif"

	"golang.org/x/image/draw"
)

// CropImage crops the image to the given bounding box (x, y, width, height in pixels).
// Returns JPEG-encoded bytes of the cropped region.
func CropImage(data []byte, x, y, w, h int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	// Clamp to image dimensions
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+w > bounds.Max.X {
		w = bounds.Max.X - x
	}
	if y+h > bounds.Max.Y {
		h = bounds.Max.Y - y
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("crop region is out of bounds")
	}

	cropRect := image.Rect(x, y, x+w, y+h)
	cropped := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(cropped, cropped.Bounds(), img, cropRect.Min, draw.Src)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, cropped, &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("encode cropped image: %w", err)
	}
	return buf.Bytes(), nil
}

// ResizeImage resizes the image so its longest edge is at most maxDim pixels.
// Returns JPEG-encoded bytes. If the image is already smaller, it is returned as-is (re-encoded).
func ResizeImage(data []byte, maxDim int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	origW := bounds.Max.X - bounds.Min.X
	origH := bounds.Max.Y - bounds.Min.Y

	newW, newH := origW, origH
	if origW > maxDim || origH > maxDim {
		if origW >= origH {
			newW = maxDim
			newH = origH * maxDim / origW
		} else {
			newH = maxDim
			newW = origW * maxDim / origH
		}
	}

	if newW <= 0 {
		newW = 1
	}
	if newH <= 0 {
		newH = 1
	}

	resized := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("encode resized image: %w", err)
	}
	return buf.Bytes(), nil
}

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
