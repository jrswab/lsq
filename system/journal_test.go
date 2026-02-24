package system_test

import (
	"fmt"
	"io"
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
		date      string
		content   string
		format    string // "Markdown" or "Org"
		setupFunc func(h *i.TestHelper)
	}{
		"New Journal": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().Format("2006-01-02"),
			content: "",
			format:  "Markdown",
		},
		"Empty Format Preference": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().Format("2006-01-02"),
			content: "",
			format:  "", // Should default to Markdown
		},
		"Todays Journal With Data": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().Format("2006-01-02"),
			content: "Test entry for today's date.",
			format:  "Markdown",
		},
		"Opening a Past Journal": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().AddDate(0, 0, -1).Format("2006-01-02"), // Yesterday
			content: "Test entry for specific date.",
			format:  "Markdown",
		},
		"Future Date": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().AddDate(0, 0, 1).Format("2006-01-02"), // Tomorrow
			content: "",
			format:  "Markdown",
		},
		"Far Past Date": {
			helper:  i.NewTestHelper(t),
			date:    "1999-12-31",
			content: "",
			format:  "Markdown",
		},
		"Unicode Content": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().Format("2006-01-02"),
			content: "测试 Test テスト",
			format:  "Markdown",
		},
		"Large Content": {
			helper:  i.NewTestHelper(t),
			date:    time.Now().Format("2006-01-02"),
			content: strings.Repeat("Large content test ", 1000),
			format:  "Markdown",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			helper := tc.helper
			defer helper.Cleanup()

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config file: %v", err)
			}

			// Simulate existing journal entries
			if tc.content != "" {
				time, err := time.Parse("2006-01-02", tc.date)
				if err != nil {
					t.Fatal("failed to parse date string", err)
				}

				date := time.Format(config.ConvertDateFormat(cfg.FileFmt))
				existingPath := filepath.Join(helper.JournalsDir, date+".md")
				if tc.format != "Markdown" {
					existingPath = filepath.Join(helper.JournalsDir, date+".org")
				}

				// Create the journal file to simulate existing data
				err = os.WriteFile(existingPath, []byte(tc.content), 0644)
				if err != nil {
					t.Fatalf("Failed to update config: %v", err)
				}
			}

			// Get journal path and create the journal entry if needed
			expectedPath, err := system.GetJournal(cfg, helper.JournalsDir, tc.date)
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

func TestPrintFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-print")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "print file with content",
			content:     "- journal entry one\n- journal entry two\n",
			expectError: false,
		},
		{
			name:        "print empty file",
			content:     "",
			expectError: false,
		},
		{
			name:        "print file with unicode",
			content:     "- 测试 Test テスト\n",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, fmt.Sprintf("%s.md", tt.name))
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Capture stdout by replacing os.Stdout with a pipe
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w

			printErr := system.PrintFile(testFile)

			w.Close()
			captured, _ := io.ReadAll(r)
			r.Close()
			os.Stdout = oldStdout

			if (printErr != nil) != tt.expectError {
				t.Errorf("PrintFile() error = %v, expectError %v", printErr, tt.expectError)
				return
			}

			if string(captured) != tt.content {
				t.Errorf("Expected %q, got %q", tt.content, string(captured))
			}
		})
	}

	// Test with a non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		err := system.PrintFile(filepath.Join(tmpDir, "does_not_exist.md"))
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

func TestAppendToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		initialContent string
		appendContent  string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "new empty file",
			initialContent: "",
			appendContent:  "new content",
			expectedResult: "- new content\n",
			expectError:    false,
		},
		{
			name:           "append to file with content and newline",
			initialContent: "- existing content\n",
			appendContent:  "new content",
			expectedResult: "- existing content\n- new content\n",
			expectError:    false,
		},
		{
			name:           "append to file without trailing newline",
			initialContent: "- existing content",
			appendContent:  "new content",
			expectedResult: "- existing content\n- new content\n",
			expectError:    false,
		},
		{
			name:           "append empty content",
			initialContent: "- existing content\n",
			appendContent:  "",
			expectedResult: "- existing content\n- \n",
			expectError:    false,
		},
		{
			name:           "append content with special characters",
			initialContent: "- existing content\n",
			appendContent:  "content with * and - and #",
			expectedResult: "- existing content\n- content with * and - and #\n",
			expectError:    false,
		},
		{
			name:           "append multiple lines",
			initialContent: "- existing content\n",
			appendContent:  "line1\nline2",
			expectedResult: "- existing content\n- line1\nline2\n",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, fmt.Sprintf("%s.md", tt.name))

			// Create file with initial content if any
			if tt.initialContent != "" {
				if err := os.WriteFile(testFile, []byte(tt.initialContent), 0644); err != nil {
					t.Fatal(err)
				}
			}

			err := system.AppendToFile(testFile, tt.appendContent)

			// Check error expectation
			if (err != nil) != tt.expectError {
				t.Errorf("AppendToFile() error = %v, expectError %v", err, tt.expectError)
				return
			}

			// Read and verify file content
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			if string(content) != tt.expectedResult {
				t.Errorf("Expected %q, got %q", tt.expectedResult, string(content))
			}
		})
	}
}
