# ANSII — ANSI Art Editor

```
  █████╗ ███╗   ██╗███████╗██╗██╗
 ██╔══██╗████╗  ██║██╔════╝██║██║
 ███████║██╔██╗ ██║███████╗██║██║
 ██╔══██║██║╚██╗██║╚════██║██║██║
 ██║  ██║██║ ╚████║███████║██║██║
 ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚═╝╚═╝
     ANSI Art Editor for Linux
```

A terminal-based ANSI art editor written in Go with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss). Draw anime characters, game sprites, or anything you like using Unicode block characters and 256-color ANSI, then set it as your terminal splash screen.

---

## Features

- **Editable canvas** with navigable cursor and scroll
- **256 ANSI colors** for foreground and background (compatible with all modern terminals)
- **Character palette** with Unicode blocks, box-drawing, braille, and symbols
- **Tools**: Draw, Erase, Flood Fill, Eyedropper
- **Import text art** — load any `.txt` file with ASCII/braille art (like the Gengar demo)
- **Import PNG/JPG** — converts to ANSI art with colors using the half-block (`▀▄`) technique; transparent backgrounds are automatically removed
- **Save/Load** in `.ansii` format (human-readable JSON)
- **Export** to `.ansi` (raw escape sequences, compatible with any terminal)
- **Install splash screen** — adds the art to `.bashrc`/`.zshrc` with a single shortcut

---

## Installation

### Pre-built binary *(recommended)*

Download from the [Releases](../../releases/latest) page — no Go installation needed.

**Linux (amd64):**
```bash
curl -L https://github.com/YOUR_USERNAME/ansii/releases/latest/download/ansii-linux-amd64 -o ansii
chmod +x ansii
sudo mv ansii /usr/local/bin/
```

**Linux ARM64 (Raspberry Pi, etc.):**
```bash
curl -L https://github.com/YOUR_USERNAME/ansii/releases/latest/download/ansii-linux-arm64 -o ansii
chmod +x ansii && sudo mv ansii /usr/local/bin/
```

**macOS Apple Silicon (M1/M2/M3):**
```bash
curl -L https://github.com/YOUR_USERNAME/ansii/releases/latest/download/ansii-darwin-arm64 -o ansii
chmod +x ansii && sudo mv ansii /usr/local/bin/
```

**macOS Intel:**
```bash
curl -L https://github.com/YOUR_USERNAME/ansii/releases/latest/download/ansii-darwin-amd64 -o ansii
chmod +x ansii && sudo mv ansii /usr/local/bin/
```

### Build from source

Requires Go 1.21+:

```bash
git clone <repo>
cd ansii/editor
go build -o ansii-editor .
sudo mv ansii-editor /usr/local/bin/ansii
```

---

## Usage

```bash
cd editor/

# New canvas (60x30 by default)
./ansii-editor

# Open or create a file
./ansii-editor -f my_art.ansii

# Specify canvas size
./ansii-editor -f hero.ansii -w 80 -h 40

# Show a .ansi file directly in terminal
./ansii-editor -show ../art/gengar.ansi
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-f <file>` | `.ansii` file to open or create | *(empty)* |
| `-w <n>` | Canvas width (columns) | `60` |
| `-h <n>` | Canvas height (rows) | `30` |
| `-import <file.txt>` | Import text/braille art as canvas | — |
| `-img <image.png>` | Import PNG/JPG as colored ANSI art | — |
| `-imgw <n>` | Width in columns when importing an image | `80` |
| `-show <file>` | Display a `.ansi` file in terminal and exit | — |
| `-install` | Reinstall splash screen in shell RC | — |

---

## Controls

### Canvas Panel *(default focus)*

| Key | Action |
|-----|--------|
| `↑ ↓ ← →` or `h j k l` | Move cursor |
| Type any character | Draw that character and advance |
| `Space` / `Enter` | Apply the active tool |
| `Backspace` | Erase cell and move left |
| `Delete` | Erase cell in place |
| `Tab` | Switch to Color panel |
| `Shift+Tab` | Switch to Character panel |

### Tools *(activate with key in Canvas panel)*

| Key | Tool | Description |
|-----|------|-------------|
| `d` | **Draw** | Draw current character/color |
| `e` | **Erase** | Clear cell |
| `f` | **Fill** | Flood fill |
| `p` | **Eyedropper** | Pick character and colors from cell |
| `t` | **Text mode** | Free-typing — all keys draw |

### Text Mode (`t`)

Press `t` to enter free-typing mode. Every key prints its character directly
(including `d`, `e`, `f`, `s`, …). Ideal for adding text, names, or
dialogue to your art.

