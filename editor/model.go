package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Tool represents the active drawing tool.
type Tool int

const (
	ToolDraw  Tool = iota
	ToolErase
	ToolFill
)

func (t Tool) String() string {
	return [...]string{"Draw", "Erase", "Fill"}[t]
}

// charPalette is the selectable character set (cycled with [ / ]).
var charPalette = []rune(
	" █▓▒░▄▀▌▐■□▪▫" +
		"─│┌┐└┘├┤┬┴┼═║" +
		"╔╗╚╝╠╣╦╩╬▲▼◄►" +
		"●○★☆♥♦♣♠◆◇♪♫☺" +
		"/\\|-_=+*#@!?~",
)

// Model is the main Bubbletea model.
type Model struct {
	canvas  *Canvas
	cursorX int
	cursorY int
	viewX   int // viewport scroll
	viewY   int

	fgColor     Color
	bgColor     Color
	currentChar rune
	tool        Tool

	filename  string
	modified  bool
	statusMsg string
	showHelp  bool

	// Text writing mode — all printable keys draw freely
	textMode       bool
	textModeStartX int

	// Inline color-pick prompt ([c] for FG, [b] for BG)
	// Input accepts 0-255 or "d" for default.
	colorMode    bool
	colorModeFG  bool // true = setting FG, false = BG
	colorInput   string
	colorErrMsg  string

	// Import mode
	importMode   bool
	importInput  string
	importIsImg  bool
	importASCII  bool
	importErrMsg string

	// Save-as mode
	saveMode   bool
	saveInput  string
	saveErrMsg string

	// Install splash confirmation
	installConfirm bool

	// Tab-completion for import/save prompts
	completions []string
	compIdx     int

	termW int
	termH int
}

func newModel(canvasW, canvasH int) Model {
	return Model{
		canvas:      NewCanvas(canvasW, canvasH),
		fgColor:     7, // White
		bgColor:     ColorDefault,
		currentChar: '█',
		tool:        ToolDraw,
		termW:       120,
		termH:       40,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// canvasViewSize returns the usable canvas display area given terminal size.
// The sidebar is 22 cols wide (20 content + 2 border).
func (m Model) canvasViewSize() (w, h int) {
	sidebarW := 22
	w = m.termW - sidebarW - 2 // canvas box border = 2; sidebar(22) + box(w+2) = termW
	h = m.termH - 6            // title(1) + border-top(1) + label(1) + border-bot(1) + status(2)
	if w < 10 {
		w = 10
	}
	if h < 5 {
		h = 5
	}
	return
}
