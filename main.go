package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/trie"
	"github.com/jrswab/lsq/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func initTrie(path string) (*trie.Trie, error) {
	var tree = *trie.NewTrie()

	// get list of all files in ~/Logseq/Pages
	fileList, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for i := range fileList {
		if !fileList[i].IsDir() {
			tree.Insert(fileList[i].Name())
		}
	}

	return &tree, nil
}

func main() {

	// Define command line flags
	useTUI := flag.Bool("t", false, "Use the custom TUI instead of directly opening the system editor")
	lsqDirName := flag.String("d", "Logseq", "The main Logseq directory to use.")
	lsqCfgDirName := flag.String("l", "logseq", "The Logseq configuration directory to use.")
	lsqCfgFileName := flag.String("c", "config.edn", "The config.edn file to use.")
	editorType := flag.String("e", "EDITOR", "The editor to use.")
	specDate := flag.String("s", "", "Open a specific journal. Use yyyy-MM-dd after the flag.")

	// Parse flags
	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// Construct paths
	lsqDir := filepath.Join(homeDir, *lsqDirName)
	lsqCfgDir := filepath.Join(lsqDir, *lsqCfgDirName)
	cfgFile := filepath.Join(lsqCfgDir, *lsqCfgFileName)

	// Init Search
	_, err = initTrie(filepath.Join(lsqDir, "pages"))
	if err != nil {
		log.Printf("error loading pages directory for search: %v\n", err)
		os.Exit(1)
	}

	cfg, err := system.LoadConfig(cfgFile)
	if err != nil {
		log.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Construct journals directory path
	journalsDir := filepath.Join(lsqDir, "journals")

	// Create journals directory if it doesn't exist
	err = os.MkdirAll(journalsDir, 0755)
	if err != nil {
		log.Printf("Error creating journals directory: %v\n", err)
		os.Exit(1)
	}

	var date = time.Now().Format(config.ConvertDateFormat(cfg.FileNameFmt))
	if *specDate != "" {
		parsedDate, err := time.Parse("2006-01-02", *specDate)
		if err != nil {
			log.Printf("Error parsing date from -s flag: %v\n", err)
			os.Exit(1)
		}

		// Return date formatted to user configuration.
		date = parsedDate.Format(config.ConvertDateFormat(cfg.FileNameFmt))
	}

	journalPath, err := system.GetJournal(cfg, journalsDir, date)
	if err != nil {
		log.Printf("Error setting journal path: %v\n", err)
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
		log.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func loadEditor(editor, path string) {
	// Get editor from environment
	editor = os.Getenv(editor)
	// if still blank, use nano
	if editor == "" {
		log.Println("$EDITOR is blank, using Nano.")
		editor = "nano"
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
