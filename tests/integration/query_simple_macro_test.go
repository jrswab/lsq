package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuerySimpleCLI_MacroStrippingIntegration(t *testing.T) {
	var receivedArg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Args []any `json:"args"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Args) > 0 {
			receivedArg = req.Args[0].(string)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"block/name":"logseq"}]`))
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
