package system_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jrswab/lsq/config"
	"github.com/jrswab/lsq/system"
	i "github.com/jrswab/lsq/tests/integration"
)

func TestBasicJournalCreation(t *testing.T) {
	// Set up test cases with different dates
	testCases := map[string]struct {
		helper    *i.TestHelper
		date      time.Time
		content   string
		format    string // "Markdown" or "Org"
		setupFunc func(h *i.TestHelper)
	}{
		"New Journal": {
			helper:  i.NewTestHelper(t),
			date:    time.Now(),
			content: "",
			format:  "Markdown",
		},
		"Empty Format Preference": {
			helper:  i.NewTestHelper(t),
			date:    time.Now(),
			content: "",
			format:  "", // Should default to Markdown
		},
		"Todays Journal With Data": {
			helper:  i.NewTestHelper(t),
			date:    time.Now(),
			content: "Test entry for today's date.",
			format:  "Markdown",
		},
		"Opening a Past Journal": {
			helper:  i.NewTestHelper(t),
			date:    time.Date(2024, 11, 28, 0, 0, 0, 0, time.UTC),
			content: "Test entry for specific date.",
			format:  "Markdown",
		},
		"Future Date": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().AddDate(0, 0, 1), // Tomorrow
			content: "",
			format:  "Markdown",
		},
		"Far Past Date": {
			helper:  i.NewTestHelper(t),
			date:    time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
			content: "",
			format:  "Markdown",
		},
		"Unicode Content": {
			helper:  i.NewTestHelper(t),
			date:    time.Now(),
			content: "测试 Test テスト",
			format:  "Markdown",
		},
		"Large Content": {
			helper:  i.NewTestHelper(t),
			date:    time.Now(),
			content: strings.Repeat("Large content test ", 1000),
			format:  "Markdown",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var helper = tc.helper
			defer helper.Cleanup()

			if tc.setupFunc != nil {
				tc.setupFunc(helper)
			}

			// Update config if needed for the format
			if tc.format != "Markdown" {
				configContent := `{
					:meta/version 1
					:preferred-format "` + tc.format + `"
					:journal/file-name-format "yyyy_MM_dd"
				}`

				err := os.WriteFile(tc.helper.ConfigPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to update config: %v", err)
				}
			}

			cfg, err := system.LoadConfig(tc.helper.ConfigPath)
			if err != nil {
				t.Fatalf("Failed to load config file: %v", err)
			}

			var date = tc.date.Format(config.ConvertDateFormat(cfg.FileNameFmt))

			if tc.content != "" {
				existingPath := filepath.Join(helper.JournalsDir, date+".md")
				if tc.format != "Markdown" {
					existingPath = filepath.Join(helper.JournalsDir, date+".org")
				}

				// Create the journal file to simulate existing data
				err := os.WriteFile(existingPath, []byte(tc.content), 0644)
				if err != nil {
					t.Fatalf("Failed to update config: %v", err)
				}
			}

			// Get journal path and create the journal entry if needed
			expectedPath, err := system.GetJournal(cfg, helper.JournalsDir, date)
			if err != nil {
				t.Fatalf("Failed to get journal file: %v", err)
			}

			helper.AssertFileExists(expectedPath, tc.content)

			// Verify file permissions
			info, err := os.Stat(expectedPath)
			if err != nil {
				t.Fatalf("Failed to stat journal file: %v", err)
			}

			expectedPerm := os.FileMode(0644)
			if info.Mode().Perm() != expectedPerm {
				t.Errorf("Incorrect file permissions. Expected: %v, Got: %v",
					expectedPerm, info.Mode().Perm())
			}
		})
	}
}

func TestAppendToFile(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "test")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)

    t.Run("new file creation", func(t *testing.T) {
        testFile := filepath.Join(tmpDir, "new.md")
        if err := system.AppendToFile(testFile, "new content"); err != nil {
            t.Errorf("Failed to create new file: %v", err)
        }

        content, err := os.ReadFile(testFile)
        if err != nil {
            t.Fatal(err)
        }
        expected := "- new content"
        if string(content) != expected {
            t.Errorf("Expected %q, got %q", expected, string(content))
        }
    })

    t.Run("append to existing", func(t *testing.T) {
        testFile := filepath.Join(tmpDir, "existing.md")
        if err := system.AppendToFile(testFile, "first"); err != nil {
            t.Fatal(err)
        }
        if err := system.AppendToFile(testFile, "second"); err != nil {
            t.Errorf("Failed to append: %v", err)
        }

        content, err := os.ReadFile(testFile)
        if err != nil {
            t.Fatal(err)
        }
        expected := "- first\n- second"
        if string(content) != expected {
            t.Errorf("Expected %q, got %q", expected, string(content))
        }
    })

    t.Run("permission denied", func(t *testing.T) {
        readOnlyDir := filepath.Join(tmpDir, "readonly")
        if err := os.Mkdir(readOnlyDir, 0500); err != nil {
            t.Fatal(err)
        }
        
        testFile := filepath.Join(readOnlyDir, "test.md")
        err := system.AppendToFile(testFile, "content")
        if err == nil {
            t.Error("Expected error for read-only directory")
        }
    })
}

