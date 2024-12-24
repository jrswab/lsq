package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jrswab/lsq/config"
)

func TestConvertDateFormat(t *testing.T) {
	tests := map[string]struct {
		cfgFileFormat string
		want          string
	}{
		"yyyy_MM_dd (Logseq Default)": {
			cfgFileFormat: "yyyy_MM_dd",
			want:          "2006_01_02",
		},
		"yyyy_MM_d": {
			cfgFileFormat: "yyyy_MM_d",
			want:          "2006_01_2",
		},
		"yyyy_M_d": {
			cfgFileFormat: "yyyy_M_d",
			want:          "2006_1_2",
		},
		"yyyy_M_dd": {
			cfgFileFormat: "yyyy_M_dd",
			want:          "2006_1_02",
		},
		"yy_MM_dd": {
			cfgFileFormat: "yy_MM_dd",
			want:          "06_01_02",
		},
		"yy_M_dd": {
			cfgFileFormat: "yy_M_dd",
			want:          "06_1_02",
		},
		"yy_M_d": {
			cfgFileFormat: "yy_M_d",
			want:          "06_1_2",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := config.ConvertDateFormat(tt.cfgFileFormat); got != tt.want {
				t.Errorf("ConvertDateFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigLoad(t *testing.T) {
	// Create a temporary test directory
	tempDir, err := os.MkdirTemp("", "lsq-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// a mock Logseq config for testing
	logseqConfig := []byte(`{:meta/version 1
		:preferred-format "Markdown"
		:journal/file-name-format "yyyy_MM_dd"}`)

	testCases := []struct {
		name          string
		xdgConfigHome string
		setupFiles    map[string][]byte
		appPath       string
		expectedCfg   config.Config
		expectError   bool
	}{
		{
			name:          "Valid XDG config with existing lsq config",
			xdgConfigHome: tempDir,
			setupFiles: map[string][]byte{
				"lsq/config.edn": []byte(`{:meta/version 1 :file/format "yyyy_MM_dd" :file/type "Markdown" :file/path "/test/path"}`),
			},
			appPath: "/test/path",
			expectedCfg: config.Config{
				Version:  1,
				FileFmt:  "yyyy_MM_dd",
				FileType: "Markdown",
				FilePath: "/test/path",
			},
			expectError: false,
		},
		{
			name:          "No existing lsq config, valid Logseq config",
			xdgConfigHome: tempDir,
			setupFiles: map[string][]byte{
				"logseq/config": logseqConfig,
			},
			appPath: "/test/path",
			expectedCfg: config.Config{
				Version:  1,
				FileFmt:  "yyyy_MM_dd",
				FileType: "Markdown",
				FilePath: "/test/path",
			},
			expectError: false,
		},
		{
			name:          "Invalid EDN in lsq config",
			xdgConfigHome: tempDir,
			setupFiles: map[string][]byte{
				"lsq/config.edn": []byte(`{:invalid edn`),
			},
			appPath:     "/test/path",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test environment
			if tc.xdgConfigHome != "" {
				oldXDG := os.Getenv("XDG_CONFIG_HOME")
				os.Setenv("XDG_CONFIG_HOME", tc.xdgConfigHome)
				defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
			} else {
				oldXDG := os.Getenv("XDG_CONFIG_HOME")
				os.Unsetenv("XDG_CONFIG_HOME")
				defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
			}

			// Create test files
			for path, content := range tc.setupFiles {
				fullPath := filepath.Join(tempDir, path)

				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}

				err = os.WriteFile(fullPath, content, 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			// Run test
			cfg := &config.Config{}
			err := cfg.Load(tc.appPath)

			// Verify results
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.expectError {
				if !reflect.DeepEqual(*cfg, tc.expectedCfg) {
					t.Errorf("Config mismatch\nGot: %+v\nWant: %+v", *cfg, tc.expectedCfg)
				}
			}
		})
	}
}
