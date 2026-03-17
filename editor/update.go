package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.installConfirm {
			return m.handleInstallConfirmKey(msg)
		}
		if m.saveMode {
			return m.handleSaveKey(msg)
		}
		if m.importMode {
			return m.handleImportKey(msg)
		}
		if m.colorMode {
			return m.handleColorInputKey(msg)
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		return m.handleCanvasKey(msg)
	}
	return m, nil
}

// ── Install confirm ───────────────────────────────────────────────────────────

func (m Model) handleInstallConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter", "y", "Y":
		m.installConfirm = false
		outFile := changeExt(m.filename, ".ansi")
		if outFile == "" {
			outFile = "art.ansi"
		}
		if err := exportANSI(m.canvas, outFile); err != nil {
			m.statusMsg = fmt.Sprintf("Error exporting: %v", err)
		} else if err := installToShell(outFile); err != nil {
			m.statusMsg = fmt.Sprintf("Error installing: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("Installed! Restart terminal. (file: %s)", outFile)
		}
	case "esc", "escape", "n", "N":
		m.installConfirm = false
		m.statusMsg = "Install cancelled"
	}
	return m, nil
}

// ── Save prompt ───────────────────────────────────────────────────────────────

func (m Model) handleSaveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "escape":
		m.saveMode = false
		m.saveInput = ""
		m.saveErrMsg = ""
		m.completions = nil
		m.compIdx = 0
	case "enter":
		if m.saveInput == "" {
			break
		}
		path := m.saveInput
		isExport := strings.EqualFold(filepath.Ext(path), ".ansi")
		var err error
		if isExport {
			err = exportANSI(m.canvas, path)
		} else {
			err = saveCanvas(m.canvas, path)
		}
		if err != nil {
			m.saveErrMsg = err.Error()
		} else {
			if !isExport {
				m.filename = path
				m.modified = false
			}
			m.statusMsg = fmt.Sprintf("Saved → %s", path)
			m.saveMode = false
			m.saveInput = ""
			m.saveErrMsg = ""
			m.completions = nil
			m.compIdx = 0
		}
	case "tab":
		m.completions, m.compIdx = cycleCompletion(m.completions, m.compIdx, m.saveInput, +1)
		if len(m.completions) > 0 {
			m.saveInput = m.completions[m.compIdx]
			m.saveErrMsg = ""
		}
	case "shift+tab":
		m.completions, m.compIdx = cycleCompletion(m.completions, m.compIdx, m.saveInput, -1)
		if len(m.completions) > 0 {
			m.saveInput = m.completions[m.compIdx]
			m.saveErrMsg = ""
		}
	case "backspace":
		runes := []rune(m.saveInput)
		if len(runes) > 0 {
			m.saveInput = string(runes[:len(runes)-1])
			m.saveErrMsg = ""
			m.completions = nil
			m.compIdx = 0
		}
	default:
		r := []rune(msg.String())
		if len(r) == 1 && r[0] >= 32 {
			m.saveInput += string(r[0])
			m.saveErrMsg = ""
			m.completions = nil
			m.compIdx = 0
		}
	}
	return m, nil
}

// ── Import prompt ─────────────────────────────────────────────────────────────

