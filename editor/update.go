package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Import mode intercepts all keys
		if m.importMode {
			return m.handleImportKey(msg)
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.focus {
	case FocusCanvas:
		return m.handleCanvasKey(msg)
	case FocusFGColor, FocusBGColor:
		return m.handleColorKey(msg)
	case FocusChars:
		return m.handleCharKey(msg)
	}
	return m, nil
}

func (m Model) handleImportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "escape":
		m.importMode = false
		m.importInput = ""
		m.importErrMsg = ""

	case "enter":
		if m.importInput == "" {
			break
		}
		path := resolvePath(m.importInput)
		var (
			c   *Canvas
			err error
		)
		if m.importIsImg {
			c, err = importFromImage(path, 80)
		} else {
			c, err = importFromText(path)
		}
		if err != nil {
			// Stay in import mode so the user can fix the path in-place.
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
		}

	case "backspace":
		runes := []rune(m.importInput)
		if len(runes) > 0 {
			m.importInput = string(runes[:len(runes)-1])
			m.importErrMsg = ""
		}

	default:
		r := []rune(msg.String())
		if len(r) == 1 && r[0] >= 32 {
			m.importInput += string(r[0])
			m.importErrMsg = ""
		}
	}
	return m, nil
}

// resolvePath expands ~ and tries to find the file relative to common locations.
func resolvePath(p string) string {
	// Expand ~/
	if len(p) >= 2 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		p = home + p[1:]
	}
	// If file exists as-is, use it directly.
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Try relative to the executable's parent directory (project root).
	if exe, err := os.Executable(); err == nil {
		projectRoot := filepath.Dir(filepath.Dir(exe)) // editor/../
		candidate := filepath.Join(projectRoot, p)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		// Also try art/ subdirectory
		candidate2 := filepath.Join(projectRoot, "art", filepath.Base(p))
		if _, err := os.Stat(candidate2); err == nil {
			return candidate2
		}
	}
	return p // return as-is; error will be reported by caller
}

func (m Model) handleCanvasKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vW, vH := m.canvasViewSize()

	// TEXT MODE: all printable chars draw freely; only special keys are intercepted.
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
	case "p":
		m.tool = ToolEyedropper
		m.statusMsg = "Tool: Eyedropper — press Space/Enter to pick color+char"

	// Draw / apply tool
	case " ", "enter":
		m = m.applyTool(m.cursorX, m.cursorY)

	// Erase with backspace
	case "backspace", "delete":
		m.canvas.Set(m.cursorX, m.cursorY, Cell{Char: ' ', FG: ColorDefault, BG: ColorDefault})
		m.modified = true
		if msg.String() == "backspace" && m.cursorX > 0 {
			m.cursorX--
		}

	// Panel switching
	case "tab":
		m.focus = FocusFGColor
		m.fgSelecting = true
	case "shift+tab":
		m.focus = FocusChars

	// File ops
	case "s", "ctrl+s":
		if m.filename == "" {
			m.filename = "art.ansii"
		}
		if err := saveCanvas(m.canvas, m.filename); err != nil {
			m.statusMsg = fmt.Sprintf("Error saving: %v", err)
		} else {
			m.modified = false
			m.statusMsg = fmt.Sprintf("Saved → %s", m.filename)
		}
	case "x":
		outFile := changeExt(m.filename, ".ansi")
		if outFile == "" {
			outFile = "art.ansi"
		}
		if err := exportANSI(m.canvas, outFile); err != nil {
			m.statusMsg = fmt.Sprintf("Error exporting: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("Exported → %s", outFile)
		}
	case "i":
		outFile := changeExt(m.filename, ".ansi")
		if outFile == "" {
			outFile = "art.ansi"
		}
		if err := exportANSI(m.canvas, outFile); err != nil {
			m.statusMsg = fmt.Sprintf("Error exporting: %v", err)
			break
		}
		if err := installToShell(outFile); err != nil {
			m.statusMsg = fmt.Sprintf("Error installing: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("Installed! Restart terminal to see art. (file: %s)", outFile)
		}

	// Import plain text / braille file
	case "r":
		m.importMode = true
		m.importIsImg = false
		m.importInput = ""
		m.statusMsg = ""

	// Import image as colored ANSI art
	case "g":
		m.importMode = true
		m.importIsImg = true
		m.importInput = ""
		m.statusMsg = ""

	// Enter text writing mode
	case "t":
		m.textMode = true
		m.textModeStartX = m.cursorX
		m.statusMsg = ""

	default:
		// Type a character to draw it
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
		m.canvas.Set(x, y, Cell{Char: ' ', FG: ColorDefault, BG: ColorDefault})
		m.modified = true
	case ToolFill:
		m.canvas.Fill(x, y, Cell{Char: m.currentChar, FG: m.fgColor, BG: m.bgColor})
		m.modified = true
	case ToolEyedropper:
		cell := m.canvas.Get(x, y)
		m.fgColor = cell.FG
		m.bgColor = cell.BG
		if cell.Char != ' ' && cell.Char != 0 {
			m.currentChar = cell.Char
		}
		m.statusMsg = fmt.Sprintf("Picked: char='%s'  fg=%s  bg=%s",
			string(cell.Char), ColorName(cell.FG), ColorName(cell.BG))
	}
	return m
}

