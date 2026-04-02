package system

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jrswab/lsq/config"
)

func CreateFilePath(cfg *config.Config, journalsDir, date string) string {
	// Construct today's journal file path
	extension := ".md"
	if strings.EqualFold(cfg.FileType, "Org") {
		extension = ".org"
	}

	// Create the path for the specified date.
	return filepath.Join(journalsDir, fmt.Sprintf("%s%s", date, extension))
}

func GetJournal(cfg *config.Config, journalsDir, specDate string) (string, error) {
	date := time.Now().Format(config.ConvertDateFormat(cfg.FileFmt))

	if specDate != "" {
		parsedDate, err := time.Parse("2006-01-02", specDate)
		if err != nil {
			return "", fmt.Errorf("Error parsing date from -s flag: %v\n", err)
		}

		// Return date formatted to user configuration.
		date = parsedDate.Format(config.ConvertDateFormat(cfg.FileFmt))
	}

	path := CreateFilePath(cfg, journalsDir, date)

	// Create file if it doesn't exist
	_, err := os.Stat(path)

	if errors.Is(err, fs.ErrNotExist) {
		err := os.WriteFile(path, []byte(""), 0644)
		if err != nil {
			return path, fmt.Errorf("error creating journal file: %s", err)
		}
	}

	return path, nil
}

// PrintFile reads the file at path and writes its contents to STDOUT.
// Returns an error if the file cannot be read.
func PrintFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	_, err = os.Stdout.Write(data)
	return err
}

func AppendToFile(path, content string, indent int) error {
	if indent < 0 {
		return fmt.Errorf("invalid indent: %d", indent)
	}

	prefix := strings.Repeat("\t", indent)
	bc := fmt.Sprintf("%s- %s\n", prefix, content)

	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file stats: %w", err)
	}

	if stat.Size() == 0 { // Can't check for a new line if the file size is 0
		_, err = file.WriteString(bc)
		return err
	}

	_, err = file.Seek(-1, io.SeekEnd) // -1 to read the byte before io.SeekEnd
	if err != nil {
		return fmt.Errorf("error seeking to end of file: %w", err)
	}

	buf := make([]byte, 1) // Only need to store the last byte
	_, err = file.Read(buf)
	if err != nil {
		return fmt.Errorf("error reading last byte: %w", err)
	}

	// When the last byte is not a new line add it to the bulleted content
	if buf[0] != '\n' {
		bc = fmt.Sprintf("\n%s- %s\n", prefix, content)
	}

	// Write data to the file
	_, err = file.WriteString(bc)
	return err
}

// isJournalEmpty checks if journal content is effectively empty
// (only whitespace, newlines, or dash placeholders)
func isJournalEmpty(content []byte) bool {
	if len(content) == 0 {
		return true
	}

	// Pattern: dash followed by optional whitespace, end of string
	re := regexp.MustCompile(`^-\s*$`)

	// Split content into lines and check each non-blank line
	lines := bytes.Split(content, []byte{'\n'})
	for _, line := range lines {
		// Check if line is blank (only whitespace)
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		// Non-blank line must match the placeholder pattern
		if !re.Match(line) {
			return false
		}
	}

	return true
}

// PrintJournalOverview prints an overview of all non-empty journal entries
// in reverse chronological order (newest first).
func PrintJournalOverview(w io.Writer, cfg *config.Config, journalsDir string) error {
	// Read the journals directory
	entries, err := os.ReadDir(journalsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("journals directory does not exist: %w", err)
		}
		return fmt.Errorf("error reading journals directory: %w", err)
	}

	// Determine expected file extension
	extension := ".md"
	if strings.EqualFold(cfg.FileType, "Org") {
		extension = ".org"
	}

	// Parse date format from config
	dateFormat := config.ConvertDateFormat(cfg.FileFmt)

	// Type to hold parsed journal entries
	type journalEntry struct {
		date    time.Time
		path    string
		content string
	}

	var journals []journalEntry

	// Process each entry in the directory
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if file has the correct extension
		if !strings.HasSuffix(name, extension) {
			continue
		}

		// Remove extension to get the date string
		dateStr := strings.TrimSuffix(name, extension)

		// Parse the date in UTC to ensure consistent weekday names
		date, err := time.ParseInLocation(dateFormat, dateStr, time.UTC)
		if err != nil {
			// Silently skip files that cannot be parsed as dates
			continue
		}

		// Read file content
		path := filepath.Join(journalsDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", name, err)
		}

		// Skip empty journals
		if isJournalEmpty(content) {
			continue
		}

		journals = append(journals, journalEntry{
			date:    date,
			path:    path,
			content: string(content),
		})
	}

	// Sort in reverse chronological order (newest first)
	sort.Slice(journals, func(i, j int) bool {
		return journals[i].date.After(journals[j].date)
	})

	// Write output for each journal
	for _, journal := range journals {
		// Write header line: date + day + 65 U+2500 chars
		header := journal.date.Format("2006-01-02") + " " +
			journal.date.Format("Mon") + " " +
			strings.Repeat("─", 65) + "\n"
		if _, err := w.Write([]byte(header)); err != nil {
			return fmt.Errorf("error writing header: %w", err)
		}

		// Write content
		if _, err := w.Write([]byte(journal.content)); err != nil {
			return fmt.Errorf("error writing content: %w", err)
		}

		// Write separator: if content ends with \n, write one more \n for blank line;
		// otherwise write \n\n to add newline after content plus blank separator
		if strings.HasSuffix(journal.content, "\n") {
			if _, err := w.Write([]byte("\n")); err != nil {
				return fmt.Errorf("error writing separator: %w", err)
			}
		} else {
			if _, err := w.Write([]byte("\n\n")); err != nil {
				return fmt.Errorf("error writing separator: %w", err)
			}
		}
	}

	return nil
}
