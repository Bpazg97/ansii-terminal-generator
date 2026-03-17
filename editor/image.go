package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"
)

// isJPEG returns true if the filename has a JPEG extension.
func isJPEG(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg")
}

// lumBG reports whether an RGB pixel should be treated as background based on luminance.
// JPEG images (no alpha channel) use a more aggressive threshold (200) to better remove
// typical light/white backgrounds. Other formats use a conservative threshold (230).
func lumBG(r, g, b uint8, jpeg bool) bool {
	lum := (299*int(r) + 587*int(g) + 114*int(b)) / 1000
	if jpeg {
		return lum > 200
	}
	return lum > 230
}

// ansi256Palette holds the RGB approximation of the 256 ANSI terminal colors.
// Index 0-15:   standard and bright colors (terminal-theme-dependent).
// Index 16-231: 6x6x6 color cube.
// Index 232-255: grayscale ramp (24 steps).
var ansi256Palette [256][3]uint8

func init() {
	standard := [][3]uint8{
		{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
		{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
		{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	}
	for i, c := range standard {
		ansi256Palette[i] = c
	}
	levels := []uint8{0, 95, 135, 175, 215, 255}
	for r := 0; r < 6; r++ {
		for g := 0; g < 6; g++ {
			for b := 0; b < 6; b++ {
				ansi256Palette[16+r*36+g*6+b] = [3]uint8{levels[r], levels[g], levels[b]}
			}
		}
	}
	for i := 0; i < 24; i++ {
		v := uint8(8 + i*10)
		ansi256Palette[232+i] = [3]uint8{v, v, v}
	}
}

// nearestANSI256 returns the ANSI 256 color index (16-255) closest to the given RGB value.
// Colors 0-15 are excluded: they are terminal-theme-dependent and not reproducible reliably.
// Distance uses the redmean weighted formula for better perceptual accuracy than plain Euclidean.
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

// sampleRegion returns the average RGBA (each component 0-255) of all pixels
// in [x1, x2) x [y1, y2), clamped to the image bounds.
// If the region collapses after clamping, the nearest single pixel is returned.
func sampleRegion(img image.Image, x1, y1, x2, y2 int) (r, g, b, a uint8) {
	bnd := img.Bounds()
	if x1 < bnd.Min.X {
		x1 = bnd.Min.X
	}
	if y1 < bnd.Min.Y {
		y1 = bnd.Min.Y
	}
	if x2 > bnd.Max.X {
		x2 = bnd.Max.X
	}
	if y2 > bnd.Max.Y {
		y2 = bnd.Max.Y
	}
	// Ensure at least a 1x1 region.
	if x2 <= x1 {
		x2 = x1 + 1
	}
	if y2 <= y1 {
		y2 = y1 + 1
	}
	// Re-clamp after expansion in case we stepped past the boundary.
	if x2 > bnd.Max.X {
		x2 = bnd.Max.X
	}
	if y2 > bnd.Max.Y {
		y2 = bnd.Max.Y
	}
	if x2 <= x1 || y2 <= y1 {
		cr, cg, cb, ca := img.At(bnd.Min.X, bnd.Min.Y).RGBA()
		return uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), uint8(ca >> 8)
	}

	var rs, gs, bs, as, n int64
	for py := y1; py < y2; py++ {
		for px := x1; px < x2; px++ {
			cr, cg, cb, ca := img.At(px, py).RGBA()
			rs += int64(cr >> 8)
			gs += int64(cg >> 8)
			bs += int64(cb >> 8)
			as += int64(ca >> 8)
			n++
		}
	}
	if n == 0 {
		return 0, 0, 0, 0
	}
	return uint8(rs / n), uint8(gs / n), uint8(bs / n), uint8(as / n)
}

// importFromImage converts an image to a canvas using the half-block technique.
//
// Each canvas cell encodes two pixel rows:
//   - Upper half: foreground color, rendered with the upper-half block character (U+2580 ▀).
//   - Lower half: background color.
//
// Source regions are box-averaged over the entire mapped pixel area to eliminate
// single-pixel aliasing ("dirty" appearance on downscaled images).
// Transparent pixels (PNG alpha < 50%) become empty cells.
//
// The target terminal character aspect ratio is assumed to be 2:1 (height:width).
// This means scaleY = scaleX preserves square image pixels without vertical distortion.
func importFromImage(filename string, targetW int) (*Canvas, error) {
	jpeg := isJPEG(filename)
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

	if targetW <= 0 {
		targetW = imgW / 2
		if targetW < 1 {
			targetW = 1
		}
	}

	// scaleX: source pixels per canvas column.
	// scaleY: source pixels per half-row; each canvas row spans two half-rows.
	// For a 2:1 terminal character (height:width), scaleY = scaleX yields square pixels.
	scaleX := float64(imgW) / float64(targetW)
	scaleY := scaleX

	canvasH := int(math.Ceil(float64(imgH) / (scaleY * 2)))
	if canvasH < 1 {
		canvasH = 1
	}

	c := NewCanvas(targetW, canvasH)

	const alphaThreshold = 128

	for cy := 0; cy < canvasH; cy++ {
		for cx := 0; cx < targetW; cx++ {
			x1 := bounds.Min.X + int(float64(cx)*scaleX)
			x2 := bounds.Min.X + int(float64(cx+1)*scaleX)
			y1U := bounds.Min.Y + int(float64(cy*2)*scaleY)
			y2U := bounds.Min.Y + int(float64(cy*2+1)*scaleY)
			y1L := y2U
			y2L := bounds.Min.Y + int(float64(cy*2+2)*scaleY)

			rU, gU, bU, aU := sampleRegion(img, x1, y1U, x2, y2U)
			rL, gL, bL, aL := sampleRegion(img, x1, y1L, x2, y2L)

			upperOpaque := aU >= alphaThreshold && !lumBG(rU, gU, bU, jpeg)
			lowerOpaque := aL >= alphaThreshold && !lumBG(rL, gL, bL, jpeg)

			if !upperOpaque && !lowerOpaque {
				continue
			}

			var fgColor, bgColor Color = ColorDefault, ColorDefault
			var ch rune

			switch {
			case !upperOpaque:
				// Only lower half visible: lower-half block, foreground = lower color.
				ch = '▄'
				fgColor = nearestANSI256(rL, gL, bL)
			case !lowerOpaque:
				// Only upper half visible: upper-half block, foreground = upper color.
				ch = '▀'
				fgColor = nearestANSI256(rU, gU, bU)
			default:
				ch = '▀'
				fgColor = nearestANSI256(rU, gU, bU)
				bgColor = nearestANSI256(rL, gL, bL)
			}

			c.Set(cx, cy, Cell{Char: ch, FG: fgColor, BG: bgColor})
		}
	}

	return c, nil
}

// asciiRamp maps luminance (0=dark … len-1=bright) to an ASCII character.
// Ordered darkest → lightest so that dark areas of the image become dense chars.
var asciiRamp = []rune(`$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\|()1{}[]?-_+~<>i!lI;:,"^'. `)

// importFromASCII converts an image to ASCII art entirely in Go — no external
// tools required. Each canvas cell receives an ASCII character chosen by
// luminance and the nearest ANSI-256 color from the source pixel.
//
// Terminal characters are approximately 2:1 (height:width), so the vertical
// scale is doubled relative to the horizontal scale to preserve the image
// aspect ratio.
func importFromASCII(filename string, targetW int) (*Canvas, error) {
	jpeg := isJPEG(filename)
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

	if targetW <= 0 {
		targetW = 80
	}

	// scaleX: source pixels per canvas column.
	// scaleY = 2*scaleX to account for the 2:1 terminal character aspect ratio,
	// so that the output canvas has the correct proportions.
	scaleX := float64(imgW) / float64(targetW)
	scaleY := scaleX * 2.0

	canvasH := int(math.Ceil(float64(imgH) / scaleY))
	if canvasH < 1 {
		canvasH = 1
	}

	c := NewCanvas(targetW, canvasH)
	rampLen := len(asciiRamp)

	const alphaThreshold = uint8(128)

	for cy := 0; cy < canvasH; cy++ {
		for cx := 0; cx < targetW; cx++ {
			x1 := bounds.Min.X + int(float64(cx)*scaleX)
			x2 := bounds.Min.X + int(float64(cx+1)*scaleX)
			y1 := bounds.Min.Y + int(float64(cy)*scaleY)
			y2 := bounds.Min.Y + int(float64(cy+1)*scaleY)

			r, g, b, a := sampleRegion(img, x1, y1, x2, y2)
			if a < alphaThreshold {
				continue
			}

			// Perceptual luminance (ITU-R BT.601)
			lum := (299*int(r) + 587*int(g) + 114*int(b)) / 1000

			// Background removal: skip bright/near-white pixels.
			// JPEG (no alpha) uses a more aggressive threshold (lum > 200);
			// other formats use lum > 230 (only removes near-white).
			if lumBG(r, g, b, jpeg) {
				continue
			}

			idx := lum * (rampLen - 1) / 255
			ch := asciiRamp[idx]
			if ch == ' ' {
				continue
			}

			fgColor := nearestANSI256(r, g, b)
			c.Set(cx, cy, Cell{Char: ch, FG: fgColor, BG: ColorDefault})
		}
	}

	return c, nil
}

