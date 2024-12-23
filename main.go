package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/trie"
)

func main() {

	// Define command line flags
	apnd := flag.String("a", "", "Append text to the current journal page. This will not open $EDITOR or the TUI.")
	lsqCfgFileName := flag.String("c", "config.edn", "The config.edn file to use.")
	lsqDirPath := flag.String("d", "", "The path to the Logseq directory to use.")
	editorType := flag.String("e", "", "The external editor to use. Will use $EDITOR when blank or omitted.")
	cliSearch := flag.String("f", "", "Search the logseq graph without the TUI")
	lsqCfgDirName := flag.String("l", "logseq", "The Logseq configuration directory to use.")
	openFirstResult := flag.Bool("o", false, "Open the first result from search automatically.")
	pageToOpen := flag.String("p", "", "Open a specific page from the pages directory. Must be a file name with extention.")
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
	lsqPath := filepath.Join(homeDir, "Logseq")

	// When this flag is used replace path
	if *lsqDirPath != "" {
		lsqPath = *lsqDirPath
	}

	lsqCfgDir := filepath.Join(lsqPath, *lsqCfgDirName)
	cfgFile := filepath.Join(lsqCfgDir, *lsqCfgFileName)
	journalsDir := filepath.Join(lsqPath, "journals")
	pagesDir := filepath.Join(lsqPath, "pages")

	cfg, err := system.LoadConfig(cfgFile)
	if err != nil {
		log.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

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


