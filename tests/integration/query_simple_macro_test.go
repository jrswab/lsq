package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestQuerySimpleCLI_MacroStrippingIntegration verifies wrapped simple macros
// are stripped before the HTTP request is sent.
func TestQuerySimpleCLI_MacroStrippingIntegration(t *testing.T) {
	var receivedArg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Args []any `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.Args) > 0 {
			arg, ok := req.Args[0].(string)
			if !ok {
				t.Errorf("expected string arg, got %T", req.Args[0])
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			receivedArg = arg
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`[{"block/name":"logseq"}]`)); err != nil {
			t.Errorf("write response: %v", err)
			return
		}
	}))
	defer srv.Close()

	res := RunCLI(lsqBinary, []string{
		"query", "simple",
		"--expr", "{{query [[logseq]]}}",
		"--format", "json",
		"--api-url", srv.URL + "/api",
	})

	if res.ExitCode != 0 {
		t.Fatalf("exit code %d; stderr: %q stdout: %q", res.ExitCode, res.Stderr, res.Stdout)
	}
	if receivedArg != "[[logseq]]" {
		t.Fatalf("expected stripped macro [[logseq]], got %q", receivedArg)
	}
}
