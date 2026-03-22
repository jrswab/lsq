package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestQuerySimpleCLI_JSON_Success verifies remote simple query success via the built binary.
func TestQuerySimpleCLI_JSON_Success(t *testing.T) {
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

	res := RunCLI(lsqBinary, []string{
		"query", "simple",
		"--expr", "[[logseq]]",
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
		t.Errorf("error=%v, want null", m["error"])
	}

	results, ok := m["results"].([]any)
	if !ok || len(results) != 1 {
		t.Errorf("expected 1 result, got %v", m["results"])
	}

	warnings, ok := m["warnings"].([]any)
	if !ok || warnings == nil {
		t.Errorf("warnings should be [], got %T %v", m["warnings"], m["warnings"])
	}
}

// TestQuerySimpleCLI_JSON_TaskNow verifies task-now simple queries via the built binary.
func TestQuerySimpleCLI_JSON_TaskNow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"block/marker":"NOW"}]`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "simple",
		"--expr", "(task now)",
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

	if m["backend"] != "http" {
		t.Errorf("backend=%v", m["backend"])
	}
	if m["input_kind"] != "simple" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
}

// TestQuerySimpleCLI_NullResponse_IsError rejects 200+null simple query responses.
func TestQuerySimpleCLI_NullResponse_IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `null`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "simple",
		"--expr", "[[something-invalid]]",
		"--format", "json",
		"--api-url", srv.URL + "/api",
	})

	// 200 + null should be treated as an error by simple query execution.
	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for null response, got %d", res.ExitCode)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(res.Stdout), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, res.Stdout)
	}

	if m["error"] == nil {
		t.Error("expected error field to be populated")
	} else if errStr, ok := m["error"].(string); !ok || !strings.Contains(errStr, "returned null") {
		t.Errorf("expected error containing 'returned null', got %v", m["error"])
	}

	if m["results"] != nil {
		t.Errorf("expected results to be explicitly null, got %v", m["results"])
	}
}

// TestQuerySimpleCLI_ErrorEnvelope_IsError rejects explicit error envelopes from DB.q.
func TestQuerySimpleCLI_ErrorEnvelope_IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":"Unsupported block property"}`)
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "simple",
		"--expr", "{{query [[logseq]]}}",
		"--format", "json",
		"--api-url", srv.URL + "/api",
	})

	// 200 + {"error": ...} should be treated as an error by simple query execution.
	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for error envelope, got %d", res.ExitCode)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(res.Stdout), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, res.Stdout)
	}

	if m["error"] == nil {
		t.Error("expected error field to be populated")
	} else if errStr, ok := m["error"].(string); !ok || !strings.Contains(errStr, `{"error":"Unsupported block property"}`) {
		t.Errorf("expected error containing error json payload, got %v", m["error"])
	}

	if m["results"] != nil {
		t.Errorf("expected results to be explicitly null, got %v", m["results"])
	}
}
