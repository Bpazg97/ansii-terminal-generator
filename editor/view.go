package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffff")).
			Background(lipgloss.Color("#000080")).
			Padding(0, 2)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444"))

	focusBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00ffff"))

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffff00"))

	statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#222222")).
			Foreground(lipgloss.Color("#aaaaaa")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#00ffff")).
			Foreground(lipgloss.Color("#000000"))

	importPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffff00"))
)

func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	title := titleStyle.Render(" ANSI Art Editor ") + " " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
			Render(fmt.Sprintf("[%s] %s", m.filename, modifiedMark(m.modified)))

	canvas := m.renderCanvas()
	sidebar := m.renderSidebar()

	content := lipgloss.JoinHorizontal(lipgloss.Top, canvas, sidebar)

	status := m.renderStatus()

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		status,
	)
}

func modifiedMark(modified bool) string {
	if modified {
		return "*"
	}
	return ""
}

// dimDot is the out-of-canvas-bounds marker, rendered once and reused.
var dimDot = lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2a2a")).Render("·")

func (m Model) renderCanvas() string {
	vW, vH := m.canvasViewSize()

	var rows []string
	for row := m.viewY; row < m.viewY+vH; row++ {
		var sb strings.Builder
		for col := m.viewX; col < m.viewX+vW; col++ {
			if row >= m.canvas.Height || col >= m.canvas.Width {
				sb.WriteString(dimDot)
			} else {
				cell := m.canvas.Get(col, row)
				sb.WriteString(renderCell(cell, col == m.cursorX && row == m.cursorY))
			}
		}
		rowStr := sb.String()

		// Pad to exactly vW visible columns.
		// lipgloss.Width strips ANSI codes before measuring, so this is safe
		// even when cells contain escape sequences.
		if vis := lipgloss.Width(rowStr); vis < vW {
			rowStr += strings.Repeat(" ", vW-vis)
		}
		rows = append(rows, rowStr)
	}

	textModeTag := ""
	if m.textMode {
		textModeTag = "  " + lipgloss.NewStyle().
			Background(lipgloss.Color("#ffff00")).Foreground(lipgloss.Color("#000000")).Bold(true).
			Render(" ✎ TEXT ")
	}
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render(fmt.Sprintf("Canvas %dx%d  cursor(%d,%d)%s",
			m.canvas.Width, m.canvas.Height, m.cursorX, m.cursorY, textModeTag))

	// Do NOT use Width() here — each row is already padded to vW so lipgloss
	// does not need to reflow ANSI-coded content (which caused corruption).
	content := label + "\n" + strings.Join(rows, "\n")
	boxStyle := borderStyle
	if m.focus == FocusCanvas {
		boxStyle = focusBorderStyle
	}
	return boxStyle.Render(content)
}

func renderCell(cell Cell, isCursor bool) string {
	ch := cell.Char
	if ch == 0 {
		ch = ' '
	}
	s := lipgloss.NewStyle()
	if cell.FG != ColorDefault {
		s = s.Foreground(lipgloss.Color(ColorCode(cell.FG)))
	}
	if cell.BG != ColorDefault {
		s = s.Background(lipgloss.Color(ColorCode(cell.BG)))
	}
	if isCursor {
		s = s.Reverse(true)
	}
	return s.Render(string(ch))
}

