package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jrswab/lsq/query/backend/httpapi"
)

// newTestServer returns an httptest.Server whose handler routes on the
// "method" field of the JSON request body.
func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// apiReq mirrors the request shape for decoding in test handlers.
type apiReq struct {
	Method string `json:"method"`
	Args   []any  `json:"args"`
}

// --- Client.DoRaw tests ---

func TestDoRaw_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api" {
			t.Errorf("expected /api, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json content-type, got %s", ct)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"],["page-b"]]`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	raw, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"[:find ?n :where [?p :block/name ?n]]"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("expected valid JSON, got: %s", raw)
	}
}

func TestDoRaw_BearerAuth(t *testing.T) {
	var gotAuth string
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `"ok"`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "test-token-123", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"[:find ?e . :where [?e :block/uuid]]"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token-123" {
		t.Errorf("expected 'Bearer test-token-123', got %q", gotAuth)
	}
}

func TestDoRaw_NoAuthHeaderWhenTokenEmpty(t *testing.T) {
	var gotAuth string
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `"ok"`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	_, _ = c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if gotAuth != "" {
		t.Errorf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestDoRaw_AuthFailure401(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "bad-token", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	var authErr *httpapi.AuthError
	if !isAuthError(err, &authErr) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
	if authErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", authErr.StatusCode)
	}
}

func TestDoRaw_AuthFailure403(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "bad-token", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	var authErr *httpapi.AuthError
	if !isAuthError(err, &authErr) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
	if authErr.StatusCode != 403 {
		t.Errorf("expected status 403, got %d", authErr.StatusCode)
	}
}

func TestDoRaw_MalformedJSON(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `this is not json{{{`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestDoRaw_Timeout(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `"ok"`)
	})
	defer srv.Close()

	httpClient := &http.Client{Timeout: 50 * time.Millisecond}
	c := httpapi.NewClient(srv.URL, "", httpClient)

	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestDoRaw_ServerError500(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestDoRaw_Unreachable(t *testing.T) {
	c := httpapi.NewClient("http://127.0.0.1:1", "", &http.Client{Timeout: 500 * time.Millisecond})
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

// --- RunDoctor tests ---

func TestRunDoctor_AllHealthy(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "tok", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if !res.Reachable {
		t.Error("expected reachable=true")
	}
	if !res.Auth.Configured {
		t.Error("expected auth.configured=true")
	}
	if !res.Auth.Succeeded {
		t.Error("expected auth.succeeded=true")
	}
	if !res.Capabilities.DBQ {
		t.Error("expected capabilities.db_q=true")
	}
	if !res.Capabilities.DatascriptQuery {
		t.Error("expected capabilities.datascript_query=true")
	}
	if res.Error != nil {
		t.Errorf("expected no error, got %q", *res.Error)
	}
	if res.Backend != "http" {
		t.Errorf("expected backend=http, got %q", res.Backend)
	}
	if res.Command != "doctor" {
		t.Errorf("expected command=doctor, got %q", res.Command)
	}
}

func TestRunDoctor_NoToken(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if !res.Reachable {
		t.Error("expected reachable=true")
	}
	if res.Auth.Configured {
		t.Error("expected auth.configured=false when no token")
	}
	// Succeeded should be false when not configured (no assertion on succeeded truth).
	if res.Auth.Succeeded {
		t.Error("expected auth.succeeded=false when no token configured")
	}
}

func TestRunDoctor_AuthFailed(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "bad", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if !res.Reachable {
		t.Error("expected reachable=true even on auth failure")
	}
	if !res.Auth.Configured {
		t.Error("expected auth.configured=true")
	}
	if res.Auth.Succeeded {
		t.Error("expected auth.succeeded=false")
	}
	if res.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestRunDoctor_DBQFailedDatascriptWorks(t *testing.T) {
	callCount := 0
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req apiReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		callCount++
		switch req.Method {
		case "logseq.DB.q":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"not supported"}`)
		case "logseq.DB.datascriptQuery":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `42`)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "tok", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if !res.Reachable {
		t.Error("expected reachable=true")
	}
	if res.Capabilities.DBQ {
		t.Error("expected capabilities.db_q=false")
	}
	if !res.Capabilities.DatascriptQuery {
		t.Error("expected capabilities.datascript_query=true")
	}
	if len(res.Warnings) == 0 {
		t.Error("expected warning about DB.q unavailability")
	}
	if res.Error != nil {
		t.Errorf("expected no error, got %q", *res.Error)
	}
}

func TestRunDoctor_Unreachable(t *testing.T) {
	c := httpapi.NewClient("http://127.0.0.1:1", "", &http.Client{Timeout: 500 * time.Millisecond})
	res := httpapi.RunDoctor(context.Background(), c)

	if res.Reachable {
		t.Error("expected reachable=false")
	}
	if res.Error == nil {
		t.Error("expected error to be set")
	}
}

// --- RunAdvancedQuery tests ---

func TestRunAdvancedQuery_DBQSuccess(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"],["page-b"]]`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	res := httpapi.RunAdvancedQuery(context.Background(), c, "[:find ?n :where [?p :block/name ?n]]")

	if res.QueryMethod != "logseq.DB.q" {
		t.Errorf("expected query_method=logseq.DB.q, got %q", res.QueryMethod)
	}
	if res.Error != nil {
		t.Errorf("expected no error, got %q", *res.Error)
	}
	if string(res.Results) != `[["page-a"],["page-b"]]` {
		t.Errorf("unexpected results: %s", res.Results)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", res.Warnings)
	}
}

func TestRunAdvancedQuery_FallbackToDatascript(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req apiReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch req.Method {
		case "logseq.DB.q":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"nope"}`)
		case "logseq.DB.datascriptQuery":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[["result"]]`)
		}
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	res := httpapi.RunAdvancedQuery(context.Background(), c, "[:find ?n :where [?p :block/name ?n]]")

	if res.QueryMethod != "logseq.DB.datascriptQuery" {
		t.Errorf("expected datascriptQuery fallback, got %q", res.QueryMethod)
	}
	if res.Error != nil {
		t.Errorf("expected no error, got %q", *res.Error)
	}
	if len(res.Warnings) == 0 {
		t.Error("expected warning about fallback")
	}
}

func TestRunAdvancedQuery_BothFail(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"broken"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(srv.URL, "", nil)
	res := httpapi.RunAdvancedQuery(context.Background(), c, "[:find ?e :where [?e :block/uuid]]")

	if res.Error == nil {
		t.Fatal("expected error when both methods fail")
	}
	if string(res.Results) != "null" {
		t.Errorf("expected null results, got %s", res.Results)
	}
}

// isAuthError is a helper to unwrap AuthError via errors.As.
func isAuthError(err error, target **httpapi.AuthError) bool {
	// Use the standard library approach.
	type asInterface interface {
		Error() string
	}
	for err != nil {
		if ae, ok := err.(*httpapi.AuthError); ok {
			*target = ae
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
