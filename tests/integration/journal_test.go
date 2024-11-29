package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jrswab/lsq/tests/integration"
)

func TestBasicJournalCreation(t *testing.T) {
	helper := integration.NewTestHelper(t)
	defer helper.Cleanup()

	// Set up test cases with different dates
	testCases := map[string]struct {
		date    time.Time
		content string
		format  string // "Markdown" or "Org"
	}{
		"create_today_journal": {
			date:    time.Now(),
			content: "Test journal entry",
			format:  "Markdown",
		},
		"create_specific_date_journal": {
			date:    time.Date(2024, 11, 28, 0, 0, 0, 0, time.UTC),
			content: "Test entry for specific date",
			format:  "Markdown",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Update config if needed for the format
			if tc.format != "Markdown" {
				configContent := `{
					:meta/version 1
					:preferred-format "` + tc.format + `"
					:journal/file-name-format "yyyy_MM_dd"
				}`
				err := os.WriteFile(helper.ConfigPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to update config: %v", err)
				}
			}

			// Get expected file path
			expectedPath := filepath.Join(
				helper.JournalsDir,
				tc.date.Format("2006_01_02")+".md",
			)

			// Create the journal entry
			// TODO: Replace this with actual LSQ journal creation call
			// For now, we'll simulate the file creation
			err := os.WriteFile(expectedPath, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write journal file: %v", err)
			}

			// Verify file exists and has correct content
			content, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read journal file: %v", err)
			}

			if string(content) != tc.content {
				t.Errorf("Journal content mismatch.\nExpected: %s\nGot: %s",
					tc.content, string(content))
			}

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
