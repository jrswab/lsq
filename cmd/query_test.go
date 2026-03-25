package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jrswab/lsq/cmd"
)

// TestRunQuery_DoctorJSON verifies JSON rendering for the doctor subcommand.
func TestRunQuery_DoctorJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if m["reachable"] != true {
		t.Errorf("reachable=%v", m["reachable"])
	}
	if m["command"] != "doctor" {
		t.Errorf("command=%v", m["command"])
	}
}

// TestRunQuery_DoctorNDJSON verifies NDJSON rendering for the doctor subcommand.
func TestRunQuery_DoctorNDJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "ndjson", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 ndjson line, got %d", len(lines))
	}
	if !json.Valid([]byte(lines[0])) {
		t.Errorf("not valid JSON: %s", lines[0])
	}
}

// TestRunQuery_DoctorText verifies text rendering for the doctor subcommand.
func TestRunQuery_DoctorText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "text", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Reachable:  true") {
		t.Errorf("missing 'Reachable:  true' in output:\n%s", stdout.String())
	}
}

// TestRunQuery_DoctorDefaultFormat verifies the default doctor output format.
func TestRunQuery_DoctorDefaultFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	// No --format flag → should default to text.
	code := cmd.RunQuery(
		[]string{"doctor", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Reachable:") {
		t.Errorf("expected text output (default format), got:\n%s", stdout.String())
	}
}

// TestRunQuery_UnsupportedFormat rejects unknown output formats.
func TestRunQuery_UnsupportedFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "xml"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unsupported format") {
		t.Errorf("expected format error in stderr, got: %s", stderr.String())
	}
}

// TestRunQuery_UnsupportedBackend rejects unsupported backends for HTTP-only commands.
func TestRunQuery_UnsupportedBackend(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--backend", "file"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unsupported backend") {
		t.Errorf("expected backend error in stderr, got: %s", stderr.String())
	}
}

// TestRunQuery_DoctorRejectsAdvancedFlags rejects flags that do not belong to doctor.
func TestRunQuery_DoctorRejectsAdvancedFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--query", "[:find ?e]"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined: -query") {
		t.Errorf("expected invalid flag error, got: %s", stderr.String())
	}
}

// TestRunQuery_DoctorRejectsExtraArgs rejects unexpected positional arguments.
func TestRunQuery_DoctorRejectsExtraArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "oops"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unexpected argument(s): oops") {
		t.Errorf("expected extras error, got: %s", stderr.String())
	}
}

// TestRunQuery_NoSubcommand reports usage when no query subcommand is provided.
func TestRunQuery_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery([]string{}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	errText := stderr.String()
	if !strings.Contains(errText, "usage: lsq query <doctor|advanced|simple>") {
		t.Errorf("expected usage in stderr, got: %s", errText)
	}
}

// TestRunQuery_Help reports generic query help for `lsq query -h`.
func TestRunQuery_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery([]string{"-h"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: lsq query <doctor|advanced|simple> [flags]") {
		t.Fatalf("expected query help in stdout, got: %s", stdout.String())
	}
}

func TestRunQuery_DoctorHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery([]string{"doctor", "-h"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "usage: lsq query doctor [flags]") {
		t.Fatalf("expected doctor help in stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "-backend") {
		t.Fatalf("expected flags in doctor help, got: %s", stdout.String())
	}
}

// TestRunQuery_UnknownSubcommand rejects unknown query subcommands.
func TestRunQuery_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"magic"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	errText := stderr.String()
	if !strings.Contains(errText, "unknown query subcommand") {
		t.Errorf("expected unknown subcommand error, got: %s", errText)
	}
	if !strings.Contains(errText, "usage: lsq query <doctor|advanced|simple>") {
		t.Errorf("expected usage in error hint, got: %s", errText)
	}
}

// TestRunQuery_BackendAutoResolvesToHTTP confirms auto backend uses HTTP.
func TestRunQuery_BackendAutoResolvesToHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--backend", "auto", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Backend in result is always "http" in Phase 1.
	if m["backend"] != "http" {
		t.Errorf("expected backend=http, got %v", m["backend"])
	}
}

// --- Advanced query tests ---

func TestRunQuery_AdvancedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"],["page-b"]]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--query", "[:find ?n :where [?p :block/name ?n]]", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if m["backend"] != "http" {
		t.Errorf("backend=%v", m["backend"])
	}
	if m["input_kind"] != "advanced" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
	if m["query_method"] != "logseq.DB.datascriptQuery" {
		t.Errorf("query_method=%v", m["query_method"])
	}
	results, ok := m["results"].([]any)
	if !ok || len(results) != 2 {
		t.Errorf("expected 2 results, got %v", m["results"])
	}
}

// TestRunQuery_AdvancedNDJSON verifies NDJSON rendering for advanced queries.
func TestRunQuery_AdvancedNDJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"],["page-b"]]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--query", "[:find ?n :where [?p :block/name ?n]]", "--format", "ndjson", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	// 2 results, no warnings → 2 expanded lines.
	if len(lines) != 2 {
		t.Errorf("expected 2 ndjson lines, got %d: %v", len(lines), lines)
	}
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d not valid JSON: %s", i, line)
		}
	}
}

