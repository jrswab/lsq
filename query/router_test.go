package query_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jrswab/lsq/query"
)

// mockHTTPExecutor returns a dummy result so we can assert the input was passed through.
func mockHTTPExecutor(ctx context.Context, expr string) query.AdvancedResult {
	return query.AdvancedResult{
		Backend:     "http",
		InputKind:   "simple",
		QueryMethod: expr, // Abusing this field purely to verify exact parameter passthrough in tests
	}
}

// TestRunSimple_AutoResolvesToHTTP verifies auto backend dispatch for simple queries.
func TestRunSimple_AutoResolvesToHTTP(t *testing.T) {
	res, err := query.RunSimple(context.Background(), "auto", "[[logseq]]", mockHTTPExecutor)
	if err != nil {
		t.Fatalf("unexpected routing error: %v", err)
	}
	if res.Backend != "http" {
		t.Errorf("expected mapped backend=http, got %q", res.Backend)
	}
	if res.QueryMethod != "[[logseq]]" {
		t.Errorf("expr not forwarded, got %q", res.QueryMethod)
	}
}

// TestRunSimple_HTTPWorks verifies explicit HTTP backend dispatch.
func TestRunSimple_HTTPWorks(t *testing.T) {
	res, err := query.RunSimple(context.Background(), "http", "(task now)", mockHTTPExecutor)
	if err != nil {
		t.Fatalf("unexpected routing error: %v", err)
	}
	if res.Backend != "http" {
		t.Errorf("expected mapped backend=http, got %q", res.Backend)
	}
	if res.QueryMethod != "(task now)" {
		t.Errorf("expr not forwarded, got %q", res.QueryMethod)
	}
}

// TestRunSimple_RejectUnsupported rejects unsupported simple backends.
func TestRunSimple_RejectUnsupported(t *testing.T) {
	// e.g. "file" or "local" should yield explicit errors since they're disabled in Phase 2.
	_, err := query.RunSimple(context.Background(), "file", "[[logseq]]", mockHTTPExecutor)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `"file"`) {
		t.Errorf("expected error containing 'file', got: %v", err)
	}
}
