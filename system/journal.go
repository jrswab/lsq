package system

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