func (m Model) handleImportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "escape":
		m.importMode = false
		m.importInput = ""
		m.importErrMsg = ""
		m.importASCII = false
		m.completions = nil
		m.compIdx = 0
	case "enter":
		if m.importInput == "" {
			break
		}
		path := resolvePath(m.importInput)
		var (
			c   *Canvas
			err error
		)
		switch {
		case m.importASCII:
			c, err = importFromASCII(path, 80)
		case m.importIsImg:
			c, err = importFromImage(path, 80)
		default:
			c, err = importFromText(path)
		}
		if err != nil {
			m.importErrMsg = err.Error()
		} else {
			m.canvas = c
			m.cursorX, m.cursorY = 0, 0
			m.viewX, m.viewY = 0, 0
			m.modified = true
			m.statusMsg = fmt.Sprintf("Imported '%s' (%dx%d) — [s] to save", path, c.Width, c.Height)
			m.importMode = false
			m.importInput = ""
			m.importErrMsg = ""
			m.importASCII = false
			m.completions = nil
			m.compIdx = 0
		}
	case "tab":
		m.completions, m.compIdx = cycleCompletion(m.completions, m.compIdx, m.importInput, +1)
		if len(m.completions) > 0 {
			m.importInput = m.completions[m.compIdx]
			m.importErrMsg = ""
		}
	case "shift+tab":
		m.completions, m.compIdx = cycleCompletion(m.completions, m.compIdx, m.importInput, -1)
		if len(m.completions) > 0 {
			m.importInput = m.completions[m.compIdx]
			m.importErrMsg = ""
		}
	case "backspace":
		runes := []rune(m.importInput)
		if len(runes) > 0 {
			m.importInput = string(runes[:len(runes)-1])
			m.importErrMsg = ""
			m.completions = nil
			m.compIdx = 0
		}
	default:
		r := []rune(msg.String())
		if len(r) == 1 && r[0] >= 32 {
			m.importInput += string(r[0])
			m.importErrMsg = ""
			m.completions = nil
			m.compIdx = 0
		}
	}
	return m, nil
}

// ── Color input prompt ────────────────────────────────────────────────────────

// handleColorInputKey handles the inline color-pick prompt.
// Accepts: digits (0-255), "d" for default, Enter to confirm, Esc to cancel.
func (m Model) handleColorInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "escape":
		m.colorMode = false
		m.colorInput = ""
		m.colorErrMsg = ""
	case "enter":
		raw := strings.TrimSpace(m.colorInput)
		var color Color
		switch strings.ToLower(raw) {
		case "", "d", "default", "-1":
			color = ColorDefault
		default:
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 || n > 255 {
				m.colorErrMsg = fmt.Sprintf("'%s' is not valid (0-255 or d)", raw)
				return m, nil
			}
			color = Color(n)
		}
		if m.colorModeFG {
			m.fgColor = color
			m.statusMsg = fmt.Sprintf("FG → %s", ColorName(color))
		} else {
			m.bgColor = color
			m.statusMsg = fmt.Sprintf("BG → %s", ColorName(color))
		}
		m.colorMode = false
		m.colorInput = ""
		m.colorErrMsg = ""
	case "backspace":
		runes := []rune(m.colorInput)
		if len(runes) > 0 {
			m.colorInput = string(runes[:len(runes)-1])
			m.colorErrMsg = ""
		}
	default:
		r := []rune(msg.String())
		if len(r) == 1 && (r[0] >= '0' && r[0] <= '9' || r[0] == 'd' || r[0] == 'D' || r[0] == '-') {
			m.colorInput += string(r[0])
			m.colorErrMsg = ""
		}
	}
	return m, nil
}

// ── Canvas key handler ────────────────────────────────────────────────────────