| Key in text mode | Action |
|------------------|--------|
| Any printable key | Draw and advance |
| `← → ↑ ↓` | Move cursor |
| `Enter` | Move down one line, return to starting column |
| `Backspace` | Erase and move left |
| `Delete` | Erase in place |
| `Ctrl+S` | Save without leaving text mode |
| `Esc` / `q` | **Exit text mode** |

### Color Panel *(Tab from Canvas)*

| Key | Action |
|-----|--------|
| `↑ ↓ ← →` | Navigate palette (8×2 colors) |
| `f` | Select color as **Foreground** |
| `b` | Select color as **Background** |
| `d` | Set **Default** (no color) |
| `Enter` / `Escape` | Apply and return to canvas |
| `Tab` | Go to Character panel |

### Character Panel *(Tab from Colors)*

| Key | Action |
|-----|--------|
| `↑ ↓ ← →` | Navigate character palette |
| `Enter` / `Escape` | Select and return to canvas |
| `Tab` | Return to canvas |

### File & Import

| Key | Action |
|-----|--------|
| `s` / `Ctrl+S` | Save in `.ansii` format |
| `x` | Export as `.ansi` (raw ANSI escape codes) |
| `i` | Install to `.bashrc`/`.zshrc` as splash screen |
| `r` | **Import text/braille file** (type path, Enter) |
| `g` | **Import PNG/JPG** as colored ANSI art |
| `?` | Show help |
| `q` | Quit |

---

## Splash Screen

The full workflow to show your art every time you open a terminal:

```
1. Draw your character in the editor
2. Press [i] inside the editor
3. Open a new terminal → 🎮
```

When you press `i`, the editor:
1. Exports the art to `~/.config/ansii/splash.ansi`
2. Automatically adds the following line to your `.bashrc` and/or `.zshrc`:

```bash
# ansii-splash
cat "/home/user/.config/ansii/splash.ansi"
```

To uninstall, remove those two lines from your RC file.

---

## Character Palette

```
Blocks   █ ▓ ▒ ░ ▄ ▀ ▌ ▐ ■ □ ▪ ▫
Box      ─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼ ═ ║
Double   ╔ ╗ ╚ ╝ ╠ ╣ ╦ ╩ ╬ ▲ ▼ ◄ ►
Symbols  ● ○ ★ ☆ ♥ ♦ ♣ ♠ ◆ ◇ ♪ ♫ ☺
ASCII    / \ | - _ = + * # @ ! ? ~
```

---

## Color Palette

The editor supports all **256 ANSI colors**:

```
  0-7   : Standard colors (Black, Red, Green, Yellow, Blue, Magenta, Cyan, White)
  8-15  : Bright variants of the above
  16-231: 6×6×6 color cube
  232-255: Grayscale ramp (24 shades)
```

Colors 16-255 are used automatically when importing images.
The editor's color panel shows the 16 standard colors for manual editing.

---

## File Format (`.ansii`)

Projects are saved as readable JSON. Only non-empty cells are stored:

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
| `fg` / `bg` | `-1` = terminal default, `0–255` = ANSI color index |

---

## Project Structure

```
ansii/
├── README.md
├── .gitignore
├── art/                    ← Art files (txt, ansi, images)
│   ├── gengar.txt          ← Braille Gengar art
│   ├── agumon.txt
│   └── SorenPortrait_FE9.png
└── editor/                 ← Editor source code
    ├── go.mod
    ├── go.sum
    ├── main.go             ← CLI entry point and flags
    ├── model.go            ← Bubbletea model, types and constants
    ├── update.go           ← Key handling and tool logic
    ├── view.go             ← TUI rendering
    ├── canvas.go           ← 2D canvas, 256-color support, flood fill
    ├── export.go           ← Save/load JSON, import text, export ANSI
    └── image.go            ← Import PNG/JPG as ANSI art
```

---

## Examples

### Import a text/braille art file

```bash
cd editor/

# From the command line
./ansii-editor -import /path/to/my-art.txt -f my-art.ansii

# Or from inside the editor: press [r], type the path, Enter
```

### Import a PNG/JPG image

```bash
cd editor/
./ansii-editor -img /path/to/my-art.png -imgw 60 -f my-art.ansii
```

> The algorithm uses the `▀`/`▄` half-block technique: each terminal row
> represents 2 pixel rows using foreground and background colors.
> Transparent pixels (PNG alpha) become empty cells automatically.

---

## Tech Stack

| Library | Purpose |
|---------|---------|
| [Bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm architecture) |
| [Lipgloss](https://github.com/charmbracelet/lipgloss) | Styles, colors, and layout |

---

## License

MIT
