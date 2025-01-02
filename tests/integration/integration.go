package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHelper contains common utilities for integration tests
type TestHelper struct {
	t *testing.T

	// Test directories
	TempDir     string
	LogseqDir   string
	LogseqSetts string
	JournalsDir string
	PagesDir    string
	ConfigDir   string

	// Original environment state
	OriginalEditor string
	OriginalConfig string
	OriginalHome   string
}

// NewTestHelper creates and initializes a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	tempDir := t.TempDir()

	helper := &TestHelper{
		t:              t,
		TempDir:        tempDir,
		LogseqDir:      filepath.Join(tempDir, "Logseq"),
		LogseqSetts:    filepath.Join(tempDir, "Logseq", "logseq"),
		JournalsDir:    filepath.Join(tempDir, "Logseq", "journals"),
		PagesDir:       filepath.Join(tempDir, "Logseq", "pages"),
		ConfigDir:      filepath.Join(tempDir, ".config", "lsq"),
		OriginalEditor: os.Getenv("EDITOR"),
		OriginalHome:   os.Getenv("HOME"),
		OriginalConfig: os.Getenv("XDG_HOME_CONFIG"),
	}

	helper.setupTestEnvironment()
	return helper
}

// setupTestEnvironment creates the necessary directory structure and files
func (h *TestHelper) setupTestEnvironment() {
	h.t.Helper()

	// Create directory structure
	dirs := []string{
		h.LogseqDir,
		h.JournalsDir,
		h.ConfigDir,
		h.LogseqSetts,
		h.PagesDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			h.t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Set environment variables
	os.Setenv("HOME", h.TempDir)
	os.Setenv("EDITOR", "echo") // Use 'echo' as a safe test editor
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(h.TempDir, ".config"))
}

// Cleanup restores the original environment state
func (h *TestHelper) Cleanup() {
	h.t.Helper()

	os.Setenv("EDITOR", h.OriginalEditor)
	os.Setenv("HOME", h.OriginalHome)
	os.Setenv("XDG_CONFIG_HOME", h.OriginalConfig)
}

// AssertFileExists checks if a file exists and contains expected content
func (h *TestHelper) AssertFileExists(path string, expectedContent string) {
	h.t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if string(content) != expectedContent {
		h.t.Errorf("File content mismatch.\nExpected:\n%s\nGot:\n%s", expectedContent, content)
	}
}