func (m Model) handleCanvasKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vW, vH := m.canvasViewSize()

	if m.textMode {
		return m.handleTextKey(msg, vW, vH)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true

	// Movement
	case "up", "k":
		if m.cursorY > 0 {
			m.cursorY--
		}
	case "down", "j":
		if m.cursorY < m.canvas.Height-1 {
			m.cursorY++
		}
	case "left", "h":
		if m.cursorX > 0 {
			m.cursorX--
		}
	case "right", "l":
		if m.cursorX < m.canvas.Width-1 {
			m.cursorX++
		}

	// Tools
	case "d":
		m.tool = ToolDraw
		m.statusMsg = "Tool: Draw"
	case "e":
		m.tool = ToolErase
		m.statusMsg = "Tool: Erase"
	case "f":
		m.tool = ToolFill
		m.statusMsg = "Tool: Fill — move cursor and press Space/Enter to fill"

	// Apply tool
	case " ", "enter":
		m = m.applyTool(m.cursorX, m.cursorY)

	// Erase
	case "backspace", "delete":
		m.canvas.Set(m.cursorX, m.cursorY, BlankCell)
		m.modified = true
		if msg.String() == "backspace" && m.cursorX > 0 {
			m.cursorX--
		}

	// Color prompts
	case "c":
		m.colorMode = true
		m.colorModeFG = true
		m.colorInput = ""
		m.colorErrMsg = ""
		m.statusMsg = ""
	case "b":
		m.colorMode = true
		m.colorModeFG = false
		m.colorInput = ""
		m.colorErrMsg = ""
		m.statusMsg = ""

	// Cycle character palette with Shift+Arrow (no conflict with drawing chars)
	case "shift+left":
		m.currentChar = prevChar(m.currentChar)
	case "shift+right":
		m.currentChar = nextChar(m.currentChar)

	// Text mode
	case "t":
		if m.tool == ToolDraw {
			m.textMode = true
			m.textModeStartX = m.cursorX
			m.statusMsg = ""
		}

	// File ops
	case "s", "ctrl+s":
		defaultName := m.filename
		if defaultName == "" {
			defaultName = "art.ansii"
		}
		m.saveMode = true
		m.saveInput = defaultName
		m.saveErrMsg = ""
	case "i":
		m.installConfirm = true

	// Import
	case "r":
		m.importMode = true
		m.importIsImg = false
		m.importASCII = false
		m.importInput = "~/"
		m.statusMsg = ""
	case "g":
		m.importMode = true
		m.importIsImg = true
		m.importASCII = false
		m.importInput = "~/"
		m.statusMsg = ""
	case "a":
		m.importMode = true
		m.importIsImg = false
		m.importASCII = true
		m.importInput = "~/"
		m.statusMsg = ""

	default:
		// Type a character to draw it — only in Draw tool mode.
		if m.tool == ToolDraw {
			r := []rune(msg.String())
			if len(r) == 1 && r[0] >= 32 && r[0] != 127 {
				m.currentChar = r[0]
				m.canvas.Set(m.cursorX, m.cursorY, Cell{
					Char: r[0],
					FG:   m.fgColor,
					BG:   m.bgColor,
				})
				m.modified = true
				if m.cursorX < m.canvas.Width-1 {
					m.cursorX++
				}
			}
		}
	}

	// Scroll viewport to follow cursor
	if m.cursorX < m.viewX {
		m.viewX = m.cursorX
	}
	if m.cursorX >= m.viewX+vW {
		m.viewX = m.cursorX - vW + 1
	}
	if m.cursorY < m.viewY {
		m.viewY = m.cursorY
	}
	if m.cursorY >= m.viewY+vH {
		m.viewY = m.cursorY - vH + 1
	}

	return m, nil
}

func (m Model) applyTool(x, y int) Model {
	switch m.tool {
	case ToolDraw:
		m.canvas.Set(x, y, Cell{Char: m.currentChar, FG: m.fgColor, BG: m.bgColor})
		m.modified = true
	case ToolErase:
		m.canvas.Set(x, y, BlankCell)
		m.modified = true
	case ToolFill:
		m.canvas.Fill(x, y, Cell{Char: m.currentChar, FG: m.fgColor, BG: m.bgColor})
		m.modified = true
	}
	return m
}

// ── Text mode ─────────────────────────────────────────────────────────────────

