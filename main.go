package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/trie"
)

func main() {
	// File Path Overrides
	lsqDirPath := flag.String("d", "", "The path to the Logseq directory to use.")
	appCfgFileName := flag.String("c", "config.edn", "The config.edn file to use.")
	appCfgDirName := flag.String("l", "logseq", "The Logseq configuration directory to use.")

	apnd := flag.String("a", "", "Append text to the current journal page. This will not open $EDITOR or the TUI.")
	editorType := flag.String("e", "", "The external editor to use. Will use $EDITOR when blank or omitted.")
	cliSearch := flag.String("f", "", "Search the logseq graph without the TUI")
	openFirstResult := flag.Bool("o", false, "Open the first result from search automatically.")
	pageToOpen := flag.String("p", "", "Open a specific page from the pages directory. Must be a file name with extention.")
	specDate := flag.String("s", "", "Open a specific journal. Use yyyy-MM-dd after the flag.")
	useTUI := flag.Bool("t", false, "Use the custom TUI instead of directly opening the system editor")

	flag.Parse()

	// Check for config file
	// if file exists load it
	// if file !exists load logseq config
	// set flags as overrides after

	// Consturct file paths:
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// Default path:
	dirPath := filepath.Join(homeDir, "Logseq")

	// When this flag is used replace path
	if *lsqDirPath != "" {
		dirPath = *lsqDirPath
	}

	// Load Config:
	//cfg := &config.Config{
	//	AppCfgDir:  *appCfgDirName,
	//	AppCfgName: *appCfgFileName,
	//}

	cfg, err := config.Load()
	if err != nil && !os.IsNotExist(err) {
		// The user has a config file but we couldn't read it.
		// Report the error instead of ignoring their configuration.
		log.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// When this flag is used override the config.
	dirPath = cfg.DirPath
	if *lsqDirPath != "" {
		dirPath = *lsqDirPath
	}

	// Load the logseq specific config when the lsq config is not present.
	if os.IsNotExist(err) {
		cfg, err = config.LoadAppConfig(dirPath,
			filepath.Join(dirPath, fmt.Sprintf("%s/%s", *appCfgDirName, *appCfgFileName)),
		)

		if err != nil && !os.IsNotExist(err) {
			log.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}
	}

	journalsDir := filepath.Join(dirPath, "journals")
	pagesDir := filepath.Join(dirPath, "pages")

	// Open page in default editor if specified:
	if *pageToOpen != "" {
		// TUI can use search feature to find a page to open
		system.LoadEditor(*editorType, fmt.Sprintf("%s/%s", pagesDir, *pageToOpen))
		return
	}

	// Init Search only when TUI or -f is passed
	var searchTrie *trie.Trie
	if *useTUI || !strings.EqualFold(*cliSearch, "") {
		searchTrie, err = trie.Init(pagesDir)
		if err != nil {
			log.Printf("error loading pages directory for search: %v\n", err)
			os.Exit(1)
		}
	}

	if *cliSearch != "" {
		results := searchTrie.Search(*cliSearch)
		if len(results) < 1 {
			fmt.Println("No results found")
			return
		}

		if *openFirstResult {
			system.LoadEditor(*editorType, fmt.Sprintf("%s/%s", pagesDir, results[0]))
			return
		}

		fmt.Fprintf(os.Stdout, "Search Results:\n")
		for _, val := range results {
			fmt.Println(val)
		}

		return
	}

	// Create journals directory if it doesn't exist
	err = os.MkdirAll(journalsDir, 0755)
	if err != nil {
		log.Printf("Error creating journals directory: %v\n", err)
		os.Exit(1)
	}

	journalPath, err := system.GetJournal(cfg, journalsDir, *specDate)
	if err != nil {
		log.Printf("Error setting journal path: %v\n", err)
		os.Exit(1)
	}

	if *apnd != "" {
		err := system.AppendToFile(journalPath, *apnd)
		if err != nil {
			log.Printf("Error appending data to file: %v\n", err)
			os.Exit(1)
		}

		// Don't open $EDITOR or TUI when append flag is used.
		return
	}

	// After the file exists, branch based on mode
	if *useTUI {
		system.LoadTui(cfg, journalPath, searchTrie)
		return
	}

	system.LoadEditor(*editorType, journalPath)
}
