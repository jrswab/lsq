package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The legacy tests use the same compiled lsqBinary that query_cli_test.go
// compiled via TestMain.

func TestLegacyCLI_DefaultJournalPath(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Write a minimal config so lsq knows where the graph is.
	configContent := []byte(`{:graph "` + helper.LogseqDir + `"}`)
	os.WriteFile(filepath.Join(helper.ConfigDir, "config.edn"), configContent, 0644)

	// Invoke lsq with no arguments. It should try to open today's journal.
	// Because EDITOR=echo, the output will just be the filepath.
	res := RunCLI(lsqBinary, []string{})

	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", res.ExitCode, res.Stderr)
	}

	// stdout should contain "journals" somewhere in the echoed path.
	if !strings.Contains(res.Stdout, "journals") {
		t.Errorf("expected default journal path in output, got: %s", res.Stdout)
	}
}

func TestLegacyCLI_PrintContent(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Setup config and a dummy page.
	configContent := []byte(`{:graph "` + helper.LogseqDir + `"}`)
	os.WriteFile(filepath.Join(helper.ConfigDir, "config.edn"), configContent, 0644)

	pagePath := filepath.Join(helper.PagesDir, "Test Page.md")
	os.WriteFile(pagePath, []byte("- smoke test content\n"), 0644)

	// Invoke lsq -p <page> -c to cat the file.
	res := RunCLI(lsqBinary, []string{"-p", "Test Page.md", "-c"})

	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", res.ExitCode, res.Stderr)
	}

	if !strings.Contains(res.Stdout, "- smoke test content") {
		t.Errorf("expected page content in output, got: %s", res.Stdout)
	}
}

func TestLegacyCLI_FindFile(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Setup config and a dummy page.
	configContent := []byte(`{:graph "` + helper.LogseqDir + `"}`)
	os.WriteFile(filepath.Join(helper.ConfigDir, "config.edn"), configContent, 0644)

	pagePath := filepath.Join(helper.PagesDir, "Another Page.md")
	os.WriteFile(pagePath, []byte("- empty\n"), 0644)

	// Invoke lsq -f to find the file path.
	res := RunCLI(lsqBinary, []string{"-f", "Another Page"})

	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", res.ExitCode, res.Stderr)
	}

	// Output should be the absolute or relative path to the file.
	if !strings.Contains(res.Stdout, "Another Page.md") {
		t.Errorf("expected page filename in output, got: %s", res.Stdout)
	}
}

func TestLegacyCLI_FindAlias(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Setup config and a dummy page with an alias property.
	configContent := []byte(`{:graph "` + helper.LogseqDir + `"}`)
	os.WriteFile(filepath.Join(helper.ConfigDir, "config.edn"), configContent, 0644)

	pagePath := filepath.Join(helper.PagesDir, "Original Target.md")
	// The lsq trie parses properties, including alias::.
	os.WriteFile(pagePath, []byte("alias:: Secret Keyword\n\n- some content\n"), 0644)

	// Invoke lsq -f to search by the alias.
	res := RunCLI(lsqBinary, []string{"-f", "Secret Keyword"})

	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", res.ExitCode, res.Stderr)
	}

	// Output should contain the path to the original file that holds the alias.
	if !strings.Contains(res.Stdout, "Original Target.md") {
		t.Errorf("expected original page filename in output for alias search, got: %s", res.Stdout)
	}
}
