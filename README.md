# ANSI Art Editor

```
  █████╗ ███╗   ██╗███████╗██╗██╗
 ██╔══██╗████╗  ██║██╔════╝██║██║
 ███████║██╔██╗ ██║███████╗██║██║
 ██╔══██║██║╚██╗██║╚════██║██║██║
 ██║  ██║██║ ╚████║███████║██║██║
 ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚═╝╚═╝
```

A terminal-based ANSI art editor written in Go. Draw sprites, type-set art, or import images, then install the result as your terminal splash screen.

---

## Features

- **Editable canvas** with cursor navigation and viewport scrolling
- **256 ANSI colors** for foreground and background per cell
- **Character palette** — Unicode blocks, box-drawing, and symbols, cycled with `Shift+←/→`
- **Tools** — Draw, Erase, Flood Fill, Text mode
- **Inline color prompts** — type a number (0–255) or `d` for default; no separate color panel
- **Import text art** — load any `.txt` file directly into the canvas
- **Import image (half-block)** — PNG/JPG → colored `▀`/`▄` art, best visual quality
- **Import image (ASCII)** — PNG/JPG → ASCII character art with ANSI-256 colors, no external tools
- **Save prompt** — always asks for a filename; no silent overwrites
- **Save/Load** in `.ansii` (human-readable JSON)
- **Export** to `.ansi` (raw ANSI escape codes, works in any terminal)
- **Install splash** — one keypress writes the art to your `.bashrc`/`.zshrc`

---

## Installation

Requires Go 1.21+:

```bash
git clone https://github.com/YOUR_USERNAME/ansii
cd ansii/editor
go build -o ansii .
sudo mv ansii /usr/local/bin/
```

---

## Usage

```bash
# New blank canvas (60×30 by default)
ansii

# Open or create a project file
ansii -f my_art.ansii

# Custom canvas size
ansii -f hero.ansii -w 80 -h 40

# Import a text art file and open it for editing
ansii -import art.txt -f art.ansii

# Import an image as half-block ANSI art (best quality)
ansii -img photo.png -imgw 80 -f art.ansii

# Import an image as ASCII art (luminance ramp + colors)
ansii -ascii photo.jpg -imgw 80 -f art.ansii

# Display a .ansi file in the terminal and exit
ansii -show ~/.config/ansii/splash.ansi

# Re-install the current splash to shell RC
ansii -install                    # reads from ~/.config/ansii/splash.ansi
ansii -install -f my_art.ansii    # reads from the given file
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-f <file>` | `.ansii` project file to open or create | *(none)* |
| `-w <n>` | Canvas width in columns (new file only) | `60` |
| `-h <n>` | Canvas height in rows (new file only) | `30` |
| `-import <file>` | Import a `.txt` file as canvas art | — |
| `-img <image>` | Import PNG/JPG as half-block ANSI art | — |
| `-ascii <image>` | Import PNG/JPG as ASCII art | — |
| `-imgw <n>` | Target width (columns) for image import | `80` |
| `-show <file>` | Print a `.ansi` file to the terminal and exit | — |
| `-install` | Re-inject splash block into `.bashrc`/`.zshrc` | — |

---

## Controls

### Navigation

| Key | Action |
|-----|--------|
| `↑ ↓ ← →` / `h j k l` | Move cursor |
| `?` | Open help overlay |
| `q` / `Ctrl+C` | Quit |

### Drawing

| Key | Action |
|-----|--------|
| Any printable key | Draw that character and advance (Draw tool only) |
| `Space` / `Enter` | Apply the active tool at cursor |
| `Backspace` | Erase cell and move left |
| `Delete` | Erase cell in place |

### Tools

| Key | Tool | Description |
|-----|------|-------------|
| `d` | Draw | Place current character + colors |
| `e` | Erase | Clear a cell to blank |
| `f` | Fill | Flood-fill the connected region |
| `t` | Text mode | Type freely — `Esc` to exit (Draw tool only) |

