package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jrswab/lsq/validator"
	"olympos.io/encoding/edn"
)

type LogseqConfig struct {
	CfgVers      int    `edn:"meta/version"`
	PreferredFmt string `edn:"preferred-format"`
	FileNameFmt  string `edn:"journal/file-name-format"`
}

type Config struct {
	Version  int    `edn:"meta/version"`
	FileType string `edn:"file/type"`
	FileFmt  string `edn:"file/format"`

	// Paths
	DirPath     string `edn:"directory"`
	JournalsDir string `edn:"journals/directory"`
	PagesDir    string `edn:"pages/directory"`
}

func Load() (*Config, error) {
	c := &Config{Version: 1}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	cfgPath := filepath.Join(configDir, "lsq", "config.edn")

	data, dErr := os.ReadFile(cfgPath)
	if dErr != nil && !os.IsNotExist(dErr) {
		// The user has a config file but we couldn't read it.
		// Report the error instead of ignoring their configuration.
		return nil, fmt.Errorf("error reading config file: %v\n", dErr)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	if os.IsNotExist(dErr) {
		// Set a defaults in case the user did not provide an override
		c.FileType = "Markdown"
		c.FileFmt = "yyyy_MM_dd"

		c.DirPath = filepath.Join(homeDir, "Logseq")
		c.JournalsDir = filepath.Join(c.DirPath, "journals")
		c.PagesDir = filepath.Join(c.DirPath, "pages")

		return c, nil
	}

	if err := validator.New().ValidateFile(cfgPath); err != nil {
		return nil, fmt.Errorf("error validating config file: %v\n", err)
	}

	err = edn.Unmarshal(data, c)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config file: %v\n", err)
	}

	// Expand ~ and environment variables in the directory path
	c.DirPath, err = ExpandPath(c.DirPath)
	if err != nil {
		return nil, fmt.Errorf("error expanding directory path: %v\n", err)
	}

	// Check for missing data and use defaults
	if c.DirPath == "" {
		c.DirPath = filepath.Join(homeDir, "Logseq")
	}

	// Set Logseq default directories
	c.JournalsDir = filepath.Join(c.DirPath, "journals")
	c.PagesDir = filepath.Join(c.DirPath, "pages")

	if c.FileType == "" {
		c.FileType = "Markdown"
	}

	if c.FileFmt == "" {
		c.FileFmt = "yyyy_MM_dd"
	}

	return c, nil
}

// ExpandPath expands a leading tilde (~) to the current user's home directory
// and then expands environment variables ($VAR and ${VAR} syntax).
// Tilde expansion only occurs when the path is exactly "~" or starts with "~/".
// Patterns like "~bob/path" or "~~" are not expanded.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Tilde expansion: only bare ~ or ~/...
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = home + path[1:]
	}

	// Environment variable expansion
	path = os.ExpandEnv(path)

	return path, nil
}

func ConvertDateFormat(cfgFileFormat string) string {
	lsqFmts := [][]string{
		{"yyyy", "2006"},
		{"yy", "06"},
		{"MM", "01"},
		{"M", "1"},
		{"dd", "02"},
		{"d", "2"},
	}

	goFormat := cfgFileFormat
	for _, val := range lsqFmts {
		goFormat = strings.ReplaceAll(goFormat, val[0], val[1])
	}

	return goFormat
}
