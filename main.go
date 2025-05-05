package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/trie"
)

const semVer string = "1.0.0"

func main() {
	// File Path Override
	lsqDirPath := flag.String("d", "", "The path to the Logseq directory to use.")

	apndStdin := flag.Bool("A", false, "Append STDIN to the current journal page. This will not open $EDITOR.")
	apnd := flag.String("a", "", "Append text to the current journal page. This will not open $EDITOR.")
	editorType := flag.String("e", "", "The external editor to use. Will use $EDITOR when blank or omitted.")
	cliSearch := flag.String("f", "", "Search by file name in your pages directory.")
	openFirstResult := flag.Bool("o", false, "Open the first result from search automatically.")
	pageToOpen := flag.String("p", "", "Open a specific page from the pages directory. Must be a file name with extension.")
	specDate := flag.String("s", "", "Open a specific journal. Use yyyy-MM-dd after the flag.")
	version := flag.Bool("v", false, "Display current lsq version")
	yesterday := flag.Bool("y", false, "Open yesterday's journal page")

	flag.Parse()

	if *version {
		fmt.Println(semVer)
		os.Exit(0)
	}

	if *apndStdin {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading STDIN: %v\n", err)
			os.Exit(1)
		}
		*apnd = string(content)
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

	if *pageToOpen != "" {
		pagePath := filepath.Join(cfg.PagesDir, *pageToOpen)

		// Append to page and exit.
		if *apnd != "" {
			err := system.AppendToFile(pagePath, *apnd)
			if err != nil {
				log.Printf("Error appending data to file: %v\n", err)
				os.Exit(1)
			}
			// Don't open $EDITOR when append flag is used.
			return
		}

		// Open page in default editor if specified:
		system.LoadEditor(*editorType, pagePath)
		return
	}

	// Init Search only when "-f" is passed
	var searchTrie *trie.Trie
	if !strings.EqualFold(*cliSearch, "") {
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
			fmt.Printf("Could not find Logseq files at '%s'.\nMake sure the path is correct and the directories exist.\n", cfg.DirPath)
			os.Exit(1)
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

		// Don't open $EDITOR  when append flag is used.
		return
	}

	system.LoadEditor(*editorType, journalPath)
}
