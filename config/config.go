package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	FilePath string `edn:"file/path"`
}

func (c *Config) Write() error {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	configPath := filepath.Join(configHome, "lsq", "config.edn")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	p, err := edn.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, p, 0644)
}

func (c *Config) Load(appPath string) error {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	cfgPath := filepath.Join(configHome, "lsq", "config.edn")

	_, err := os.Stat(cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error checking file path: %v\n", err)
		}

		// lsq config does not exist
		// Load app config for defaults.
		appCfg, err := loadAppConfig(appPath)
		if err != nil {
			return fmt.Errorf("error checking Logseq config path: %v\n", err)
		}

		c.FileFmt = appCfg.FileNameFmt
		c.FileType = appCfg.PreferredFmt
		c.FilePath = appPath
		c.Version = 1

		err = c.Write()
		if err != nil {
			return fmt.Errorf("error writing to lsq config file: %v\n", err)
		}

		return nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v\n", err)
	}

	err = edn.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("error unmarshaling config file: %v\n", err)
	}

	return nil
}

func loadAppConfig(cfgFile string) (*LogseqConfig, error) {
	// Set defaults before extracting data from config file:
	cfg := &LogseqConfig{
		CfgVers:      1,
		PreferredFmt: "Markdown",
		FileNameFmt:  "yyyy_MM_dd",
	}

	// Read config file to determine preferred format
	configData, err := os.ReadFile(cfgFile)
	if err != nil {
		return cfg, fmt.Errorf("error reading config file: %v\n", err)
	}

	// Update cfg with config values
	err = edn.Unmarshal(configData, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("error unmarshaling config data:%v", err)
	}

	return cfg, nil
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
