package system_test

import (
	"bytes"
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
		indent         int
		expectedResult string
		expectError    bool
	}{
		{
			name:           "new empty file",
			initialContent: "",
			appendContent:  "new content",
			indent:         0,
			expectedResult: "- new content\n",
			expectError:    false,
		},
		{
			name:           "append to file with content and newline",
			initialContent: "- existing content\n",
			appendContent:  "new content",
			indent:         0,
			expectedResult: "- existing content\n- new content\n",
			expectError:    false,
		},
		{
			name:           "append to file without trailing newline",
			initialContent: "- existing content",
			appendContent:  "new content",
			indent:         0,
			expectedResult: "- existing content\n- new content\n",
			expectError:    false,
		},
		{
			name:           "append empty content",
			initialContent: "- existing content\n",
			appendContent:  "",
			indent:         0,
			expectedResult: "- existing content\n- \n",
			expectError:    false,
		},
		{
			name:           "append content with special characters",
			initialContent: "- existing content\n",
			appendContent:  "content with * and - and #",
			indent:         0,
			expectedResult: "- existing content\n- content with * and - and #\n",
			expectError:    false,
		},
		{
			name:           "append multiple lines",
			initialContent: "- existing content\n",
			appendContent:  "line1\nline2",
			indent:         0,
			expectedResult: "- existing content\n- line1\nline2\n",
			expectError:    false,
		},
		// Indentation test cases
		{
			name:           "indent level 0 explicit",
			initialContent: "- existing content\n",
			appendContent:  "text",
			indent:         0,
			expectedResult: "- existing content\n- text\n",
			expectError:    false,
		},
		{
			name:           "indent level 1",
			initialContent: "- existing content\n",
			appendContent:  "text",
			indent:         1,
			expectedResult: "- existing content\n\t- text\n",
			expectError:    false,
		},
		{
			name:           "indent level 2",
			initialContent: "- existing content\n",
			appendContent:  "text",
			indent:         2,
			expectedResult: "- existing content\n\t\t- text\n",
			expectError:    false,
		},
		{
			name:           "indent level 3",
			initialContent: "- existing content\n",
			appendContent:  "#work/org",
			indent:         3,
			expectedResult: "- existing content\n\t\t\t- #work/org\n",
			expectError:    false,
		},
		{
			name:           "indent level 1 empty file",
			initialContent: "",
			appendContent:  "text",
			indent:         1,
			expectedResult: "\t- text\n",
			expectError:    false,
		},
		{
			name:           "indent level 2 empty file",
			initialContent: "",
			appendContent:  "text",
			indent:         2,
			expectedResult: "\t\t- text\n",
			expectError:    false,
		},
		{
			name:           "indent level 1 no trailing newline",
			initialContent: "- existing content",
			appendContent:  "text",
			indent:         1,
			expectedResult: "- existing content\n\t- text\n",
			expectError:    false,
		},
		{
			name:           "indent level 2 no trailing newline",
			initialContent: "- existing content",
			appendContent:  "text",
			indent:         2,
			expectedResult: "- existing content\n\t\t- text\n",
			expectError:    false,
		},
		{
			name:           "indent level 5",
			initialContent: "- existing content\n",
			appendContent:  "text",
			indent:         5,
			expectedResult: "- existing content\n\t\t\t\t\t- text\n",
			expectError:    false,
		},
		{
			name:           "indent with empty content",
			initialContent: "- existing content\n",
			appendContent:  "",
			indent:         1,
			expectedResult: "- existing content\n\t- \n",
			expectError:    false,
		},
		{
			name:           "indent with special characters in content",
			initialContent: "- existing content\n",
			appendContent:  "hello\tworld",
			indent:         1,
			expectedResult: "- existing content\n\t- hello\tworld\n",
			expectError:    false,
		},
		{
			name:           "negative indent returns error",
			initialContent: "",
			appendContent:  "text",
			indent:         -1,
			expectedResult: "",
			expectError:    true,
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

			err := system.AppendToFile(testFile, tt.appendContent, tt.indent)

			// Check error expectation
			if (err != nil) != tt.expectError {
				t.Errorf("AppendToFile() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError {
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

func TestPrintJournalOverview(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string // filename -> content (empty string = 0-byte file)
		setupDirs   []string          // subdirectories to create
		fileType    string            // "Markdown" or "Org"
		journalsDir string            // override directory path (empty = use temp dir)
		wantErr     bool
		wantOutput  string
	}{
		{
			name:        "directory does not exist",
			journalsDir: "/nonexistent/path/to/journals",
			fileType:    "Markdown",
			setupFiles:  nil,
			wantErr:     true,
			wantOutput:  "",
		},
		{
			name:       "empty directory",
			setupFiles: map[string]string{},
			fileType:   "Markdown",
			wantErr:    false,
			wantOutput: "",
		},
		{
			name: "all files are empty",
			setupFiles: map[string]string{
				"2006_01_02.md": "",
				"2006_01_03.md": "",
			},
			fileType:   "Markdown",
			wantErr:    false,
			wantOutput: "",
		},
		{
			name: "single non-empty md file",
			setupFiles: map[string]string{
				"2006_01_02.md": "- Journal entry 1\n- Journal entry 2\n",
			},
			fileType: "Markdown",
			wantErr:  false,
			wantOutput: "2006-01-02 Mon " + strings.Repeat("─", 65) + "\n" +
				"- Journal entry 1\n- Journal entry 2\n\n",
		},
		{
			name: "multiple non-empty md files reverse chronological order",
			setupFiles: map[string]string{
				"2006_01_02.md": "Older journal content",
				"2006_01_04.md": "Newest journal content",
				"2006_01_03.md": "Middle journal content",
			},
			fileType: "Markdown",
			wantErr:  false,
			wantOutput: "2006-01-04 Wed " + strings.Repeat("─", 65) + "\n" +
				"Newest journal content\n\n" +
				"2006-01-03 Tue " + strings.Repeat("─", 65) + "\n" +
				"Middle journal content\n\n" +
				"2006-01-02 Mon " + strings.Repeat("─", 65) + "\n" +
				"Older journal content\n\n",
		},
		{
			name: "mix of empty and non-empty files",
			setupFiles: map[string]string{
				"2006_01_02.md": "",
				"2006_01_03.md": "Has content",
				"2006_01_04.md": "",
			},
			fileType: "Markdown",
			wantErr:  false,
			wantOutput: "2006-01-03 Tue " + strings.Repeat("─", 65) + "\n" +
				"Has content\n\n",
		},
		{
			name: "file name cannot be parsed as date",
			setupFiles: map[string]string{
				"2006_01_02.md":   "Valid date",
				"invalid-date.md": "Invalid date content",
			},
			fileType: "Markdown",
			wantErr:  false,
			wantOutput: "2006-01-02 Mon " + strings.Repeat("─", 65) + "\n" +
				"Valid date\n\n",
		},
		{
			name: "non-journal file wrong extension",
			setupFiles: map[string]string{
				"2006_01_02.md":  "Valid journal",
				"2006_01_03.txt": "Text file content",
				"2006_01_04.org": "Org file content",
			},
			fileType: "Markdown",
			wantErr:  false,
			wantOutput: "2006-01-02 Mon " + strings.Repeat("─", 65) + "\n" +
				"Valid journal\n\n",
		},
		{
			name: "subdirectory inside journals dir",
			setupFiles: map[string]string{
				"2006_01_02.md": "Valid journal",
			},
			setupDirs: []string{"subdir"},
			fileType:  "Markdown",
			wantErr:   false,
			wantOutput: "2006-01-02 Mon " + strings.Repeat("─", 65) + "\n" +
				"Valid journal\n\n",
		},
		{
			name: "file content without trailing newline",
			setupFiles: map[string]string{
				"2006_01_02.md": "Content without newline",
			},
			fileType: "Markdown",
			wantErr:  false,
			wantOutput: "2006-01-02 Mon " + strings.Repeat("─", 65) + "\n" +
				"Content without newline\n\n",
		},
		{
			name: "unicode content in file",
			setupFiles: map[string]string{
				"2006_01_02.md": "测试 Test テスト 🎉",
			},
			fileType: "Markdown",
			wantErr:  false,
			wantOutput: "2006-01-02 Mon " + strings.Repeat("─", 65) + "\n" +
				"测试 Test テスト 🎉\n\n",
		},
		{
			name: "org file type config",
			setupFiles: map[string]string{
				"2006_01_02.md":  "Markdown file ignored",
				"2006_01_03.org": "Org file content",
				"2006_01_04.md":  "Another md ignored",
				"2006_01_05.org": "Another org content",
			},
			fileType: "Org",
			wantErr:  false,
			wantOutput: "2006-01-05 Thu " + strings.Repeat("─", 65) + "\n" +
				"Another org content\n\n" +
				"2006-01-03 Tue " + strings.Repeat("─", 65) + "\n" +
				"Org file content\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var journalsDir string
			if tt.journalsDir != "" {
				journalsDir = tt.journalsDir
			} else {
				journalsDir = t.TempDir()

				// Create subdirectories
				for _, dir := range tt.setupDirs {
					err := os.MkdirAll(filepath.Join(journalsDir, dir), 0755)
					if err != nil {
						t.Fatalf("Failed to create subdirectory: %v", err)
					}
				}

				// Create files
				for filename, content := range tt.setupFiles {
					path := filepath.Join(journalsDir, filename)
					err := os.WriteFile(path, []byte(content), 0644)
					if err != nil {
						t.Fatalf("Failed to write test file %s: %v", filename, err)
					}
				}
			}

			cfg := &config.Config{
				FileType: tt.fileType,
				FileFmt:  "yyyy_MM_dd",
			}

			var buf bytes.Buffer
			err := system.PrintJournalOverview(&buf, cfg, journalsDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("PrintJournalOverview() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got := buf.String(); got != tt.wantOutput {
				t.Errorf("PrintJournalOverview() output:\n%q\nwant:\n%q", got, tt.wantOutput)
			}
		})
	}

	// Header format assertion test - specific check for exact format
	t.Run("header format assertion", func(t *testing.T) {
		journalsDir := t.TempDir()

		// 2006-01-02 is a Monday
		cfg := &config.Config{
			FileType: "Markdown",
			FileFmt:  "yyyy_MM_dd",
		}

		err := os.WriteFile(filepath.Join(journalsDir, "2006_01_02.md"), []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		var buf bytes.Buffer
		err = system.PrintJournalOverview(&buf, cfg, journalsDir)
		if err != nil {
			t.Fatalf("PrintJournalOverview() error = %v", err)
		}

		// Expected header: 15 chars (date + day) + 65 U+2500 chars + newline
		expectedHeader := "2006-01-02 Mon " + strings.Repeat("─", 65) + "\n"
		got := buf.String()

		if !strings.HasPrefix(got, expectedHeader) {
			t.Errorf("Header format mismatch.\nGot prefix: %q\nWant:       %q", got[:min(len(got), len(expectedHeader))], expectedHeader)
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
