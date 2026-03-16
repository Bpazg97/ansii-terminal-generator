package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Tool represents the active drawing tool.
type Tool int

const (
	ToolDraw       Tool = iota
	ToolErase
	ToolFill
	ToolEyedropper
)

func (t Tool) String() string {
	return [...]string{"Draw", "Erase", "Fill", "Eyedrop"}[t]
}

// Focus represents which UI panel has keyboard focus.
type Focus int

const (
	FocusCanvas Focus = iota
	FocusFGColor
	FocusBGColor
	FocusChars
)

// Character palette rows.
var charPalette = [][]rune{
	{' ', '█', '▓', '▒', '░', '▄', '▀', '▌', '▐', '■', '□', '▪', '▫'},
	{'─', '│', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼', '═', '║'},
	{'╔', '╗', '╚', '╝', '╠', '╣', '╦', '╩', '╬', '▲', '▼', '◄', '►'},
	{'●', '○', '★', '☆', '♥', '♦', '♣', '♠', '◆', '◇', '♪', '♫', '☺'},
	{'/', '\\', '|', '-', '_', '=', '+', '*', '#', '@', '!', '?', '~'},
}

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

	focus       Focus
	colorCurX   int // color palette cursor
	colorCurY   int
	charCurX    int // char palette cursor
	charCurY    int
	fgSelecting bool // true=selecting FG in color panel, false=BG

	filename  string
	modified  bool
	statusMsg string
	showHelp  bool

	// Text writing mode — all printable keys draw freely
	textMode       bool
	textModeStartX int // column where text mode was activated (for Enter → new line)

	// Import mode (text or image)
	importMode   bool
	importInput  string
	importIsImg  bool   // true = image import, false = text import
	importErrMsg string // last error message (stays in import mode so user can fix path)

	termW int
	termH int
}

func newModel(canvasW, canvasH int) Model {
	return Model{
		canvas:      NewCanvas(canvasW, canvasH),
		fgColor:     7,  // White
		bgColor:     ColorDefault,
		currentChar: '█',
		tool:        ToolDraw,
		focus:       FocusCanvas,
		fgSelecting: true,
		termW:       120,
		termH:       40,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// canvasViewSize returns the usable canvas display area given terminal size.
func (m Model) canvasViewSize() (w, h int) {
	sidebarW := 36
	w = m.termW - sidebarW - 2 // box border adds 2; sidebar(36)+box(w+2)=termW
	h = m.termH - 6 // title(1) + border-top(1) + label(1) + border-bot(1) + status(2)
	if w < 10 {
		w = 10
	}
	if h < 5 {
		h = 5
	}
	return
}