func (m Model) handleTextKey(msg tea.KeyMsg, vW, vH int) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "escape":
		m.textMode = false
		m.statusMsg = ""
	case "ctrl+s":
		defaultName := m.filename
		if defaultName == "" {
			defaultName = "art.ansii"
		}
		m.textMode = false
		m.saveMode = true
		m.saveInput = defaultName
		m.saveErrMsg = ""
	case "up":
		if m.cursorY > 0 {
			m.cursorY--
		}
	case "down":
		if m.cursorY < m.canvas.Height-1 {
			m.cursorY++
		}
	case "left":
		if m.cursorX > 0 {
			m.cursorX--
		}
	case "right":
		if m.cursorX < m.canvas.Width-1 {
			m.cursorX++
		}
	case "enter":
		if m.cursorY < m.canvas.Height-1 {
			m.cursorY++
		}
		m.cursorX = m.textModeStartX
	case "backspace":
		m.canvas.Set(m.cursorX, m.cursorY, BlankCell)
		m.modified = true
		if m.cursorX > 0 {
			m.cursorX--
		}
	case "delete":
		m.canvas.Set(m.cursorX, m.cursorY, BlankCell)
		m.modified = true
	default:
		r := []rune(msg.String())
		if len(r) == 1 && r[0] >= 32 && r[0] != 127 {
			m.canvas.Set(m.cursorX, m.cursorY, Cell{
				Char: r[0],
				FG:   m.fgColor,
				BG:   m.bgColor,
			})
			m.modified = true
			if m.cursorX < m.canvas.Width-1 {
				m.cursorX++
			}
		}
	}

	if m.cursorX < m.viewX {
		m.viewX = m.cursorX
	}
	if m.cursorX >= m.viewX+vW {
		m.viewX = m.cursorX - vW + 1
	}
	if m.cursorY < m.viewY {
		m.viewY = m.cursorY
	}
	if m.cursorY >= m.viewY+vH {
		m.viewY = m.cursorY - vH + 1
	}

	return m, nil
}

// ── Char palette cycling ──────────────────────────────────────────────────────

func prevChar(current rune) rune {
	for i, r := range charPalette {
		if r == current {
			if i == 0 {
				return charPalette[len(charPalette)-1]
			}
			return charPalette[i-1]
		}
	}
	return charPalette[0]
}

func nextChar(current rune) rune {
	for i, r := range charPalette {
		if r == current {
			if i == len(charPalette)-1 {
				return charPalette[0]
			}
			return charPalette[i+1]
		}
	}
	return charPalette[0]
}

// ── Path completion ───────────────────────────────────────────────────────────

// completePath returns filesystem completions for the given path prefix.
// Empty input or "~" lists the user's home directory.
// Hidden files (names starting with ".") are never shown.
// Directories are suffixed with /. Results are capped at 64.
func completePath(prefix string) []string {
	home, _ := os.UserHomeDir()

	expanded := prefix
	switch {
	case expanded == "" || expanded == "~":
		expanded = home + "/"
	case strings.HasPrefix(expanded, "~/"):
		if home != "" {
			expanded = home + expanded[1:]
		}
	}

	var dir, base string
	if strings.HasSuffix(expanded, "/") {
		dir, base = expanded, ""
	} else {
		dir = filepath.Dir(expanded)
		base = filepath.Base(expanded)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if base != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(base)) {
			continue
		}
		full := filepath.Join(dir, name)
		if e.IsDir() {
			full += "/"
		}
		if home != "" && strings.HasPrefix(full, home+"/") {
			full = "~/" + full[len(home)+1:]
		}
		results = append(results, full)
		if len(results) == 64 {
			break
		}
	}
	return results
}

func cycleCompletion(completions []string, idx int, input string, dir int) ([]string, int) {
	if len(completions) == 0 || strings.HasSuffix(input, "/") {
		completions = completePath(input)
		if dir > 0 {
			idx = 0
		} else {
			idx = len(completions) - 1
			if idx < 0 {
				idx = 0
			}
		}
		return completions, idx
	}
	n := len(completions)
	idx = (idx + dir + n) % n
	return completions, idx
}

func resolvePath(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	try := func(candidate string) string {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return ""
	}
	if c := try(p); c != "" {
		return c
	}
	if c := try(filepath.Join("art", filepath.Base(p))); c != "" {
		return c
	}
	exe, err := os.Executable()
	if err != nil {
		return p
	}
	exeDir := filepath.Dir(exe)
	projectRoot := filepath.Dir(exeDir)
	if c := try(filepath.Join(exeDir, p)); c != "" {
		return c
	}
	if c := try(filepath.Join(projectRoot, p)); c != "" {
		return c
	}
	if c := try(filepath.Join(projectRoot, "art", filepath.Base(p))); c != "" {
		return c
	}
	return p
}

func changeExt(filename, newExt string) string {
	if filename == "" {
		return ""
	}
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[:i] + newExt
		}
		if filename[i] == '/' {
			break
		}
	}
	return filename + newExt
}
