package main

import "fmt"

// Color represents an ANSI color index.
// -1 = default terminal color, 0-15 = standard ANSI 16 colors.
type Color int

const ColorDefault Color = -1

// ColorName returns a human-readable name for the color.
func ColorName(c Color) string {
	if c == ColorDefault {
		return "Default"
	}
	names := []string{
		"Black", "Red", "Green", "Yellow",
		"Blue", "Magenta", "Cyan", "White",
		"Br.Black", "Br.Red", "Br.Green", "Br.Yellow",
		"Br.Blue", "Br.Magenta", "Br.Cyan", "Br.White",
	}
	if int(c) < len(names) {
		return names[c]
	}
	if int(c) <= 255 {
		return fmt.Sprintf("c%d", c)
	}
	return "?"
}

// ColorCode returns a hex string for use with lipgloss.
// Supports all 256 ANSI colors.
func ColorCode(c Color) string {
	if c == ColorDefault || c < 0 {
		return ""
	}
	if int(c) > 255 {
		return ""
	}
	// Colors 0-15 use hardcoded values for accuracy
	if c < 16 {
		hexes := []string{
			"#000000", "#800000", "#008000", "#808000",
			"#000080", "#800080", "#008080", "#c0c0c0",
			"#808080", "#ff0000", "#00ff00", "#ffff00",
			"#0000ff", "#ff00ff", "#00ffff", "#ffffff",
		}
		return hexes[c]
	}
	// Colors 16-255 computed from ansi256Palette (defined in image.go)
	rgb := ansi256Palette[c]
	return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
}

// Cell is a single canvas cell.
type Cell struct {
	Char rune
	FG   Color
	BG   Color
}

// Canvas is a 2D grid of cells.
type Canvas struct {
	Width  int
	Height int
	Cells  [][]Cell
}

// NewCanvas creates a blank canvas.
func NewCanvas(w, h int) *Canvas {
	cells := make([][]Cell, h)
	for i := range cells {
		cells[i] = make([]Cell, w)
		for j := range cells[i] {
			cells[i][j] = Cell{Char: ' ', FG: ColorDefault, BG: ColorDefault}
		}
	}
	return &Canvas{Width: w, Height: h, Cells: cells}
}

func (c *Canvas) Set(x, y int, cell Cell) {
	if x >= 0 && x < c.Width && y >= 0 && y < c.Height {
		c.Cells[y][x] = cell
	}
}

func (c *Canvas) Get(x, y int) Cell {
	if x >= 0 && x < c.Width && y >= 0 && y < c.Height {
		return c.Cells[y][x]
	}
	return Cell{Char: ' ', FG: ColorDefault, BG: ColorDefault}
}

// Fill does a flood fill from (x,y) replacing cells matching target with replacement.
func (c *Canvas) Fill(x, y int, replacement Cell) {
	target := c.Get(x, y)
	if target == replacement {
		return
	}
	// BFS to avoid stack overflow on large canvases
	type point struct{ x, y int }
	queue := []point{{x, y}}
	visited := make([][]bool, c.Height)
	for i := range visited {
		visited[i] = make([]bool, c.Width)
	}
	visited[y][x] = true
	dirs := []point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		c.Cells[cur.y][cur.x] = replacement
		for _, d := range dirs {
			nx, ny := cur.x+d.x, cur.y+d.y
			if nx >= 0 && nx < c.Width && ny >= 0 && ny < c.Height &&
				!visited[ny][nx] && c.Cells[ny][nx] == target {
				visited[ny][nx] = true
				queue = append(queue, point{nx, ny})
			}
		}
	}
}

// Resize returns a new canvas with the given dimensions, preserving content.
func (c *Canvas) Resize(w, h int) *Canvas {
	n := NewCanvas(w, h)
	maxY := h
	if c.Height < maxY {
		maxY = c.Height
	}
	maxX := w
	if c.Width < maxX {
		maxX = c.Width
	}
	for y := 0; y < maxY; y++ {
		for x := 0; x < maxX; x++ {
			n.Cells[y][x] = c.Cells[y][x]
		}
	}
	return n
}
