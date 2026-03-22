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

func TestRunQuery_AdvancedFileInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	// Write a query file.
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.edn")
	os.WriteFile(queryPath, []byte("[:find ?e . :where [?e :block/uuid]]\n"), 0644)

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

func TestRunQuery_AdvancedBothQueryAndFile(t *testing.T) {
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.edn")
	os.WriteFile(queryPath, []byte("[:find ?e :where [?e :block/uuid]]"), 0644)

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

func TestRunQuery_AdvancedEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "empty.edn")
	os.WriteFile(queryPath, []byte("   \n  "), 0644)

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
		json.NewDecoder(r.Body).Decode(&req)
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

