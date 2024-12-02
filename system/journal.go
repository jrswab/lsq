package system

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jrswab/lsq/config"
)

func GetJournal(cfg *config.Config, journalsDir, date string) (string, error) {
	// Construct today's journal file path
	var extension = ".md"
	if cfg.PreferredFmt == "Org" {
		extension = ".org"
	}

	// Create the path for the specified date.
	path := filepath.Join(journalsDir, fmt.Sprintf("%s%s", date, extension))

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
