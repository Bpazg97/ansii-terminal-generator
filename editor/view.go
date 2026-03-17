package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")).  // bright cyan
			Background(lipgloss.Color("17")).  // dark blue (ANSI 256)
			Padding(0, 2)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")) // dark gray

	focusBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("14")) // bright cyan

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("11")) // bright yellow

	statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("250")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("14")).
			Foreground(lipgloss.Color("0"))

	importPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("11"))
)

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	filenameStr := m.filename
	if filenameStr == "" {
		filenameStr = "(unsaved)"
	}
	modMark := ""
	if m.modified {
		modMark = "*"
	}
	title := titleStyle.Render(" ANSI Art Editor ") + " " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("242")).
			Render(fmt.Sprintf("[%s]%s", filenameStr, modMark))

	canvas := m.renderCanvas()
	sidebar := m.renderSidebar()
	content := lipgloss.JoinHorizontal(lipgloss.Top, canvas, sidebar)
	status := m.renderStatus()

	return lipgloss.JoinVertical(lipgloss.Left, title, content, status)
}

// ── Canvas ────────────────────────────────────────────────────────────────────

var dimDot = lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render("·")

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
		if vis := lipgloss.Width(rowStr); vis < vW {
			rowStr += strings.Repeat(" ", vW-vis)
		}
		rows = append(rows, rowStr)
	}

	modeTag := ""
	if m.textMode {
		modeTag = " " + lipgloss.NewStyle().
			Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0")).Bold(true).
			Render("T")
	}
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).
		Render(fmt.Sprintf("Canvas %dx%d  cur(%d,%d)", m.canvas.Width, m.canvas.Height, m.cursorX, m.cursorY)) +
		modeTag

	content := label + "\n" + strings.Join(rows, "\n")
	return focusBorderStyle.Render(content)
}

func renderCell(cell Cell, isCursor bool) string {
	ch := cell.Char
	if ch == 0 {
		ch = ' '
	}
	s := lipgloss.NewStyle()
	if cell.FG != ColorDefault {
		s = s.Foreground(lipgloss.Color(fmt.Sprintf("%d", int(cell.FG))))
	}
	if cell.BG != ColorDefault {
		s = s.Background(lipgloss.Color(fmt.Sprintf("%d", int(cell.BG))))
	}
	if isCursor {
		s = s.Reverse(true)
	}
	return s.Render(string(ch))
}

// ── Sidebar ───────────────────────────────────────────────────────────────────
// The sidebar is compact: FG/BG display, current char, and tool selector.
// Width(20) → content 20, border 2 → outer 22. Matches canvasViewSize sidebarW=22.

func (m Model) renderSidebar() string {
	const w = 20
	var sb strings.Builder

	// FG / BG
	fgSwatch := renderColorSwatch(m.fgColor)
	bgSwatch := renderColorSwatch(m.bgColor)
	sb.WriteString(fmt.Sprintf("FG %s %s\n", fgSwatch, ColorName(m.fgColor)))
	sb.WriteString(fmt.Sprintf("BG %s %s\n", bgSwatch, ColorName(m.bgColor)))

	sb.WriteString("\n")

	// Current character with preview
	cs := lipgloss.NewStyle().Bold(true)
	if m.fgColor != ColorDefault {
		cs = cs.Foreground(lipgloss.Color(fmt.Sprintf("%d", int(m.fgColor))))
	}
	if m.bgColor != ColorDefault {
		cs = cs.Background(lipgloss.Color(fmt.Sprintf("%d", int(m.bgColor))))
	}
	charPreview := cs.Render(string(m.currentChar))
	sb.WriteString("Char: " + charPreview + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render("S+←→ cycle") + "\n")

	sb.WriteString("\n")

	// Tools
	sb.WriteString(sectionTitleStyle.Render("Tool") + "\n")
	tools := []struct {
		key  string
		t    Tool
	}{{"d", ToolDraw}, {"e", ToolErase}, {"f", ToolFill}}
	for _, entry := range tools {
		label := fmt.Sprintf("[%s]%s", entry.key, entry.t.String())
		if m.tool == entry.t {
			sb.WriteString(selectedStyle.Render(label))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(label))
		}
		sb.WriteString("\n")
	}

	return borderStyle.Width(w).Render(sb.String())
}

