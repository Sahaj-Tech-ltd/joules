package ai

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/image/draw"
)

// ExtractTextFromImage runs Tesseract OCR on the provided image bytes.
// Returns the extracted text, or an error if Tesseract is not available.
// The caller should check IsTesseractAvailable() before calling this.
func ExtractTextFromImage(data []byte) (string, error) {
	// Write image to a temp file
	tmp, err := os.CreateTemp("", "joule-ocr-*.jpg")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	// If PNG, convert to JPEG for better Tesseract compatibility
	processed := data
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 {
		processed, err = pngToJPEG(data)
		if err != nil {
			processed = data // fall back to original
		}
	}

	if _, err := tmp.Write(processed); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	tmp.Close()

	// Run tesseract: output to stdout, page-segmentation-mode 3 (fully automatic)
	cmd := exec.Command("tesseract", tmp.Name(), "stdout", "--psm", "3", "-l", "eng")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract failed: %w (stderr: %s)", err, stderr.String())
	}

	text := strings.TrimSpace(out.String())
	return text, nil
}

// IsTesseractAvailable checks whether the tesseract binary is on PATH.
func IsTesseractAvailable() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

func pngToJPEG(data []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Convert to RGBA if needed
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
