package main

import "fmt"

// Color is an ANSI color index.
// -1 (ColorDefault) = terminal default, 0–7 = standard, 8–15 = bright, 16–255 = 256-color.
type Color int

const ColorDefault Color = -1

// colorNames is package-level to avoid allocation on every call.
var colorNames = [16]string{
	"Black", "Red", "Green", "Yellow",
	"Blue", "Magenta", "Cyan", "White",
	"Br.Black", "Br.Red", "Br.Green", "Br.Yellow",
	"Br.Blue", "Br.Magenta", "Br.Cyan", "Br.White",
}

// ColorName returns a human-readable label for c.
func ColorName(c Color) string {
	switch {
	case c == ColorDefault:
		return "Default"
	case c < 16:
		return colorNames[c]
	case c <= 255:
		return fmt.Sprintf("c%d", int(c))
	default:
		return "?"
	}
}

// ── Cell & Canvas ────────────────────────────────────────────────────────────

// Cell is a single canvas cell: a Unicode character with foreground and background colors.
type Cell struct {
	Char rune
	FG   Color
	BG   Color
}

// BlankCell is the canonical empty cell: space with terminal-default colors.
var BlankCell = Cell{Char: ' ', FG: ColorDefault, BG: ColorDefault}

// IsBlank reports whether the cell has no visible content or color.
// Both the canonical blank and the Go zero value are treated as empty.
func (c Cell) IsBlank() bool {
	charEmpty := c.Char == 0 || c.Char == ' '
	return charEmpty && c.FG == ColorDefault && c.BG == ColorDefault ||
		c == (Cell{}) // Go zero value: {0, 0(black), 0(black)} — treat as unset
}

// Canvas is a 2D grid of cells.
type Canvas struct {
	Width  int
	Height int
	Cells  [][]Cell
}

// NewCanvas allocates a blank canvas. Every cell is a space with default colors.
func NewCanvas(w, h int) *Canvas {
	cells := make([][]Cell, h)
	for y := range cells {
		cells[y] = make([]Cell, w)
		for x := range cells[y] {
			cells[y][x] = BlankCell
		}
	}
	return &Canvas{Width: w, Height: h, Cells: cells}
}

// Set writes cell to (x, y), silently ignoring out-of-bounds coordinates.
func (c *Canvas) Set(x, y int, cell Cell) {
	if x >= 0 && x < c.Width && y >= 0 && y < c.Height {
		c.Cells[y][x] = cell
	}
}

// Get returns the cell at (x, y), or a blank default cell if out of bounds.
func (c *Canvas) Get(x, y int) Cell {
	if x >= 0 && x < c.Width && y >= 0 && y < c.Height {
		return c.Cells[y][x]
	}
	return BlankCell
}

// Fill flood-fills from (x, y), replacing all connected cells that equal the
// starting cell with replacement. Uses iterative BFS to avoid stack overflow.
func (c *Canvas) Fill(x, y int, replacement Cell) {
	target := c.Get(x, y)
	if target == replacement {
		return
	}
	type pt struct{ x, y int }
	visited := make([][]bool, c.Height)
	for i := range visited {
		visited[i] = make([]bool, c.Width)
	}
	visited[y][x] = true

	queue := []pt{{x, y}}
	dirs := [4]pt{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for qi := 0; qi < len(queue); qi++ { // index-based: avoids O(n) front-dequeue
		cur := queue[qi]
		c.Cells[cur.y][cur.x] = replacement
		for _, d := range dirs {
			nx, ny := cur.x+d.x, cur.y+d.y
			if nx >= 0 && nx < c.Width && ny >= 0 && ny < c.Height &&
				!visited[ny][nx] && c.Cells[ny][nx] == target {
				visited[ny][nx] = true
				queue = append(queue, pt{nx, ny})
			}
		}
	}
}