func (m Model) renderSidebar() string {
	sidebarW := 34

	sections := []string{
		m.renderColorSection(sidebarW),
		m.renderCharSection(sidebarW),
		m.renderToolSection(sidebarW),
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderColorSection(w int) string {
	var sb strings.Builder

	// FG/BG display
	fgLabel := "FG: "
	fgBlock := renderColorSwatch(m.fgColor, false) + " " + ColorName(m.fgColor)
	bgLabel := "BG: "
	bgBlock := renderColorSwatch(m.bgColor, false) + " " + ColorName(m.bgColor)

	if m.focus == FocusFGColor {
		fgLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ffff")).Render("FG▶ ")
	}
	if m.focus == FocusBGColor {
		bgLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ffff")).Render("BG▶ ")
	}

	sb.WriteString(fgLabel + fgBlock + "\n")
	sb.WriteString(bgLabel + bgBlock + "\n\n")

	sb.WriteString(sectionTitleStyle.Render("Colors") + "\n")
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("[f]FG [b]BG [d]Default")
	sb.WriteString(hint + "\n")

	// Color grid 8x2
	for row := 0; row < 2; row++ {
		for col := 0; col < 8; col++ {
			colorIdx := Color(row*8 + col)
			isCursor := m.colorCurX == col && m.colorCurY == row &&
				(m.focus == FocusFGColor || m.focus == FocusBGColor)
			sb.WriteString(renderColorSwatch(colorIdx, isCursor))
		}
		// Color names hint
		if row == 0 {
			sb.WriteString(" 0-7\n")
		} else {
			sb.WriteString(" 8-15\n")
		}
	}

	boxStyle := borderStyle
	if m.focus == FocusFGColor || m.focus == FocusBGColor {
		boxStyle = focusBorderStyle
	}
	return boxStyle.Width(w).Render(sb.String())
}

func renderColorSwatch(c Color, selected bool) string {
	s := lipgloss.NewStyle()
	if c == ColorDefault {
		s = s.Foreground(lipgloss.Color("#666666"))
		if selected {
			s = s.Reverse(true)
		}
		return s.Render("░░")
	}
	hex := ColorCode(c)
	s = s.Background(lipgloss.Color(hex)).Foreground(lipgloss.Color(hex))
	if selected {
		s = lipgloss.NewStyle().
			Background(lipgloss.Color("#ffffff")).
			Foreground(lipgloss.Color(hex))
	}
	if selected {
		return s.Render("▶◀")
	}
	return s.Render("██")
}

func (m Model) renderCharSection(w int) string {
	var sb strings.Builder
	sb.WriteString(sectionTitleStyle.Render("Characters") + "\n")

	for rowIdx, row := range charPalette {
		for colIdx, ch := range row {
			isCursor := m.focus == FocusChars && m.charCurX == colIdx && m.charCurY == rowIdx
			isCurrentChar := ch == m.currentChar

			s := lipgloss.NewStyle()
			if isCursor {
				s = selectedStyle
			} else if isCurrentChar {
				s = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#ffff00")).
					Bold(true)
			}
			sb.WriteString(s.Render(string(ch)))
			sb.WriteRune(' ')
		}
		sb.WriteRune('\n')
	}

	sb.WriteString("\n")
	currentLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Current: ")
	currentChar := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorCode(m.fgColor))).
		Background(lipgloss.Color(ColorCode(m.bgColor))).
		Bold(true).
		Render(string(m.currentChar))
	sb.WriteString(currentLabel + currentChar)

	boxStyle := borderStyle
	if m.focus == FocusChars {
		boxStyle = focusBorderStyle
	}
	return boxStyle.Width(w).Render(sb.String())
}

func (m Model) renderToolSection(w int) string {
	tools := []Tool{ToolDraw, ToolErase, ToolFill, ToolEyedropper}
	keys := []string{"d", "e", "f", "p"}
	var parts []string
	for i, t := range tools {
		label := fmt.Sprintf("[%s]%s", keys[i], t.String())
		if m.tool == t && m.focus == FocusCanvas {
			label = selectedStyle.Render(label)
		} else {
			label = lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa")).Render(label)
		}
		parts = append(parts, label)
	}
	content := strings.Join(parts, " ")
	return borderStyle.Width(w).Render(sectionTitleStyle.Render("Tools") + "\n" + content)
}

// kh renders a key hint: key in bright cyan, description in dim gray.
func kh(key, desc string) string {
	k := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")).Bold(true).Render(key)
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(desc)
	return k + d
}

