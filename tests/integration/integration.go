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
	JournalsDir string
	ConfigPath  string

	// Original environment state
	OriginalEditor string
	OriginalHome   string
}

// NewTestHelper creates and initializes a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	tempDir := t.TempDir()
	logseqDir := filepath.Join(tempDir, "Logseq")
	journalsDir := filepath.Join(logseqDir, "journals")

	helper := &TestHelper{
		t:              t,
		TempDir:        tempDir,
		LogseqDir:      logseqDir,
		JournalsDir:    journalsDir,
		ConfigPath:     filepath.Join(logseqDir, "logseq", "config.edn"),
		OriginalEditor: os.Getenv("EDITOR"),
		OriginalHome:   os.Getenv("HOME"),
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
		filepath.Join(h.LogseqDir, "logseq"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			h.t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Set up basic config.edn
	defaultConfig := `{
		:meta/version 1
		:preferred-format "Markdown"
		:journal/file-name-format "yyyy_MM_dd"
	}`

	if err := os.WriteFile(h.ConfigPath, []byte(defaultConfig), 0644); err != nil {
		h.t.Fatalf("Failed to write config file: %v", err)
	}

	// Set environment variables
	os.Setenv("HOME", h.TempDir)
	os.Setenv("EDITOR", "echo") // Use 'echo' as a safe test editor
}

// Cleanup restores the original environment state
func (h *TestHelper) Cleanup() {
	h.t.Helper()

	os.Setenv("EDITOR", h.OriginalEditor)
	os.Setenv("HOME", h.OriginalHome)
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
