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
	Version    int    `edn:"meta/version"`
	FileType   string `edn:"file/type"`
	FileFmt    string `edn:"file/format"`
	DirPath    string `edn:"directory"`
	AppCfgDir  string `edn:"app/cfg-path"`
	AppCfgName string `edn:"app/cfg-name"`
}

func Load() (*Config, error) {
	var c *Config

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	cfgPath := filepath.Join(configDir, "lsq", "config.edn")

	data, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		// The user has a config file but we couldn't read it.
		// Report the error instead of ignoring their configuration.
		return nil, fmt.Errorf("error reading config file: %v\n", err)
	}

	if os.IsNotExist(err) {
		return nil, nil
	}

	if err := validator.New().ValidateFile(cfgPath); err != nil {
		return nil, fmt.Errorf("error validating config file: %v\n", err)
	}

	err = edn.Unmarshal(data, c)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config file: %v\n", err)
	}

	return c, nil
}

func LoadAppConfig(dirPath, appCfg string) (*Config, error) {
	// Set defaults before extracting data from config file:
	logCfg := &LogseqConfig{
		CfgVers:      1,
		PreferredFmt: "Markdown",
		FileNameFmt:  "yyyy_MM_dd",
	}

	// Read config file to determine preferred format
	configData, err := os.ReadFile(appCfg)
	if err != nil {
		return nil, err
	}

	// Update cfg with config values
	err = edn.Unmarshal(configData, &logCfg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config data:%v", err)
	}

	return &Config{
		FileFmt:  logCfg.FileNameFmt,
		FileType: logCfg.PreferredFmt,
		DirPath:  dirPath,
		Version:  1,
	}, nil
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
