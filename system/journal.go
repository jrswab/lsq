package system

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jrswab/lsq/config"
)

func GetTodaysJournal(cfg *config.Config, journalsDir string) (string, error) {
	// Construct today's journal file path
	var extension = ".md"
	if cfg.PreferredFmt == "Org" {
		extension = ".org"
	}

	// Get today's date in YYYY_MM_DD format
	today := time.Now().Format(config.ConvertDateFormat(cfg.FileNameFmt))

	journalPath := filepath.Join(journalsDir, fmt.Sprintf("%s%s", today, extension))

	// Create file if it doesn't exist
	_, err := os.Stat(journalPath)

	if errors.Is(err, fs.ErrNotExist) {
		err := os.WriteFile(journalPath, []byte(""), 0644)
		if err != nil {
			return journalPath, fmt.Errorf("error creating journal file: %s", err)
		}
	}

	return journalPath, nil
}