func (m Model) handleColorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "escape":
		// Return to canvas WITHOUT applying the color
		m.focus = FocusCanvas

	case "enter":
		// Apply selected color and return to canvas
		m.applySelectedColor()
		m.focus = FocusCanvas

	case " ":
		// Apply selected color, stay in panel
		m.applySelectedColor()

	case "up":
		if m.colorCurY > 0 {
			m.colorCurY--
		}
	case "down":
		if m.colorCurY < 1 {
			m.colorCurY++
		}
	case "left":
		if m.colorCurX > 0 {
			m.colorCurX--
		}
	case "right":
		if m.colorCurX < 7 {
			m.colorCurX++
		}
	case "f", "F":
		m.focus = FocusFGColor
		m.fgSelecting = true
	case "b", "B":
		m.focus = FocusBGColor
		m.fgSelecting = false
	case "tab":
		m.focus = FocusChars
	case "shift+tab":
		m.focus = FocusCanvas
	case "d":
		// Set Default (no color) and return
		if m.fgSelecting {
			m.fgColor = ColorDefault
		} else {
			m.bgColor = ColorDefault
		}
		m.focus = FocusCanvas
	}

	// Sync focus with fgSelecting
	if m.focus == FocusFGColor {
		m.fgSelecting = true
	} else if m.focus == FocusBGColor {
		m.fgSelecting = false
	}
	return m, nil
}

// applySelectedColor sets fgColor or bgColor from the palette cursor position.
func (m *Model) applySelectedColor() {
	selected := Color(m.colorCurY*8 + m.colorCurX)
	if m.fgSelecting {
		m.fgColor = selected
	} else {
		m.bgColor = selected
	}
}

func (m Model) handleCharKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "escape":
		// Return without selecting
		m.focus = FocusCanvas
	case "enter", " ":
		if m.charCurY < len(charPalette) && m.charCurX < len(charPalette[m.charCurY]) {
			m.currentChar = charPalette[m.charCurY][m.charCurX]
		}
		m.focus = FocusCanvas
	case "up":
		if m.charCurY > 0 {
			m.charCurY--
			if m.charCurX >= len(charPalette[m.charCurY]) {
				m.charCurX = len(charPalette[m.charCurY]) - 1
			}
		}
	case "down":
		if m.charCurY < len(charPalette)-1 {
			m.charCurY++
			if m.charCurX >= len(charPalette[m.charCurY]) {
				m.charCurX = len(charPalette[m.charCurY]) - 1
			}
		}
	case "left":
		if m.charCurX > 0 {
			m.charCurX--
		}
	case "right":
		if m.charCurY < len(charPalette) && m.charCurX < len(charPalette[m.charCurY])-1 {
			m.charCurX++
		}
	case "tab":
		m.focus = FocusCanvas
	case "shift+tab":
		m.focus = FocusBGColor
		m.fgSelecting = false
	}
	return m, nil
}

// handleTextKey handles keyboard events while in text writing mode.
// All printable characters draw freely; only navigation/control keys are special.
func (m Model) handleTextKey(msg tea.KeyMsg, vW, vH int) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "escape", "q":
		// Exit text mode
		m.textMode = false
		m.statusMsg = ""

	case "ctrl+s":
		// Save still works in text mode
		if m.filename == "" {
			m.filename = "art.ansii"
		}
		if err := saveCanvas(m.canvas, m.filename); err != nil {
			m.statusMsg = fmt.Sprintf("Error saving: %v", err)
		} else {
			m.modified = false
			m.statusMsg = fmt.Sprintf("Saved → %s", m.filename)
		}

	// Navigation — same as normal mode
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

	// Enter: go to next line, return to the column where text mode started
	case "enter":
		if m.cursorY < m.canvas.Height-1 {
			m.cursorY++
		}
		m.cursorX = m.textModeStartX

	// Backspace: erase current cell and move left
	case "backspace":
		m.canvas.Set(m.cursorX, m.cursorY, Cell{Char: ' ', FG: ColorDefault, BG: ColorDefault})
		m.modified = true
		if m.cursorX > 0 {
			m.cursorX--
		}

	// Delete: erase in place without moving
	case "delete":
		m.canvas.Set(m.cursorX, m.cursorY, Cell{Char: ' ', FG: ColorDefault, BG: ColorDefault})
		m.modified = true

	default:
		// Draw ANY printable character (including d, e, f, s, q, etc.)
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

// changeExt changes file extension (or appends if no ext).
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
