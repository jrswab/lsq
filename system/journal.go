package system

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jrswab/lsq/config"
)

func CreateFilePath(cfg *config.Config, journalsDir, date string) string {

	// Construct today's journal file path
	var extension = ".md"
	if cfg.PreferredFmt == "Org" {
		extension = ".org"
	}

	// Create the path for the specified date.
	return filepath.Join(journalsDir, fmt.Sprintf("%s%s", date, extension))
}

func GetJournal(cfg *config.Config, journalsDir, date string) (string, error) {
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
	bc := fmt.Sprintf("- %s", content)

    // Try to create new file with O_EXCL
    file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
    if err == nil {
        // Successfully created new file
        defer file.Close()
        return os.WriteFile(path, []byte(bc), 0644)
    }
    
    // If error is not "file exists", return the error
	// (it should have been created at this point)
    if !os.IsExist(err) {
        return err
    }
    
    // File exists, append to it
    file, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = file.WriteString(fmt.Sprintf("\n%s", bc))
    return err
}
