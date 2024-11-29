package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/editor"
	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Define command line flags
	useTUI := flag.Bool("t", false, "Use the custom TUI instead of directly opening the system editor")
	lsqDirName := flag.String("d", "Logseq", "The main Logseq directory to use.")
	lsqCfgDirName := flag.String("l", "logseq", "The Logseq configuration directory to use.")
	lsqCfgFileName := flag.String("c", "config.edn", "The config.edn file to use.")
	editorType := flag.String("e", "EDITOR", "The editor to use.")

	// Parse flags
	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// Construct paths
	lsqDir := filepath.Join(homeDir, *lsqDirName)
	lsqCfgDir := filepath.Join(lsqDir, *lsqCfgDirName)
	cfgFile := filepath.Join(lsqCfgDir, *lsqCfgFileName)

	cfg, err := system.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Construct journals directory path
	journalsDir := filepath.Join(lsqDir, "journals")

	// Create journals directory if it doesn't exist
	err = os.MkdirAll(journalsDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating journals directory: %v\n", err)
		os.Exit(1)
	}

	journalPath, err := system.GetTodaysJournal(cfg, journalsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting journal path: %v\n", err)
		os.Exit(1)
	}

	// After the file exists, branch based on mode
	if *useTUI {
		loadTui(cfg, journalPath)
	} else {
		loadEditor(*editorType, journalPath)
	}

	os.Exit(0)
}

func loadTui(cfg *config.Config, path string) {
	p := tea.NewProgram(
		tui.InitialModel(cfg, path),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func loadEditor(program, path string) {
	// Get editor from environment
	editing := editor.Select(program)

	// Open file in editor
	cmd := exec.Command(editing, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening editor: %v\n", err)
		os.Exit(1)
	}
}