func renderColorSwatch(c Color) string {
	if c == ColorDefault {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render("--")
	}
	idx := fmt.Sprintf("%d", int(c))
	return lipgloss.NewStyle().Background(lipgloss.Color(idx)).Render("  ")
}

// ── Status bar ────────────────────────────────────────────────────────────────

func (m Model) renderStatus() string {
	// Install confirmation
	if m.installConfirm {
		outFile := changeExt(m.filename, ".ansi")
		if outFile == "" {
			outFile = "art.ansi"
		}
		warn := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		line1 := warn.Render("INSTALL SPLASH") + "  " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("250")).
				Render(fmt.Sprintf("→ ~/.config/ansii/splash.ansi + .bashrc/.zshrc  (export: %s)", outFile))
		hint := kh("[Enter/y]", " confirm   ") + kh("[Esc/n]", " cancel")
		return statusStyle.Width(m.termW).Render(line1 + "\n" + hint)
	}

	// Save prompt
	if m.saveMode {
		prompt := importPromptStyle.Render("Save (.ansii=project  .ansi=export): ")
		cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("█")
		line1 := prompt + m.saveInput + cursor
		if m.saveErrMsg != "" {
			line1 += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render("  ✗ " + m.saveErrMsg)
		}
		var line2 string
		if len(m.completions) > 0 {
			line2 = renderCompletionBar(m.completions, m.compIdx) + "   " +
				kh("[Tab]", " next  ") + kh("[S+Tab]", " prev  ") + kh("[Enter]", " confirm  ") + kh("[Esc]", " cancel")
		} else {
			line2 = kh("[Enter]", " save   ") + kh("[Esc]", " cancel   ") + kh("[Tab]", " complete path")
		}
		return statusStyle.Width(m.termW).Render(line1 + "\n" + line2)
	}

	// Import prompt
	if m.importMode {
		var promptText string
		switch {
		case m.importASCII:
			promptText = "Import image as ASCII (PNG/JPG): "
		case m.importIsImg:
			promptText = "Import image half-block (PNG/JPG): "
		default:
			promptText = "Import text file: "
		}
		cwd, _ := os.Getwd()
		prompt := importPromptStyle.Render(promptText)
		cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("█")
		line1 := prompt + m.importInput + cursor
		if m.importErrMsg != "" {
			line1 += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render("  ✗ " + m.importErrMsg)
		}
		var line2 string
		if len(m.completions) > 0 {
			line2 = renderCompletionBar(m.completions, m.compIdx) + "   " +
				kh("[Tab]", " next  ") + kh("[S+Tab]", " prev  ") + kh("[Enter]", " import  ") + kh("[Esc]", " cancel")
		} else {
			line2 = kh("[Enter]", " import   ") + kh("[Esc]", " cancel   ") + kh("[Tab]", " complete   ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("cwd: "+cwd)
		}
		return statusStyle.Width(m.termW).Render(line1 + "\n" + line2)
	}

	// Color input prompt
	if m.colorMode {
		which := "FG"
		if !m.colorModeFG {
			which = "BG"
		}
		prompt := importPromptStyle.Render(fmt.Sprintf("%s color (0-255, d=default): ", which))
		cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("█")
		line1 := prompt + m.colorInput + cursor
		if m.colorErrMsg != "" {
			line1 += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render("  ✗ " + m.colorErrMsg)
		}
		line2 := kh("[Enter]", " confirm   ") + kh("[Esc]", " cancel")
		return statusStyle.Width(m.termW).Render(line1 + "\n" + line2)
	}

	// Normal status
	fgSw := renderColorSwatch(m.fgColor)
	bgSw := renderColorSwatch(m.bgColor)
	info := fmt.Sprintf("(%d,%d)  FG:%s%-10s  BG:%s%-10s  Char:%s  Tool:%s",
		m.cursorX, m.cursorY,
		fgSw, ColorName(m.fgColor),
		bgSw, ColorName(m.bgColor),
		string(m.currentChar),
		m.tool.String(),
	)

	var msg string
	if m.statusMsg != "" {
		msg = "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(m.statusMsg)
	}

	// Key hints — single line, ≤78 visible chars on an 80-col terminal.
	// Text mode does NOT change the hints; the canvas "T" badge is enough context.
	textHint := ""
	if m.tool == ToolDraw {
		textHint = kh("[t]", "text ")
	}
	escHint := ""
	if m.textMode {
		escHint = kh("[Esc]", "exit-text  ")
	}
	keys := escHint +
		textHint +
		kh("[c]", "fg ") +
		kh("[b]", "bg ") +
		kh("[S+←→]", "char ") +
		kh("[r]", "txt ") +
		kh("[g]", "img ") +
		kh("[a]", "ascii ") +
		kh("[s]", "save ") +
		kh("[i]", "install ") +
		kh("[?]", "help ") +
		kh("[q]", "quit")

	return statusStyle.Width(m.termW).Render(info + msg + "\n" + keys)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func renderCompletionBar(completions []string, idx int) string {
	const maxVisible = 5
	n := len(completions)
	start := idx - 2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > n {
		end = n
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	arrow := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	var parts []string
	if start > 0 {
		parts = append(parts, arrow.Render(fmt.Sprintf("◀%d", start)))
	}
	for i := start; i < end; i++ {
		name := filepath.Base(completions[i])
		if strings.HasSuffix(completions[i], "/") {
			name += "/"
		}
		if i == idx {
			parts = append(parts, selectedStyle.Render(" "+name+" "))
		} else {
			parts = append(parts, dim.Render(name))
		}
	}
	if end < n {
		parts = append(parts, arrow.Render(fmt.Sprintf("+%d▶", n-end)))
	}
	return strings.Join(parts, "  ")
}

// kh renders a key hint: key in bright cyan, description in dim gray.
func kh(key, desc string) string {
	k := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render(key)
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(desc)
	return k + d
}

// ── Help screen ───────────────────────────────────────────────────────────────

func helpLine(key, desc string) string {
	k := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Render(fmt.Sprintf("  %-20s", key))
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(desc)
	return k + d
}

func helpSection(title string) string {
	return "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).Render("  "+title) + "\n"
}

func (m Model) renderHelp() string {
	lines := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render("  ANSI Art Editor — Help"),
		"",
		helpSection("NAVIGATION"),
		helpLine("↑↓←→  /  h j k l", "Move cursor"),
		helpSection("DRAWING  (Draw tool)"),
		helpLine("Space / Enter", "Apply current tool at cursor"),
		helpLine("Any printable key", "Draw char and advance (Draw tool only)"),
		helpLine("Backspace", "Erase and move left"),
		helpLine("Delete", "Erase in place"),
		helpSection("TOOLS"),
		helpLine("[d]  Draw", "Place current char+colors"),
		helpLine("[e]  Erase", "Clear a cell"),
		helpLine("[f]  Fill", "Flood fill connected region"),
		helpLine("[t]  Text mode", "Type freely — Esc to exit (Draw tool only)"),
		helpSection("COLOR"),
		helpLine("[c]", "Set FG color — type 0-255 or d for default"),
		helpLine("[b]", "Set BG color — type 0-255 or d for default"),
		helpSection("CHARACTER"),
		helpLine("Shift+←  /  Shift+→", "Cycle through character palette"),
		helpLine("Any printable key", "Draw that char directly (Draw mode)"),
		helpSection("FILE"),
		helpLine("[s] / Ctrl+S", "Save  (.ansii=project  .ansi=raw export)"),
		helpLine("[i]", "Install as terminal splash  (asks confirmation)"),
		helpSection("IMPORT"),
		helpLine("[r]", "Import text file into canvas"),
		helpLine("[g]", "Import PNG/JPG as half-block art (▀/▄)"),
		helpLine("[a]", "Import PNG/JPG as ASCII art"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("  Press any key to close"),
	}
	return strings.Join(lines, "\n")
}
