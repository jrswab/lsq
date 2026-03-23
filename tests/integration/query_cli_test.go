package integration

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lsqBinary holds the path to the compiled lsq binary shared across all
// TestQueryCLI_* tests in this package. It is set once by TestMain.
var lsqBinary string

// TestMain builds the lsq binary once for all integration tests, then runs them.
func TestMain(m *testing.M) {
	// Locate the module root (two levels up from tests/integration/).
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve module root: %v\n", err)
		os.Exit(1)
	}

	bin, err := BuildBinary(moduleRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: %v\n", err)
		os.Exit(1)
	}
	lsqBinary = bin
	code := m.Run()
	_ = os.Remove(bin)
	os.Exit(code)
}

func refusedAPIURL(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return "http://" + addr + "/api"
}

// --- query doctor integration tests ---

func TestQueryCLI_DoctorJSON_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`) // scalar → probe success
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "doctor",
		"--format", "json",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %s", res.ExitCode, res.Stderr)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(res.Stdout), &m); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, res.Stdout)
	}

	// Verify spec-mandated shape.
	if m["backend"] != "http" {
		t.Errorf("backend=%v", m["backend"])
	}
	if m["command"] != "doctor" {
		t.Errorf("command=%v", m["command"])
	}
	if m["reachable"] != true {
		t.Errorf("reachable=%v", m["reachable"])
	}
	if m["error"] != nil {
		t.Errorf("error=%v, want null", m["error"])
	}

	// warnings must be an array, never null.
	warnings, ok := m["warnings"].([]any)
	if !ok || warnings == nil {
		t.Errorf("warnings should be [], got %T %v", m["warnings"], m["warnings"])
	}

	// Nested objects present.
	if _, ok := m["auth"].(map[string]any); !ok {
		t.Errorf("auth not an object: %v", m["auth"])
	}
	if _, ok := m["capabilities"].(map[string]any); !ok {
		t.Errorf("capabilities not an object: %v", m["capabilities"])
	}

	// Must end with newline (stable output contract).
	if !strings.HasSuffix(res.Stdout, "\n") {
		t.Error("JSON output must end with newline")
	}
}

// TestQueryCLI_DoctorJSON_Unreachable reports transport failures in JSON output.
func TestQueryCLI_DoctorJSON_Unreachable(t *testing.T) {
	res := RunCLI(lsqBinary, []string{
		"query", "doctor",
		"--format", "json",
		"--api-url", refusedAPIURL(t),
	})

	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for unreachable, got %d", res.ExitCode)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(res.Stdout), &m); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, res.Stdout)
	}
	if m["reachable"] != false {
		t.Errorf("reachable=%v, want false", m["reachable"])
	}
	if m["error"] == nil {
		t.Error("expected error to be set for unreachable")
	}
}

// TestQueryCLI_DoctorNDJSON_Healthy verifies NDJSON doctor output from the built binary.
func TestQueryCLI_DoctorNDJSON_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "doctor",
		"--format", "ndjson",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %s", res.ExitCode, res.Stderr)
	}

	lines := strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 ndjson line for doctor, got %d", len(lines))
	}
	if !json.Valid([]byte(lines[0])) {
		t.Errorf("ndjson line not valid JSON: %s", lines[0])
	}
}

// TestQueryCLI_DoctorText_Healthy verifies text doctor output from the built binary.
func TestQueryCLI_DoctorText_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "doctor",
		"--format", "text",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "Reachable:  true") {
		t.Errorf("expected 'Reachable:  true' in text output:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "Backend:    http") {
		t.Errorf("expected 'Backend:    http' in text output:\n%s", res.Stdout)
	}
}

// --- query advanced integration tests ---

func TestQueryCLI_AdvancedJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"],["page-b"]]`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "advanced",
		"--query", "[:find ?n :where [?p :block/name ?n]]",
		"--format", "json",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %s", res.ExitCode, res.Stderr)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(res.Stdout), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, res.Stdout)
	}

	// Verify spec-mandated shape.
	if m["backend"] != "http" {
		t.Errorf("backend=%v", m["backend"])
	}
	if m["input_kind"] != "advanced" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
	if m["query_method"] != "logseq.DB.datascriptQuery" {
		t.Errorf("query_method=%v", m["query_method"])
	}
	if m["error"] != nil {
		t.Errorf("error=%v, want null", m["error"])
	}

	results, ok := m["results"].([]any)
	if !ok || len(results) != 2 {
		t.Errorf("expected 2 results, got %v", m["results"])
	}

	warnings, ok := m["warnings"].([]any)
	if !ok || warnings == nil {
		t.Errorf("warnings should be [], got %T %v", m["warnings"], m["warnings"])
	}
}

