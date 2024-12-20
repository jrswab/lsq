package system

import (
	"log"
	"os"
	"os/exec"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/tui"
	"github.com/jrswab/lsq/trie"

	tea "github.com/charmbracelet/bubbletea"
)


func LoadTui(cfg *config.Config, path string, t *trie.Trie) {
	p := tea.NewProgram(
		tui.InitialModel(cfg, path, t),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func LoadEditor(editor, path string) {
	// Get editor from environment
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	// if still blank, use nano
	if editor == "" {
		log.Println("$EDITOR is blank, using Nano.")
		editor = "vim"
	}

	// Open file in editor
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("Error opening editor: %v\n", err)
		os.Exit(1)
	}
}
