package system

import (
	"fmt"
	"os"

	"github.com/jrswab/lsq/config"
	"olympos.io/encoding/edn"
)

func LoadConfig(cfgFile string) (*config.Config, error) {
	// Set defaults before extracting data from config file:
	var cfg = &config.Config{
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