// TestQueryCLI_AdvancedJSON_BothMethodsFail reports advanced query failures cleanly.
func TestQueryCLI_AdvancedJSON_BothMethodsFail(t *testing.T) {
	// Server always returns 401 → both methods fail → non-zero exit.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "advanced",
		"--query", "[:find ?n :where [?p :block/name ?n]]",
		"--format", "json",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", res.ExitCode)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(res.Stdout), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, res.Stdout)
	}
	if m["error"] == nil {
		t.Error("expected error field to be set on failure")
	}
	if m["results"] != nil {
		t.Errorf("expected null results on failure, got %v", m["results"])
	}
}

// TestQueryCLI_AdvancedNDJSON_ExpandsResults emits one line per advanced result row.
func TestQueryCLI_AdvancedNDJSON_ExpandsResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"],["page-b"],["page-c"]]`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "advanced",
		"--query", "[:find ?n :where [?p :block/name ?n]]",
		"--format", "ndjson",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %s", res.ExitCode, res.Stderr)
	}

	// 3 results, no warnings → 3 expanded ndjson lines.
	lines := strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 ndjson lines, got %d: %v", len(lines), lines)
	}
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d not valid JSON: %s", i, line)
		}
	}
}

// TestQueryCLI_AdvancedText_Success pretty-prints successful advanced results.
func TestQueryCLI_AdvancedText_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["my-page"]]`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "advanced",
		"--query", "[:find ?n :where [?p :block/name ?n]]",
		"--format", "text",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "my-page") {
		t.Errorf("expected 'my-page' in text output:\n%s", res.Stdout)
	}
}

// TestQueryCLI_AdvancedFileInput loads advanced query text from disk.
func TestQueryCLI_AdvancedFileInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	// Write a query file.
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.edn")
	if err := os.WriteFile(queryPath, []byte("[:find ?e . :where [?e :block/uuid]]\n"), 0644); err != nil {
		t.Fatalf("failed to write query file: %v", err)
	}

	res := RunCLI(lsqBinary, []string{
		"query", "advanced",
		"--file", queryPath,
		"--format", "json",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %s", res.ExitCode, res.Stderr)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(res.Stdout), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, res.Stdout)
	}
	if m["input_kind"] != "advanced" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
}

// --- input validation via real binary ---

func TestQueryCLI_AdvancedMissingQuery(t *testing.T) {
	res := RunCLI(lsqBinary, []string{"query", "advanced", "--format", "json"})

	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "one of --query or --file is required") {
		t.Errorf("expected validation error in stderr, got: %s", res.Stderr)
	}
}

// TestQueryCLI_UnsupportedFormat rejects unknown render formats in the built binary.
func TestQueryCLI_UnsupportedFormat(t *testing.T) {
	res := RunCLI(lsqBinary, []string{"query", "doctor", "--format", "xml"})

	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "unsupported format") {
		t.Errorf("expected format error in stderr, got: %s", res.Stderr)
	}
}

// TestQueryCLI_UnknownSubcommand rejects unknown query subcommands in the built binary.
func TestQueryCLI_UnknownSubcommand(t *testing.T) {
	res := RunCLI(lsqBinary, []string{"query", "magic"})

	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "unknown query subcommand") {
		t.Errorf("expected unknown subcommand error in stderr, got: %s", res.Stderr)
	}
}
