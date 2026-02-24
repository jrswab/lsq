package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/system"
	"github.com/jrswab/lsq/trie"
)

const semVer string = "1.4.0"

// Search regex pattern in given file, print matching lines
func searchInFile(filePath string, pattern *regexp.Regexp) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if pattern.MatchString(line) {
			fmt.Printf("%s#%d: %s\n", filePath, lineNumber, line)
		}
	}
	return scanner.Err()
}

// Search regex pattern in all files within directory
func searchInDirectory(directory string, pattern *regexp.Regexp) error {
	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			err := searchInFile(path, pattern)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func main() {
	// File Path Override
	lsqDirPath := flag.String("d", "", "The path to the Logseq directory to use.")

	apndStdin := flag.Bool("A", false, "Append STDIN to the current journal page. This will not open $EDITOR.")
	apnd := flag.String("a", "", "Append text to the current journal page. This will not open $EDITOR.")
	catFile := flag.Bool("c", false, "Print journal or page content to STDOUT instead of opening an editor.")
	editorType := flag.String("e", "", "The external editor to use. Will use $EDITOR when blank or omitted.")
	cliSearch := flag.String("f", "", "Search by file name in your pages directory.")
	indent := flag.Int("i", 0, "Absolute indentation level (number of tab characters) for appended text. Requires -a or -A.")
	daysAgo := flag.Int("n", 0, "Number of days ago to target for the journal entry.")
	openFirstResult := flag.Bool("o", false, "Open the first result from search automatically.")
	pageToOpen := flag.String("p", "", "Open a specific page from the pages directory. Must be a file name with extension.")
	regexSearch := flag.String("r", "", "Search by regex pattern in pages directory.")
	specDate := flag.String("s", "", "Open a specific journal. Use yyyy-MM-dd after the flag.")
	version := flag.Bool("v", false, "Display current lsq version")
	yesterday := flag.Bool("y", false, "Open yesterday's journal page")

	flag.Parse()

	if *version {
		fmt.Println(semVer)
		os.Exit(0)
	}

	if *daysAgo < 0 {
		fmt.Fprintln(os.Stderr, "Error: -n must be a non-negative integer")
		os.Exit(1)
	}

	if *indent < 0 {
		fmt.Fprintln(os.Stderr, "Error: indentation level must be zero or greater")
		os.Exit(1)
	}

	if *indent > 0 && *apnd == "" && !*apndStdin {
		fmt.Fprintln(os.Stderr, "Error: -i requires -a or -A")
		os.Exit(1)
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
		expanded, expandErr := config.ExpandPath(*lsqDirPath)
		if expandErr != nil {
			log.Printf("Error expanding directory path: %v\n", expandErr)
			os.Exit(1)
		}
		cfg.DirPath = expanded
		cfg.JournalsDir = filepath.Join(cfg.DirPath, "journals")
		cfg.PagesDir = filepath.Join(cfg.DirPath, "pages")
	}

	if *pageToOpen != "" {
		pagePath := filepath.Join(cfg.PagesDir, *pageToOpen)

		// Print page content to STDOUT and exit.
		if *catFile {
			if err := system.PrintFile(pagePath); err != nil {
				log.Printf("Error printing page: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Append to page and exit.
		if *apnd != "" {
			err := system.AppendToFile(pagePath, *apnd, *indent)
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

	// Init regex search if "-r"
	if *regexSearch != "" {
		pattern, err := regexp.Compile(*regexSearch)
		if err != nil {
			log.Printf("Error compiling regex pattern: %v\n", err)
			os.Exit(1)
		}

		// Search in the directories
		for _, searchDirectory := range []string{cfg.JournalsDir, cfg.PagesDir} {
			err = searchInDirectory(searchDirectory, pattern)
			if err != nil {
				fmt.Println("Error searching directory:", err)
			}
		}

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

	// When -n is provided, compute the date N days ago.
	if *daysAgo > 0 {
		*specDate = time.Now().AddDate(0, 0, -*daysAgo).Format("2006-01-02")
	}

	journalPath, err := system.GetJournal(cfg, cfg.JournalsDir, *specDate)
	if err != nil {
		log.Printf("Error setting journal path: %v\n", err)
		os.Exit(1)
	}

	// Print journal content to STDOUT and exit.
	if *catFile {
		if err := system.PrintFile(journalPath); err != nil {
			log.Printf("Error printing journal: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *apnd != "" {
		err := system.AppendToFile(journalPath, *apnd, *indent)
		if err != nil {
			log.Printf("Error appending data to file: %v\n", err)
			os.Exit(1)
		}

		// Don't open $EDITOR  when append flag is used.
		return
	}

	system.LoadEditor(*editorType, journalPath)
}