### Text mode (`t`)

All printable keys draw directly, including `d`, `e`, `f`, `s`, etc.

| Key | Action |
|-----|--------|
| Any printable key | Draw and advance cursor |
| `← → ↑ ↓` | Move cursor |
| `Enter` | Move down one line, return to the starting column |
| `Backspace` | Erase and move left |
| `Ctrl+S` | Open save prompt |
| `Esc` | Exit text mode |

### Colors

| Key | Action |
|-----|--------|
| `c` | Set foreground color — type `0–255` or `d` for default, then `Enter` |
| `b` | Set background color — type `0–255` or `d` for default, then `Enter` |

### Character palette

| Key | Action |
|-----|--------|
| `Shift+←` | Previous character in palette |
| `Shift+→` | Next character in palette |

### File & Import

| Key | Action |
|-----|--------|
| `s` / `Ctrl+S` | Save — prompts for filename. `.ansii` = project JSON, `.ansi` = raw export |
| `i` | Install as terminal splash screen — shows confirmation before writing |
| `r` | Import text file — opens path prompt (Tab to autocomplete) |
| `g` | Import image as half-block ANSI art — opens path prompt |
| `a` | Import image as ASCII art — opens path prompt |

In any file prompt, `Tab` / `Shift+Tab` cycles through filesystem completions starting from `~/`. Directories end with `/`.

---

## Splash Screen

Full workflow to display your art on every terminal open:

```
1. Draw or import your art in the editor
2. Press [i] — exports and injects into .bashrc / .zshrc
3. Open a new terminal
```

The editor writes the art to `~/.config/ansii/splash.ansi` and adds this block:

```bash
# ansii-splash
cat "/home/user/.config/ansii/splash.ansi"
```

To uninstall, remove those two lines from your RC file. To update the splash with new art, simply press `[i]` again — the block is replaced in place.

---

## Image Import Methods

### Half-block (`g` / `-img`)

Each terminal cell encodes two pixel rows using `▀` (upper) and `▄` (lower) as the character, with the pixel colors mapped to foreground and background. Produces smooth, photo-realistic results. Recommended for splash screens.

### ASCII (`a` / `-ascii`)

Maps each source region's luminance to a character in the ramp `$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\|()1{}[]?-_+~<>i!lI;:,"^'. ` (dark→dense, light→sparse) and assigns the nearest ANSI-256 color. The result looks like classic ASCII art and is easy to edit by hand afterward. No external tools required.

---

## File Format (`.ansii`)

Projects are saved as human-readable JSON. Only non-blank cells are stored:

```json
{
  "version": 1,
  "width": 60,
  "height": 30,
  "cells": [
    { "x": 10, "y": 5, "char": "█", "fg": 11, "bg": -1 },
    { "x": 11, "y": 5, "char": "▓", "fg": 3,  "bg": 0  }
  ]
}
```

| Field | Values |
|-------|--------|
| `char` | Any Unicode character |
| `fg` / `bg` | `-1` = terminal default · `0–7` = standard · `8–15` = bright · `16–255` = 256-color |

---

## Project Structure

```
ansii/
├── README.md
└── editor/
    ├── main.go      — CLI flags and entry point
    ├── model.go     — Bubbletea model, types, constants, palette data
    ├── update.go    — All key handling and tool logic
    ├── view.go      — TUI rendering (canvas, sidebar, status bar, help)
    ├── canvas.go    — Canvas type, flood fill, color helpers
    ├── export.go    — JSON save/load, ANSI export, text import, shell install
    └── image.go     — Image import: half-block (▀/▄) and ASCII ramp
```

---

## Tech Stack

| Library | Purpose |
|---------|---------|
| [Bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm architecture) |
| [Lipgloss](https://github.com/charmbracelet/lipgloss) | Styles, colors, and layout |
