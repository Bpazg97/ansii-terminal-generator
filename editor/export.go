package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// --- JSON save/load ---

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
	var cells []savedCell
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			cell := c.Cells[y][x]
			if cell.Char == 0 || (cell.Char == ' ' && cell.FG == ColorDefault && cell.BG == ColorDefault) {
				continue
			}
			cells = append(cells, savedCell{
				X:    x,
				Y:    y,
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
	var sc savedCanvas
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}
	c := NewCanvas(sc.Width, sc.Height)
	for _, sc := range sc.Cells {
		r := []rune(sc.Char)
		ch := ' '
		if len(r) > 0 {
			ch = r[0]
		}
		c.Set(sc.X, sc.Y, Cell{
			Char: ch,
			FG:   Color(sc.FG),
			BG:   Color(sc.BG),
		})
	}
	return c, nil
}

// --- ANSI export ---

// exportANSI writes the canvas as ANSI escape codes.
func exportANSI(c *Canvas, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for y := 0; y < c.Height; y++ {
		prevFG := Color(-2) // sentinel
		prevBG := Color(-2)
		rowEmpty := true
		for x := 0; x < c.Width; x++ {
			if c.Cells[y][x].Char != ' ' || c.Cells[y][x].BG != ColorDefault {
				rowEmpty = false
				break
			}
		}
		if rowEmpty {
			fmt.Fprintln(w)
			continue
		}

		for x := 0; x < c.Width; x++ {
			cell := c.Cells[y][x]
			if cell.Char == 0 {
				cell.Char = ' '
			}

			// Build ANSI code if colors changed
			var codes []string
			if cell.FG != prevFG {
				if cell.FG == ColorDefault {
					codes = append(codes, "39")
				} else {
					codes = append(codes, fgCode(cell.FG))
				}
				prevFG = cell.FG
			}
			if cell.BG != prevBG {
				if cell.BG == ColorDefault {
					codes = append(codes, "49")
				} else {
					codes = append(codes, bgCode(cell.BG))
				}
				prevBG = cell.BG
			}
			if len(codes) > 0 {
				fmt.Fprintf(w, "\033[%sm", strings.Join(codes, ";"))
			}
			fmt.Fprintf(w, "%s", string(cell.Char))
		}
		fmt.Fprintf(w, "\033[0m\n")
	}

	return w.Flush()
}

func fgCode(c Color) string {
	if c < 8 {
		return fmt.Sprintf("%d", 30+int(c))
	}
	if c < 16 {
		return fmt.Sprintf("%d", 90+int(c)-8)
	}
	// 256-color mode
	return fmt.Sprintf("38;5;%d", int(c))
}

func bgCode(c Color) string {
	if c < 8 {
		return fmt.Sprintf("%d", 40+int(c))
	}
	if c < 16 {
		return fmt.Sprintf("%d", 100+int(c)-8)
	}
	// 256-color mode
	return fmt.Sprintf("48;5;%d", int(c))
}

// --- Import from plain text ---

// importFromText reads a plain UTF-8 text file (ASCII art, braille art, etc.)
// and converts it to a canvas with default colors.
func importFromText(filename string) (*Canvas, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// Normalize line endings and trim trailing blank lines
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return nil, fmt.Errorf("file is empty")
	}
	lines := strings.Split(content, "\n")

	h := len(lines)
	w := 0
	for _, line := range lines {
		if n := len([]rune(line)); n > w {
			w = n
		}
	}
	if w == 0 {
		return nil, fmt.Errorf("file has no content")
	}

	c := NewCanvas(w, h)
	for y, line := range lines {
		for x, r := range []rune(line) {
			if r != 0 && r != '\t' {
				c.Set(x, y, Cell{Char: r, FG: ColorDefault, BG: ColorDefault})
			}
		}
	}
	return c, nil
}

// --- Shell install ---

func installToShell(ansiFile string) error {
	// Make path absolute
	if !strings.HasPrefix(ansiFile, "/") {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		ansiFile = wd + "/" + ansiFile
	}

	// Copy to ~/.config/ansii/splash.ansi
	configDir := os.Getenv("HOME") + "/.config/ansii"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	splashPath := configDir + "/splash.ansi"
	data, err := os.ReadFile(ansiFile)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", ansiFile, err)
	}
	if err := os.WriteFile(splashPath, data, 0644); err != nil {
		return err
	}

	// Add to shell RC if not already present
	catCmd := fmt.Sprintf("cat \"%s\"", splashPath)
	marker := "# ansii-splash"
	line := marker + "\n" + catCmd

	shells := []string{
		os.Getenv("HOME") + "/.bashrc",
		os.Getenv("HOME") + "/.zshrc",
	}

	added := false
	for _, rc := range shells {
		if _, err := os.Stat(rc); os.IsNotExist(err) {
			continue
		}
		content, err := os.ReadFile(rc)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), marker) {
			// Replace existing block
			lines := strings.Split(string(content), "\n")
			var out []string
			skip := false
			for _, l := range lines {
				if l == marker {
					out = append(out, line)
					skip = true
					continue
				}
				if skip && strings.Contains(l, "cat") && strings.Contains(l, "splash.ansi") {
					skip = false
					continue
				}
				skip = false
				out = append(out, l)
			}
			os.WriteFile(rc, []byte(strings.Join(out, "\n")), 0644)
		} else {
			f, err := os.OpenFile(rc, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				continue
			}
			fmt.Fprintf(f, "\n%s\n", line)
			f.Close()
		}
		added = true
	}

	if !added {
		return fmt.Errorf("no .bashrc or .zshrc found in HOME")
	}
	return nil
}
