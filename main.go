package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/trie"
)

const semVer string = "0.11.0"

func main() {
	// File Path Override
	lsqDirPath := flag.String("d", "", "The path to the Logseq directory to use.")

	apnd := flag.String("a", "", "Append text to the current journal page. This will not open $EDITOR or the TUI.")
	editorType := flag.String("e", "", "The external editor to use. Will use $EDITOR when blank or omitted.")
	cliSearch := flag.String("f", "", "Search the logseq graph without the TUI")
	openFirstResult := flag.Bool("o", false, "Open the first result from search automatically.")
	pageToOpen := flag.String("p", "", "Open a specific page from the pages directory. Must be a file name with extention.")
	specDate := flag.String("s", "", "Open a specific journal. Use yyyy-MM-dd after the flag.")
	useTUI := flag.Bool("t", false, "Use the custom TUI instead of directly opening the system editor")
	version := flag.Bool("v", false, "Display current lsq version")
	yesterday := flag.Bool("y", false, "Open yesterday's journal page")

	flag.Parse()

	if *version {
		fmt.Println(semVer)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil && !os.IsNotExist(err) {
		// The user has a config file but we couldn't read it.
		// Report the error instead of ignoring their configuration.
		log.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// When this flag is used override the config.
	if *lsqDirPath != "" {
		cfg.DirPath = *lsqDirPath
		cfg.JournalsDir = filepath.Join(cfg.DirPath, "journals")
		cfg.PagesDir = filepath.Join(cfg.DirPath, "pages")
	}

	// Open page in default editor if specified:
	if *pageToOpen != "" {
		system.LoadEditor(*editorType, fmt.Sprintf("%s/%s", cfg.PagesDir, *pageToOpen))
		return
	}

	// Init Search only when "-f" is passed
	var searchTrie *trie.Trie
	if *useTUI || !strings.EqualFold(*cliSearch, "") {
		searchTrie, err = trie.Init(cfg.PagesDir)
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
			system.LoadEditor(*editorType, fmt.Sprintf("%s/%s", cfg.PagesDir, results[0]))
			return
		}

		fmt.Fprintf(os.Stdout, "Search Results:\n")
		for _, val := range results {
			fmt.Println(val)
		}

		return
	}

	// Check that the journals directory exists
	_, err = os.Stat(cfg.DirPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Could not find Logseq files at '%s'.\n", cfg.DirPath)
			fmt.Printf("Make sure the path is correct and the directories exist./n")
			os.Exit(0)
		}

		log.Printf("Error loading the main directory: %v\n", err)
		os.Exit(1)
	}

	if *yesterday {
		*specDate = time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	}

	journalPath, err := system.GetJournal(cfg, cfg.JournalsDir, *specDate)
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