func (m Model) renderStatus() string {
	// Import mode: show filename input prompt
	if m.importMode {
		var promptText, hintExt string
		if m.importIsImg {
			promptText = "Import image (PNG/JPG): "
			hintExt = "e.g. /path/to/my-art.png"
		} else {
			promptText = "Import text/braille file: "
			hintExt = "e.g. /path/to/my-art.txt"
		}
		cwd, _ := os.Getwd()
		prompt := importPromptStyle.Render(promptText)
		cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render("█")
		hint := kh("[Enter]", " import   ") + kh("[Esc]", " cancel   ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(hintExt) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).Render("   cwd: "+cwd)
		line1 := prompt + m.importInput + cursor
		if m.importErrMsg != "" {
			errLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Bold(true).Render("  ✗ " + m.importErrMsg)
			return statusStyle.Width(m.termW).Render(line1 + errLine + "\n" + hint)
		}
		return statusStyle.Width(m.termW).Render(line1 + "\n" + hint)
	}

	fgPrev := renderColorSwatch(m.fgColor, false)
	bgPrev := renderColorSwatch(m.bgColor, false)

	pos := fmt.Sprintf("(%d,%d)", m.cursorX, m.cursorY)
	info := fmt.Sprintf("Pos:%-8s FG:%s%-10s BG:%s%-10s Char:%s Tool:%-8s",
		pos,
		fgPrev, ColorName(m.fgColor),
		bgPrev, ColorName(m.bgColor),
		string(m.currentChar),
		m.tool.String(),
	)

	var msg string
	if m.statusMsg != "" {
		msg = " | " + lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(m.statusMsg)
	}

	var keys string
	if m.textMode {
		keys = lipgloss.NewStyle().
			Background(lipgloss.Color("#ffff00")).Foreground(lipgloss.Color("#000000")).Bold(true).
			Render(" ✎ TEXT MODE ") + "  " +
			kh("[↑↓←→]", " move   ") +
			kh("[Enter]", " new line   ") +
			kh("[Backspace]", " erase   ") +
			kh("[Esc/q]", " exit text mode")
	} else {
		keys = kh("[t]", "Text   ") +
			kh("[Tab]", "Panel   ") +
			kh("[r]", "Import   ") +
			kh("[g]", "Image   ") +
			kh("[s]", "Save   ") +
			kh("[x]", "Export   ") +
			kh("[i]", "Install   ") +
			kh("[?]", "Help   ") +
			kh("[q]", "Quit")
	}

	return statusStyle.Width(m.termW).Render(info + msg + "\n" + keys)
}

func helpLine(key, desc string) string {
	k := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")).Bold(true).Render(fmt.Sprintf("  %-22s", key))
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(desc)
	return k + d
}

func helpSection(title string) string {
	return "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Bold(true).Render("  "+title) + "\n"
}

func (m Model) renderHelp() string {
	lines := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true).Render("  ANSI Art Editor — Help"),
		"",
		helpSection("NAVIGATION"),
		helpLine("↑↓←→  /  h j k l", "Move cursor"),
		helpLine("Tab / Shift+Tab", "Switch panel"),
		helpSection("DRAWING  (canvas panel)"),
		helpLine("Space / Enter", "Apply current tool"),
		helpLine("Type any char", "Draw it and advance"),
		helpLine("Backspace", "Erase cell and move left"),
		helpLine("Delete", "Erase in place"),
		helpSection("TOOLS"),
		helpLine("[d]  Draw", "Draw current char+colors"),
		helpLine("[e]  Erase", "Clear a cell"),
		helpLine("[f]  Fill", "Flood fill area"),
		helpLine("[p]  Eyedrop", "Pick color+char from canvas"),
		helpLine("[t]  Text mode", "All keys draw freely — Esc to exit"),
		helpSection("COLOR PANEL  (Tab from canvas)"),
		helpLine("↑↓←→", "Navigate palette"),
		helpLine("[f]", "Select FG color"),
		helpLine("[b]", "Select BG color"),
		helpLine("[d]", "Reset to default (no color)"),
		helpLine("Enter / Esc", "Apply and return to canvas"),
		helpSection("CHARACTER PANEL  (Tab from colors)"),
		helpLine("↑↓←→", "Navigate character palette"),
		helpLine("Enter / Esc", "Select and return to canvas"),
		helpSection("FILE"),
		helpLine("[s] / Ctrl+S", "Save project  (.ansii JSON)"),
		helpLine("[x]", "Export as .ansi  (raw escape codes)"),
		helpLine("[i]", "Install as terminal splash screen"),
		helpSection("IMPORT"),
		helpLine("[r]  Import text", "Load .txt / braille art into canvas"),
		helpLine("[g]  Import image", "Load PNG/JPG → colored ANSI art"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("  Press any key to close"),
	}

	return strings.Join(lines, "\n")
}
