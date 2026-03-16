package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var (
		filename   = flag.String("f", "", "open or create a .ansii project file")
		canvasW    = flag.Int("w", 60, "canvas width (new file)")
		canvasH    = flag.Int("h", 30, "canvas height (new file)")
		importFile = flag.String("import", "", "import a plain text/braille .txt file as canvas art")
		importImg  = flag.String("img", "", "import an image (PNG/JPG) as colored ANSI art")
		imgWidth   = flag.Int("imgw", 80, "target canvas width when importing an image")
		show       = flag.String("show", "", "display an .ansi file and exit (for shell splash)")
		install    = flag.Bool("install", false, "re-install current splash to shell RC")
	)
	flag.Parse()

	// Show mode: just cat the ANSI file (used from .bashrc)
	if *show != "" {
		data, err := os.ReadFile(*show)
		if err != nil {
			// Silently fail in shell context
			os.Exit(0)
		}
		fmt.Print(string(data))
		return
	}

	// Install mode: install a given file (or default splash)
	if *install {
		path := *filename
		if path == "" {
			path = os.Getenv("HOME") + "/.config/ansii/splash.ansi"
		}
		if err := installToShell(path); err != nil {
			log.Fatalf("install error: %v", err)
		}
		fmt.Println("Installed! Restart your terminal to see the splash art.")
		return
	}

	// Editor mode
	m := newModel(*canvasW, *canvasH)

	if *filename != "" {
		m.filename = *filename
		c, err := loadCanvas(*filename)
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("could not load %s: %v", *filename, err)
		}
		if err == nil {
			m.canvas = c
		}
	}

	// Import a plain text file directly (overrides canvas)
	if *importFile != "" {
		c, err := importFromText(*importFile)
		if err != nil {
			log.Fatalf("could not import %s: %v", *importFile, err)
		}
		m.canvas = c
		m.statusMsg = fmt.Sprintf("Imported '%s' (%dx%d) — add colors and press [s] to save", *importFile, c.Width, c.Height)
	}

	// Import image as colored ANSI art
	if *importImg != "" {
		c, err := importFromImage(*importImg, *imgWidth)
		if err != nil {
			log.Fatalf("could not import image %s: %v", *importImg, err)
		}
		m.canvas = c
		m.statusMsg = fmt.Sprintf("Image imported (%dx%d) — press [s] to save, [i] to install", c.Width, c.Height)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
