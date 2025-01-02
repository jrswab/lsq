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

	cfgPath := filepath.Join("lsq", "config.edn")

	testCases := []struct {
		name          string
		xdgConfigHome string
		setupFiles    map[string][]byte
		expectedCfg   config.Config
		expectError   bool
	}{
		{
			name:          "No existing lsq config, load defaults",
			xdgConfigHome: tempDir,
			setupFiles:    map[string][]byte{},
			expectedCfg: config.Config{
				Version:     1,
				FileFmt:     "yyyy_MM_dd",
				FileType:    "Markdown",
				DirPath:     filepath.Join(tempDir, "Logseq"),
				JournalsDir: filepath.Join(tempDir, "Logseq", "journals"),
				PagesDir:    filepath.Join(tempDir, "Logseq", "pages"),
			},
			expectError: false,
		},
		{
			name:          "Valid XDG config with existing lsq config",
			xdgConfigHome: tempDir,
			setupFiles: map[string][]byte{
				cfgPath: []byte(`{:file/type "Markdown"
                              :file/format "yyyy_MM_dd"
                              :directory "/custom/path"}`),
			},
			expectedCfg: config.Config{
				Version:     1,
				FileFmt:     "yyyy_MM_dd",
				FileType:    "Markdown",
				DirPath:     "/custom/path",
				JournalsDir: "/custom/path/journals",
				PagesDir:    "/custom/path/pages",
			},
			expectError: false,
		},
		{
			name:          "Invalid EDN in lsq config",
			xdgConfigHome: tempDir,
			setupFiles: map[string][]byte{
				cfgPath: []byte(`invalid edn`),
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test environment
			origConfigDir := os.Getenv("XDG_CONFIG_HOME")
			origHomeDir := os.Getenv("HOME")

			defer func() {
				os.Setenv("XDG_CONFIG_HOME", origConfigDir)
				os.Setenv("HOME", origHomeDir)
			}()

			// Set test environment variables
			os.Setenv("XDG_CONFIG_HOME", tempDir)
			os.Setenv("HOME", tempDir)

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

			cfg, err := config.Load()

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
