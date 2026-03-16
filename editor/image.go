package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// ansi256Palette is the RGB approximation of the 256 ANSI colors.
// Indices 0-15: standard + bright.
// Indices 16-231: 6×6×6 color cube.
// Indices 232-255: grayscale ramp.
var ansi256Palette [256][3]uint8

func init() {
	// 0-15: standard ANSI (approximate)
	standard := [][3]uint8{
		{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
		{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
		{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	}
	for i, c := range standard {
		ansi256Palette[i] = c
	}
	// 16-231: 6×6×6 cube
	levels := []uint8{0, 95, 135, 175, 215, 255}
	for r := 0; r < 6; r++ {
		for g := 0; g < 6; g++ {
			for b := 0; b < 6; b++ {
				idx := 16 + r*36 + g*6 + b
				ansi256Palette[idx] = [3]uint8{levels[r], levels[g], levels[b]}
			}
		}
	}
	// 232-255: grayscale
	for i := 0; i < 24; i++ {
		v := uint8(8 + i*10)
		ansi256Palette[232+i] = [3]uint8{v, v, v}
	}
}

// nearestANSI256 returns the index (16-255) of the closest ANSI 256 color to rgb.
// Colors 0-15 are skipped because they are terminal-theme-dependent and unreliable.
// Uses the redmean weighted distance for perceptually better results.
func nearestANSI256(r, g, b uint8) Color {
	best := 16
	bestDist := math.MaxFloat64
	for i := 16; i < 256; i++ {
		c := ansi256Palette[i]
		rMean := (float64(r) + float64(c[0])) / 2
		dr := float64(r) - float64(c[0])
		dg := float64(g) - float64(c[1])
		db := float64(b) - float64(c[2])
		d := (2+rMean/256)*dr*dr + 4*dg*dg + (2+(255-rMean)/256)*db*db
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return Color(best)
}

// ColorCode256 returns the hex representation of an ANSI 256 color.
func ColorCode256(c Color) string {
	if c == ColorDefault {
		return ""
	}
	if int(c) < 0 || int(c) > 255 {
		return ""
	}
	rgb := ansi256Palette[c]
	return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
}

// isTransparent returns true if the pixel at (x,y) is transparent (alpha < threshold).
func isTransparent(img image.Image, x, y int, threshold uint32) bool {
	_, _, _, a := img.At(x, y).RGBA()
	return a < threshold
}

// importFromImage converts an image file to a canvas using the half-block technique.
// Each terminal cell represents 2 vertical pixels using '▀' (upper half block):
//   - Foreground color = upper pixel
//   - Background color = lower pixel
//
// Transparent pixels (PNG alpha) are treated as empty cells.
// targetW is the desired canvas width in columns; 0 = auto (image width / 2).
func importFromImage(filename string, targetW int) (*Canvas, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("cannot decode image: %w", err)
	}

	bounds := img.Bounds()
	imgW := bounds.Max.X - bounds.Min.X
	imgH := bounds.Max.Y - bounds.Min.Y

	// Compute scale so the canvas fits targetW columns.
	// Each column = 2 source pixels wide (characters are ~2x taller than wide).
	// Each row    = 2 source pixels tall (half-block uses 2px per row).
	var scaleX, scaleY float64
	if targetW <= 0 {
		// Default: use image width (1 char per 2 pixels)
		targetW = imgW / 2
		if targetW < 1 {
			targetW = 1
		}
	}
	scaleX = float64(imgW) / float64(targetW)
	// Terminal chars are ~2:1 (h:w). Each cell covers 2 source rows via ▀/▄.
	// Correct vertical scale = scaleX (not scaleX*0.5 which caused 2× stretch).
	scaleY = scaleX

	canvasW := targetW
	canvasH := int(math.Ceil(float64(imgH) / scaleY / 2))
	if canvasH < 1 {
		canvasH = 1
	}

	c := NewCanvas(canvasW, canvasH)
	alphaThreshold := uint32(0x8000) // 50% opacity

	for cy := 0; cy < canvasH; cy++ {
		for cx := 0; cx < canvasW; cx++ {
			// Source pixel coordinates for upper and lower halves
			srcX := int(float64(cx)*scaleX) + bounds.Min.X
			srcYUpper := int(float64(cy*2)*scaleY) + bounds.Min.Y
			srcYLower := int(float64(cy*2+1)*scaleY) + bounds.Min.Y

			clamp := func(v, max int) int {
				if v >= max {
					return max - 1
				}
				return v
			}
			srcX = clamp(srcX, bounds.Max.X)
			srcYUpper = clamp(srcYUpper, bounds.Max.Y)
			srcYLower = clamp(srcYLower, bounds.Max.Y)

			upperTransparent := isTransparent(img, srcX, srcYUpper, alphaThreshold)
			lowerTransparent := isTransparent(img, srcX, srcYLower, alphaThreshold)

			if upperTransparent && lowerTransparent {
				// Both transparent — empty cell
				continue
			}

			var fgColor, bgColor Color
			fgColor = ColorDefault
			bgColor = ColorDefault

			if !upperTransparent {
				r, g, b, _ := img.At(srcX, srcYUpper).RGBA()
				fgColor = nearestANSI256(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			}
			if !lowerTransparent {
				r, g, b, _ := img.At(srcX, srcYLower).RGBA()
				bgColor = nearestANSI256(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			}

			var ch rune
			switch {
			case upperTransparent:
				// Only lower half has color — use '▄' (lower half block)
				ch = '▄'
				fgColor = bgColor
				bgColor = ColorDefault
			case lowerTransparent:
				// Only upper half — use '▀', bg = default
				ch = '▀'
				bgColor = ColorDefault
			default:
				ch = '▀'
			}

			c.Set(cx, cy, Cell{Char: ch, FG: fgColor, BG: bgColor})
		}
	}

	return c, nil
}
