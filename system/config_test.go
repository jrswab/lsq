package system

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jrswab/lsq/config"
)

func TestLoadConfig(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()

	testCases := map[string]struct {
		configContent string
		setupFunc     func(string) error
		want          *config.Config
		wantErr       bool
	}{
		"default values when file doesn't exist": {
			configContent: "",
			setupFunc:     nil,
			want: &config.Config{
				CfgVers:      1,
				PreferredFmt: "Markdown",
				FileNameFmt:  "yyyy_MM_dd",
			},
			wantErr: true,
		},
		"successfully load custom values": {
			configContent: `{:meta/version 2
                           :preferred-format "Org"
                           :journal/file-name-format "dd-MM-yyyy"}`,
			setupFunc: nil,
			want: &config.Config{
				CfgVers:      2,
				PreferredFmt: "Org",
				FileNameFmt:  "dd-MM-yyyy",
			},
			wantErr: false,
		},
		"malformed EDN file": {
			configContent: `{:meta/version 1 :preferred-format}`, // Invalid EDN
			setupFunc:     nil,
			want: &config.Config{
				CfgVers:      1,
				PreferredFmt: "Markdown",
				FileNameFmt:  "yyyy_MM_dd",
			},
			wantErr: true,
		},
		"unreadable file": {
			configContent: `{:meta/version 1}`,
			setupFunc: func(path string) error {
				if err := os.WriteFile(path, []byte(`{:meta/version 1}`), 0644); err != nil {
					return err
				}
				return os.Chmod(path, 0000)
			},
			want: &config.Config{
				CfgVers:      1,
				PreferredFmt: "Markdown",
				FileNameFmt:  "yyyy_MM_dd",
			},
			wantErr: true,
		},
		"partial config keeps defaults for unspecified values": {
			configContent: `{:meta/version 2}`,
			setupFunc:     nil,
			want: &config.Config{
				CfgVers:      2,
				PreferredFmt: "Markdown",
				FileNameFmt:  "yyyy_MM_dd",
			},
			wantErr: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, fmt.Sprintf("%s.edn", name))

			// If there's content, write it to the file
			if tc.configContent != "" {
				err := os.WriteFile(configPath, []byte(tc.configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write test config file: %v", err)
				}
			}

			// If there's a setup function, run it
			if tc.setupFunc != nil {
				err := tc.setupFunc(configPath)
				if err != nil {
					t.Fatalf("Failed to setup test: %v", err)
				}
			}

			// Test the LoadConfig function
			got, err := LoadConfig(configPath)

			// Check error condition matches expected
			if (err != nil) != tc.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// If we expect success, compare the config values
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("LoadConfig() = %+v, want %+v", got, tc.want)
			}

			// Clean up test files
			err = os.Chmod(configPath, 0644) // Reset permissions for cleanup
			if err != nil {
				t.Logf("Failed to reset file permissions for cleanup: %v", err)
			}
		})
	}
}
