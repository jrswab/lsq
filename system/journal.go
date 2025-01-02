package system

import (
	"errors"
	"fmt"
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

func AppendToFile(path, content string) error {
	bc := fmt.Sprintf("- %s\n", content)

	// Open the file in append and write only mode; create if needed.
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write data to the file
	_, err = file.WriteString(bc)
	return err
}
