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
	if err != nil && !os.IsNotExist(err) {
		// The user has a config file but we couldn't read it.
		// Report the error instead of ignoring their configuration.
		return nil, fmt.Errorf("error reading config file: %v\n", err)
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

	// Check for missing data and use defaults
	if c.DirPath == "" {
		c.DirPath = homeDir
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
