package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ── JSON project format ──────────────────────────────────────────────────────

type savedCell struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Char string `json:"char"`
	FG   int    `json:"fg"`
	BG   int    `json:"bg"`
}

type savedCanvas struct {
	Version int         `json:"version"`
	Width   int         `json:"width"`
	Height  int         `json:"height"`
	Cells   []savedCell `json:"cells"`
}

func saveCanvas(c *Canvas, filename string) error {
	cells := make([]savedCell, 0) // explicit empty slice → JSON "[]", not "null"
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			cell := c.Cells[y][x]
			if cell.IsBlank() {
				continue
			}
			cells = append(cells, savedCell{
				X: x, Y: y,
				Char: string(cell.Char),
				FG:   int(cell.FG),
				BG:   int(cell.BG),
			})
		}
	}
	data, err := json.MarshalIndent(savedCanvas{
		Version: 1,
		Width:   c.Width,
		Height:  c.Height,
		Cells:   cells,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func loadCanvas(filename string) (*Canvas, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var saved savedCanvas // renamed from "sc" to avoid shadowing in the loop below
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, err
	}
	if saved.Width <= 0 || saved.Height <= 0 {
		return nil, fmt.Errorf("invalid canvas dimensions: %dx%d", saved.Width, saved.Height)
	}
	c := NewCanvas(saved.Width, saved.Height)
	for _, sc := range saved.Cells {
		runes := []rune(sc.Char)
		ch := rune(' ')
		if len(runes) > 0 {
			ch = runes[0]
		}
		c.Set(sc.X, sc.Y, Cell{Char: ch, FG: Color(sc.FG), BG: Color(sc.BG)})
	}
	return c, nil
}

// ── ANSI export ──────────────────────────────────────────────────────────────

// exportANSI writes the canvas as a file of raw ANSI escape sequences.
// Trailing blank cells are trimmed per row to keep the file small.
func exportANSI(c *Canvas, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	fmt.Fprint(w, "\033[0m") // reset before art so shell colors don't bleed in

	for y := 0; y < c.Height; y++ {
		lastX := lastNonBlank(c, y)
		if lastX < 0 {
			fmt.Fprintln(w)
			continue
		}
		prevFG, prevBG := ColorDefault, ColorDefault // after \033[0m\n, terminal is at default
		for x := 0; x <= lastX; x++ {
			cell := c.Cells[y][x]
			if cell.Char == 0 {
				cell.Char = ' '
			}
			var codes []string
			if cell.FG != prevFG {
				if cell.FG == ColorDefault {
					codes = append(codes, "39")
				} else {
					codes = append(codes, colorCode(cell.FG, true))
				}
				prevFG = cell.FG
			}
			if cell.BG != prevBG {
				if cell.BG == ColorDefault {
					codes = append(codes, "49")
				} else {
					codes = append(codes, colorCode(cell.BG, false))
				}
				prevBG = cell.BG
			}
			if len(codes) > 0 {
				fmt.Fprintf(w, "\033[%sm", strings.Join(codes, ";"))
			}
			fmt.Fprint(w, string(cell.Char))
		}
		fmt.Fprint(w, "\033[0m\n")
	}
	return w.Flush()
}

// lastNonBlank returns the x index of the rightmost visible cell in row y,
// or -1 if the entire row is blank.
func lastNonBlank(c *Canvas, y int) int {
	last := -1
	for x := 0; x < c.Width; x++ {
		if !c.Cells[y][x].IsBlank() {
			last = x
		}
	}
	return last
}

// colorCode returns the ANSI SGR parameter string for color c.
// isFG=true → foreground code, false → background code.
func colorCode(c Color, isFG bool) string {
	base, bright, ext := 30, 90, 38
	if !isFG {
		base, bright, ext = 40, 100, 48
	}
	switch {
	case c < 8:
		return fmt.Sprintf("%d", base+int(c))
	case c < 16:
		return fmt.Sprintf("%d", bright+int(c)-8)
	default:
		return fmt.Sprintf("%d;5;%d", ext, int(c))
	}
}

// ── Plain text import ────────────────────────────────────────────────────────

// importFromText loads a UTF-8 text file into a canvas with default colors.
func importFromText(filename string) (*Canvas, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return nil, fmt.Errorf("file is empty")
	}
	lines := strings.Split(content, "\n")

	w := 0
	for _, line := range lines {
		if n := len([]rune(line)); n > w {
			w = n
		}
	}
	if w == 0 {
		return nil, fmt.Errorf("file has no content")
	}
	c := NewCanvas(w, len(lines))
	for y, line := range lines {
		col := 0
		for _, r := range []rune(line) {
			if r == '\t' {
				col = (col/8 + 1) * 8 // advance to next 8-column tab stop
				continue
			}
			if r != 0 {
				c.Set(col, y, Cell{Char: r, FG: ColorDefault, BG: ColorDefault})
			}
			col++
		}
	}
	return c, nil
}

// ── Shell install ────────────────────────────────────────────────────────────

func installToShell(ansiFile string) error {
	absPath, err := filepath.Abs(ansiFile)
	if err != nil {
		return err
	}

	// Copy art to ~/.config/ansii/splash.ansi
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "ansii")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	splashPath := filepath.Join(configDir, "splash.ansi")
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", absPath, err)
	}
	if err := os.WriteFile(splashPath, data, 0644); err != nil {
		return err
	}

	// Inject or update the splash block in .bashrc / .zshrc
	const marker = "# ansii-splash"
	catCmd := fmt.Sprintf(`cat "%s"`, splashPath)

	added := false
	for _, rc := range shellRCs() {
		updated, changed, err := upsertShellBlock(rc, marker, catCmd)
		if err != nil || !changed {
			continue
		}
		if err := os.WriteFile(rc, []byte(updated), 0644); err != nil {
			continue
		}
		added = true
	}
	if !added {
		return fmt.Errorf("no .bashrc or .zshrc found in HOME")
	}
	return nil
}

// shellRCs returns the RC file paths to check for shell splash installation.
func shellRCs() []string {
	home := os.Getenv("HOME")
	return []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}
}

// upsertShellBlock inserts or replaces the 2-line splash block (marker + cat cmd)
// in the given RC file. Returns the updated content and whether a change occurred.
func upsertShellBlock(rc, marker, catCmd string) (string, bool, error) {
	data, err := os.ReadFile(rc)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	content := string(data)

	if !strings.Contains(content, marker) {
		// Append block, ensuring a trailing newline before it.
		sep := "\n"
		if strings.HasSuffix(content, "\n") {
			sep = ""
		}
		return content + sep + marker + "\n" + catCmd + "\n", true, nil
	}

	// Replace: scan line by line for the marker, then swap in the new cat cmd.
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		if lines[i] != marker {
			out = append(out, lines[i])
			continue
		}
		out = append(out, marker, catCmd)
		i++ // skip the marker line we just consumed
		// Skip the old cat line if it references splash.ansi
		if i < len(lines) && strings.Contains(lines[i], "splash.ansi") {
			i++
		}
		i-- // compensate for the outer loop increment
	}
	return strings.Join(out, "\n"), true, nil
}

