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

func TestExpandPath(t *testing.T) {
	// Save and restore HOME so tilde expansion uses a known value.
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", "/home/alice")

	tests := map[string]struct {
		input   string
		envVars map[string]string // extra env vars to set for this test
		want    string
	}{
		// FR-1: Tilde expansion
		"tilde with subpath": {
			input: "~/Documents/Logseq",
			want:  "/home/alice/Documents/Logseq",
		},
		"tilde only": {
			input: "~",
			want:  "/home/alice",
		},
		"tilde with trailing slash": {
			input: "~/",
			want:  "/home/alice/",
		},
		"tilde mid-path not expanded": {
			input: "/data/~backup/notes",
			want:  "/data/~backup/notes",
		},

		// FR-2: Environment variable expansion
		"env var HOME": {
			input: "$HOME/Documents/Logseq",
			want:  "/home/alice/Documents/Logseq",
		},
		"env var braced HOME": {
			input: "${HOME}/Logseq",
			want:  "/home/alice/Logseq",
		},
		"env var custom": {
			input:   "$XDG_DATA_HOME/logseq",
			envVars: map[string]string{"XDG_DATA_HOME": "/home/alice/.local/share"},
			want:    "/home/alice/.local/share/logseq",
		},

		// FR-3: Combined tilde and env var
		"tilde then env var": {
			input:   "~/$PROJECT/Logseq",
			envVars: map[string]string{"PROJECT": "work"},
			want:    "/home/alice/work/Logseq",
		},

		// FR-5: No-op for absolute paths
		"absolute path unchanged": {
			input: "/home/alice/Logseq",
			want:  "/home/alice/Logseq",
		},

		// FR-6: Empty string
		"empty string": {
			input: "",
			want:  "",
		},

		// Edge cases from spec
		"undefined env var expands to empty": {
			input: "$LSQ_TEST_UNDEFINED_VAR_XYZ/Logseq",
			want:  "/Logseq",
		},
		"relative path unchanged": {
			input: "Documents/Logseq",
			want:  "Documents/Logseq",
		},
		"tilde with username not expanded": {
			input: "~bob/Logseq",
			want:  "~bob/Logseq",
		},
		"double tilde not expanded": {
			input: "~~",
			want:  "~~",
		},
		"env var then tilde mid-path": {
			input: "$HOME/~/data",
			want:  "/home/alice/~/data",
		},
		"only env var": {
			input: "$HOME",
			want:  "/home/alice",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set any extra env vars for this test case, then clean up.
			for k, v := range tt.envVars {
				orig := os.Getenv(k)
				os.Setenv(k, v)
				defer os.Setenv(k, orig)
			}
			// Unset the undefined var to be sure.
			os.Unsetenv("LSQ_TEST_UNDEFINED_VAR_XYZ")

			got, err := config.ExpandPath(tt.input)
			if err != nil {
				t.Fatalf("ExpandPath(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfigLoad(t *testing.T) {
	// Create a temporary test directory to act as HOME.
	tempDir, err := os.MkdirTemp("", "lsq-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Resolve the config directory that os.UserConfigDir() will return
	// when HOME is set to tempDir. On macOS this is $HOME/Library/Application Support;
	// on Linux with XDG_CONFIG_HOME it is XDG_CONFIG_HOME directly.
	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	origHomeDir := os.Getenv("HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	os.Setenv("HOME", tempDir)
	resolvedConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("Failed to resolve config directory: %v", err)
	}
	os.Setenv("XDG_CONFIG_HOME", origConfigDir)
	os.Setenv("HOME", origHomeDir)

	// cfgPath is relative to resolvedConfigDir; setupFiles keys are relative to resolvedConfigDir.
	cfgRelPath := filepath.Join("lsq", "config.edn")

	testCases := []struct {
		name        string
		setupFiles  map[string][]byte // paths relative to resolvedConfigDir
		expectedCfg config.Config
		expectError bool
	}{
		{
			name:       "No existing lsq config, load defaults",
			setupFiles: map[string][]byte{},
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
			name: "Valid config with existing lsq config",
			setupFiles: map[string][]byte{
				cfgRelPath: []byte(`{:file/type "Markdown"
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
			name: "Invalid EDN in lsq config",
			setupFiles: map[string][]byte{
				cfgRelPath: []byte(`invalid edn`),
			},
			expectError: true,
		},
		{
			name: "Tilde in directory is expanded",
			setupFiles: map[string][]byte{
				cfgRelPath: []byte(`{:directory "~/Documents/Logseq"}`),
			},
			expectedCfg: config.Config{
				Version:     1,
				FileFmt:     "yyyy_MM_dd",
				FileType:    "Markdown",
				DirPath:     filepath.Join(tempDir, "Documents", "Logseq"),
				JournalsDir: filepath.Join(tempDir, "Documents", "Logseq", "journals"),
				PagesDir:    filepath.Join(tempDir, "Documents", "Logseq", "pages"),
			},
			expectError: false,
		},
		{
			name: "Env var in directory is expanded",
			setupFiles: map[string][]byte{
				cfgRelPath: []byte(`{:directory "$HOME/Documents/Logseq"}`),
			},
			expectedCfg: config.Config{
				Version:     1,
				FileFmt:     "yyyy_MM_dd",
				FileType:    "Markdown",
				DirPath:     filepath.Join(tempDir, "Documents", "Logseq"),
				JournalsDir: filepath.Join(tempDir, "Documents", "Logseq", "journals"),
				PagesDir:    filepath.Join(tempDir, "Documents", "Logseq", "pages"),
			},
			expectError: false,
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

			// Clean up any config files from prior subtests.
			lsqCfgDir := filepath.Join(resolvedConfigDir, "lsq")
			os.RemoveAll(lsqCfgDir)

			// Create test files relative to the resolved config directory.
			for relPath, content := range tc.setupFiles {
				fullPath := filepath.Join(resolvedConfigDir, relPath)

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