// TestRunQuery_AdvancedText verifies text rendering for advanced queries.
func TestRunQuery_AdvancedText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"]]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--query", "[:find ?n :where [?p :block/name ?n]]", "--format", "text", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "page-a") {
		t.Errorf("expected results in text output, got:\n%s", stdout.String())
	}
}

// TestRunQuery_AdvancedFileInput loads advanced query text from a file.
func TestRunQuery_AdvancedFileInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	// Write a query file.
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.edn")
	if err := os.WriteFile(queryPath, []byte("[:find ?e . :where [?e :block/uuid]]\n"), 0644); err != nil {
		t.Fatalf("failed to write query fixture %q: %v", queryPath, err)
	}

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--file", queryPath, "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if m["input_kind"] != "advanced" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
}

// TestRunQuery_AdvancedMissingQuery rejects advanced calls without query input.
func TestRunQuery_AdvancedMissingQuery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--format", "json"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "one of --query or --file is required") {
		t.Errorf("expected missing input error, got: %s", stderr.String())
	}
}

// TestRunQuery_AdvancedWhitespaceOnlyQuery rejects blank advanced query text.
func TestRunQuery_AdvancedWhitespaceOnlyQuery(t *testing.T) {
	// A --query value that is nothing but whitespace must be treated as absent,
	// matching the trimming already applied to --file content.
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--query", "   \t  "},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for whitespace-only --query, got %d", code)
	}
	if !strings.Contains(stderr.String(), "one of --query or --file is required") {
		t.Errorf("expected missing input error for whitespace query, got: %s", stderr.String())
	}
}

// TestRunQuery_AdvancedBothQueryAndFile rejects conflicting advanced inputs.
func TestRunQuery_AdvancedBothQueryAndFile(t *testing.T) {
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.edn")
	if err := os.WriteFile(queryPath, []byte("[:find ?e :where [?e :block/uuid]]"), 0644); err != nil {
		t.Fatalf("failed to write query fixture %q: %v", queryPath, err)
	}

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--query", "[:find ?e :where [?e :block/uuid]]", "--file", queryPath},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Errorf("expected mutually exclusive error, got: %s", stderr.String())
	}
}

// TestRunQuery_AdvancedFileNotFound reports missing advanced query files.
func TestRunQuery_AdvancedFileNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--file", "/nonexistent/query.edn"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "error reading query file") {
		t.Errorf("expected file error, got: %s", stderr.String())
	}
}

// TestRunQuery_AdvancedEmptyFile rejects empty advanced query files.
func TestRunQuery_AdvancedEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "empty.edn")
	if err := os.WriteFile(queryPath, []byte("   \n  "), 0644); err != nil {
		t.Fatalf("failed to write query fixture %q: %v", queryPath, err)
	}

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--file", queryPath},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "query file is empty") {
		t.Errorf("expected empty file error, got: %s", stderr.String())
	}
}

// TestRunQuery_AdvancedBackendValidation enforces advanced backend policy.
func TestRunQuery_AdvancedBackendValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--query", "[:find ?e :where [?e :block/uuid]]", "--backend", "file"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unsupported backend") {
		t.Errorf("expected backend error, got: %s", stderr.String())
	}
}

// TestRunQuery_DoctorStillWorksAfterAdvanced guards doctor after advanced wiring.
func TestRunQuery_DoctorStillWorksAfterAdvanced(t *testing.T) {
	// Confirm doctor path is not broken by the advanced addition.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("doctor broke after advanced addition: exit %d; stderr: %s", code, stderr.String())
	}
	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid doctor JSON: %v", err)
	}
	if m["command"] != "doctor" {
		t.Errorf("command=%v", m["command"])
	}
}

// --- Simple query tests ---

