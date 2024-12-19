package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/trie"
	"github.com/jrswab/lsq/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	// Define command line flags
	apnd := flag.String("a", "", "Append text to the current journal page. This will not open $EDITOR or the TUI.")
	lsqCfgFileName := flag.String("c", "config.edn", "The config.edn file to use.")
	lsqDirName := flag.String("d", "Logseq", "The main Logseq directory to use.")
	editorType := flag.String("e", "", "The external editor to use. Will use $EDITOR when blank or omitted.")
	cliSearch := flag.String("f", "", "Search the logseq graph without the TUI")
	lsqCfgDirName := flag.String("l", "logseq", "The Logseq configuration directory to use.")
	openFirstResult := flag.Bool("o", false, "Open the first result from search automatically.")
	specDate := flag.String("s", "", "Open a specific journal. Use yyyy-MM-dd after the flag.")
	useTUI := flag.Bool("t", false, "Use the custom TUI instead of directly opening the system editor")

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
	pagesPath := filepath.Join(lsqDir, "pages")

	// Init Search only when TUI or -f is passed
	var searchTrie *trie.Trie
	if *useTUI || !strings.EqualFold(*cliSearch, "") {
		searchTrie, err = trie.Init(pagesPath)
		if err != nil {
			log.Printf("error loading pages directory for search: %v\n", err)
			os.Exit(1)
		}
	}

	if !strings.EqualFold(*cliSearch, ""){
		results := searchTrie.Search(*cliSearch)
		if len(results) < 1 {
			fmt.Println("No results found")
			return
		}
		
		if *openFirstResult {
			loadEditor(*editorType, fmt.Sprintf("%s/%s", pagesPath, results[0]))
			return
		}

		fmt.Fprintf(os.Stdout, "Search Results:\n")
		for _, val := range results {
			fmt.Println(val)
		}

		return
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

	if *apnd != "" {
		path := system.CreateFilePath(cfg, journalsDir, date)
		err := system.AppendToFile(path, *apnd)
		if err != nil {
			log.Printf("Error appending data to file: %v\n", err)
			os.Exit(1)
		}

		// Don't open $EDITOR or TUI when append flag is used.
		return
	}

	journalPath, err := system.GetJournal(cfg, journalsDir, date)
	if err != nil {
		log.Printf("Error setting journal path: %v\n", err)
		os.Exit(1)
	}

	// After the file exists, branch based on mode
	if *useTUI {
		loadTui(cfg, journalPath, searchTrie)
	} else {
		loadEditor(*editorType, journalPath)
	}
}

func loadTui(cfg *config.Config, path string, t *trie.Trie) {
	p := tea.NewProgram(
		tui.InitialModel(cfg, path, t),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func loadEditor(editor, path string) {
	// Get editor from environment
	if editor == "" {
		editor = os.Getenv("EDITOR")
		log.Println(editor)
	}

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