func TestRunQuery_SimpleJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Args   []any  `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Method != "logseq.DB.q" {
			t.Errorf("expected method logseq.DB.q, got %q", req.Method)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"block/name":"logseq"}]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "[[logseq]]", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if m["backend"] != "http" {
		t.Errorf("backend=%v", m["backend"])
	}
	if m["input_kind"] != "simple" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
	if m["query_method"] != "logseq.DB.q" {
		t.Errorf("query_method=%v", m["query_method"])
	}
	if m["error"] != nil {
		t.Errorf("error=%v", m["error"])
	}
}

// TestRunQuery_SimpleTaskNowJSON verifies remote simple task query JSON output.
func TestRunQuery_SimpleTaskNowJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"block/marker":"NOW"}]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "(task now)", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if m["input_kind"] != "simple" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
}

// TestRunQuery_SimpleText verifies text rendering for simple queries.
func TestRunQuery_SimpleText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"block/name":"logseq"}]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "[[logseq]]", "--format", "text", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "logseq") {
		t.Errorf("expected results in text output, got:\n%s", stdout.String())
	}
}

// TestRunQuery_SimpleMissingExpr rejects simple queries without --expr.
func TestRunQuery_SimpleMissingExpr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--format", "json"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--expr is required") {
		t.Errorf("expected missing --expr error, got: %s", stderr.String())
	}
}

// TestRunQuery_SimpleWhitespaceOnlyExpr rejects blank simple expressions.
func TestRunQuery_SimpleWhitespaceOnlyExpr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "   \t  "},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for whitespace-only --expr, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--expr is required") {
		t.Errorf("expected missing --expr error for whitespace expr, got: %s", stderr.String())
	}
}

// TestRunQuery_SimpleBackendFileRejected rejects file backend during remote simple phases.
func TestRunQuery_SimpleBackendFileRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "[[logseq]]", "--backend", "file"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unsupported backend") {
		t.Errorf("expected backend error, got: %s", stderr.String())
	}
}

// TestRunQuery_DoctorStillWorksAfterSimple guards doctor after simple routing changes.
func TestRunQuery_DoctorStillWorksAfterSimple(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("doctor broke after simple addition: exit %d; stderr: %s", code, stderr.String())
	}
	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid doctor JSON: %v", err)
	}
	if m["command"] != "doctor" {
		t.Errorf("command=%v", m["command"])
	}
}

// TestRunQuery_AdvancedStillWorksAfterSimple guards advanced after simple routing changes.
func TestRunQuery_AdvancedStillWorksAfterSimple(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"]]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"advanced", "--query", "[:find ?n :where [?p :block/name ?n]]", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("advanced broke after simple addition: exit %d; stderr: %s", code, stderr.String())
	}
	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid advanced JSON: %v", err)
	}
	if m["input_kind"] != "advanced" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
}

// TestRunQuery_SimpleMacroSupported accepts wrapped page-ref simple queries.
func TestRunQuery_SimpleMacroSupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Args   []any  `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.Args) == 0 || req.Args[0] != "(task now)" {
			t.Errorf("expected stripped payload '(task now)', got %v", req.Args)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"block/marker":"NOW"}]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "{{query (task now)}}", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}
}

// TestRunQuery_SimpleMacroSupported_PageProperty accepts wrapped page-property queries.
func TestRunQuery_SimpleMacroSupported_PageProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Args []any `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.Args) == 0 || req.Args[0] != "(page-property type project)" {
			t.Errorf("expected stripped payload '(page-property type project)', got %v", req.Args)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "{{query (page-property type project)}}", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}
}

// TestRunQuery_SimpleMacroRejected_Maps rejects wrapped EDN map inputs.
func TestRunQuery_SimpleMacroRejected_Maps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "{{query {:query [:find ?b]}}}"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for map input, got %d", code)
	}
	if !strings.Contains(stderr.String(), "cannot contain maps") {
		t.Errorf("expected map rejection error, got: %s", stderr.String())
	}
}

// TestRunQuery_SimpleRawRejected_Maps rejects raw EDN map payloads.
func TestRunQuery_SimpleRawRejected_Maps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "{:query [:find ?b]}"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for raw map input, got %d", code)
	}
	if !strings.Contains(stderr.String(), "cannot contain maps") {
		t.Errorf("expected map rejection error, got: %s", stderr.String())
	}
}

func TestRunQuery_SimpleRawRejected_Vectors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "[:find ?b]"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for raw vector input, got %d", code)
	}
	if !strings.Contains(stderr.String(), "cannot contain advanced EDN vectors or Datalog") {
		t.Errorf("expected vector rejection error, got: %s", stderr.String())
	}
}

// TestRunQuery_SimpleMacroRejected_Datalog rejects wrapped Datalog inputs.
func TestRunQuery_SimpleMacroRejected_Datalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "{{query [:find ?b]}}"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for datalog input, got %d", code)
	}
	if !strings.Contains(stderr.String(), "cannot contain advanced EDN vectors or Datalog") {
		t.Errorf("expected datalog rejection error, got: %s", stderr.String())
	}
}

// TestRunQuery_SimpleMacroRejected_BeginQuery rejects BEGIN_QUERY blocks.
func TestRunQuery_SimpleMacroRejected_BeginQuery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "{{query #+BEGIN_QUERY\n[:find ?b]\n#+END_QUERY}}"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for BEGIN_QUERY input, got %d", code)
	}
	if !strings.Contains(stderr.String(), "cannot contain BEGIN_QUERY blocks") {
		t.Errorf("expected BEGIN_QUERY rejection error, got: %s", stderr.String())
	}
}

// TestRunQuery_SimpleMacroRejected_UnknownWrappedForm rejects unsupported wrapped DSL forms.
func TestRunQuery_SimpleMacroRejected_UnknownWrappedForm(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"simple", "--expr", "{{query (do-something-else foo)}}"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1 for unknown wrapped form, got %d", code)
	}
	if !strings.Contains(stderr.String(), "supported simple DSL subset") {
		t.Errorf("expected subset rejection error, got: %s", stderr.String())
	}
}
