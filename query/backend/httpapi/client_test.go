package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jrswab/lsq/query/backend/httpapi"
)

// newTestServer returns an httptest.Server with the given handler.
func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// apiURL returns the full API endpoint URL for a test server.
// BaseURL semantics: full endpoint including path, e.g. http://host:port/api.
func apiURL(srv *httptest.Server) string {
	return srv.URL + "/api"
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

	c := httpapi.NewClient(apiURL(srv), "", nil)
	raw, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"[:find ?n :where [?p :block/name ?n]]"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("expected valid JSON, got: %s", raw)
	}
}

func TestDoRaw_BaseURLIsFullEndpoint(t *testing.T) {
	// Verify DoRaw posts directly to BaseURL without appending "/api".
	// We set BaseURL to srv.URL+"/api" and expect the request at /api.
	// If DoRaw appended "/api" again, the path would be /api/api.
	var gotPath string
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `"ok"`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	_, _ = c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})

	if gotPath != "/api" {
		t.Errorf("expected request path /api, got %q (double-append bug?)", gotPath)
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

	c := httpapi.NewClient(apiURL(srv), "test-token-123", nil)
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

	c := httpapi.NewClient(apiURL(srv), "", nil)
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

	c := httpapi.NewClient(apiURL(srv), "bad-token", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	var authErr *httpapi.AuthError
	if !errors.As(err, &authErr) {
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

	c := httpapi.NewClient(apiURL(srv), "bad-token", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	var authErr *httpapi.AuthError
	if !errors.As(err, &authErr) {
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

	c := httpapi.NewClient(apiURL(srv), "", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	var me *httpapi.MethodError
	if !errors.As(err, &me) {
		t.Fatalf("expected MethodError, got %T: %v", err, err)
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
	c := httpapi.NewClient(apiURL(srv), "", httpClient)

	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var te *httpapi.TransportError
	if !errors.As(err, &te) {
		t.Fatalf("expected TransportError for timeout, got %T: %v", err, err)
	}
}

func TestDoRaw_ServerError500(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	var me *httpapi.MethodError
	if !errors.As(err, &me) {
		t.Fatalf("expected MethodError for 500, got %T: %v", err, err)
	}
	if me.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", me.StatusCode)
	}
}

func TestDoRaw_Unreachable(t *testing.T) {
	// Use a port that is extremely unlikely to be listening.
	c := httpapi.NewClient("http://127.0.0.1:1/api", "", &http.Client{Timeout: 500 * time.Millisecond})
	_, err := c.DoRaw(context.Background(), "logseq.DB.q", []any{"test"})
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	var te *httpapi.TransportError
	if !errors.As(err, &te) {
		t.Fatalf("expected TransportError, got %T: %v", err, err)
	}
}

// --- Probe validation tests ---

func TestProbeDBQ_ErrorEnvelope200(t *testing.T) {
	// Server returns 200 with a JSON error envelope — probe should fail.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":"method not found"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err == nil {
		t.Fatal("expected probe failure for 200 + error envelope, got nil")
	}
	var me *httpapi.MethodError
	if !errors.As(err, &me) {
		t.Fatalf("expected MethodError, got %T: %v", err, err)
	}
}

func TestProbeDatascriptQuery_ErrorEnvelope200(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":"unsupported method"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDatascriptQuery(context.Background())
	if err == nil {
		t.Fatal("expected probe failure for 200 + error envelope, got nil")
	}
}

func TestProbeDBQ_EmptyErrorField200(t *testing.T) {
	// 200 with {"error":""} should be treated as success (empty error string).
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":""}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err != nil {
		t.Fatalf("expected success for empty error field, got %v", err)
	}
}

func TestProbeDBQ_NullErrorField200(t *testing.T) {
	// 200 with {"error":null} should be treated as success.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":null,"result":42}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err != nil {
		t.Fatalf("expected success for null error field, got %v", err)
	}
}

func TestProbeDBQ_ArrayResult200(t *testing.T) {
	// 200 with an array result should succeed (common query response shape).
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[["page-a"],["page-b"]]`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err != nil {
		t.Fatalf("expected success for array result, got %v", err)
	}
}

func TestProbeDBQ_ScalarResult200(t *testing.T) {
	// 200 with a scalar result (number) should succeed.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err != nil {
		t.Fatalf("expected success for scalar result, got %v", err)
	}
}

func TestProbeDBQ_ErrorObjectEnvelope200(t *testing.T) {
	// 200 with {"error": {"message": "..."}} — structured error object.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":{"message":"method not supported","code":404}}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err == nil {
		t.Fatal("expected probe failure for 200 + error object envelope, got nil")
	}
}

func TestProbeDBQ_NumericErrorValue200(t *testing.T) {
	// 200 with {"error": 500} — numeric error value.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":500}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err == nil {
		t.Fatal("expected probe failure for 200 + numeric error, got nil")
	}
}

func TestProbeDBQ_OkFalseEnvelope200(t *testing.T) {
	// 200 with {"ok": false, "error": "something"} — boolean flag.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":false,"error":"something went wrong"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err == nil {
		t.Fatal("expected probe failure for 200 + ok:false, got nil")
	}
}

func TestProbeDBQ_SuccessFalseEnvelope200(t *testing.T) {
	// 200 with {"success": false, "error": "..."} — boolean flag.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"success":false,"error":"query engine unavailable"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err == nil {
		t.Fatal("expected probe failure for 200 + success:false, got nil")
	}
}

func TestProbeDBQ_OkTrueWithResult200(t *testing.T) {
	// 200 with {"ok": true, "result": [...]} — should succeed.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true,"result":[["page-a"]]}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err != nil {
		t.Fatalf("expected success for ok:true envelope, got %v", err)
	}
}

func TestProbeDBQ_ResultWithNonBoolOk200(t *testing.T) {
	// 200 with {"ok": "yes"} — non-boolean ok is ignored (treated as success).
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":"yes","data":42}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	err := c.ProbeDBQ(context.Background())
	if err != nil {
		t.Fatalf("expected success for non-boolean ok field, got %v", err)
	}
}

// --- RunDoctor tests ---

func TestRunDoctor_AllHealthy(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "tok", nil)
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

func TestRunDoctor_APIURLMatchesBaseURL(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	})
	defer srv.Close()

	fullURL := apiURL(srv)
	c := httpapi.NewClient(fullURL, "", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if res.APIURL != fullURL {
		t.Errorf("expected api_url=%q, got %q", fullURL, res.APIURL)
	}
	if !strings.HasSuffix(res.APIURL, "/api") {
		t.Errorf("expected api_url to end with /api, got %q", res.APIURL)
	}
}

func TestRunDoctor_NoToken(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if !res.Reachable {
		t.Error("expected reachable=true")
	}
	if res.Auth.Configured {
		t.Error("expected auth.configured=false when no token")
	}
	if res.Auth.Succeeded {
		t.Error("expected auth.succeeded=false when no token configured")
	}
}

func TestRunDoctor_AuthFailed(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "bad", nil)
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
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req apiReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
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

	c := httpapi.NewClient(apiURL(srv), "tok", nil)
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

func TestRunDoctor_BothMethodsFail_ReachableTrue(t *testing.T) {
	// FIX VERIFICATION: Both methods return 500 (method-layer error).
	// API IS reachable — only capabilities are missing.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"broken"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "tok", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if !res.Reachable {
		t.Error("expected reachable=true when server responds with 500")
	}
	if res.Capabilities.DBQ {
		t.Error("expected capabilities.db_q=false")
	}
	if res.Capabilities.DatascriptQuery {
		t.Error("expected capabilities.datascript_query=false")
	}
	if res.Error == nil {
		t.Fatal("expected error to be set")
	}
	if !res.Auth.Succeeded {
		t.Error("expected auth.succeeded=true (server responded, no auth error)")
	}
	// Verify the error message includes context from both methods.
	if !strings.Contains(*res.Error, "logseq.DB.q") {
		t.Errorf("expected error to mention logseq.DB.q, got %q", *res.Error)
	}
	if !strings.Contains(*res.Error, "logseq.DB.datascriptQuery") {
		t.Errorf("expected error to mention logseq.DB.datascriptQuery, got %q", *res.Error)
	}
}

func TestRunDoctor_BothMethodsReturnErrorEnvelope200(t *testing.T) {
	// FIX VERIFICATION: Both methods return 200 with error envelopes.
	// API is reachable, but probe validation catches the error envelopes.
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":"method not found"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunDoctor(context.Background(), c)

	if !res.Reachable {
		t.Error("expected reachable=true when server returns 200")
	}
	if res.Capabilities.DBQ {
		t.Error("expected capabilities.db_q=false for error envelope")
	}
	if res.Capabilities.DatascriptQuery {
		t.Error("expected capabilities.datascript_query=false for error envelope")
	}
	if res.Error == nil {
		t.Fatal("expected error to be set")
	}
	// Verify message is actionable: includes both method names.
	if !strings.Contains(*res.Error, "logseq.DB.q") || !strings.Contains(*res.Error, "logseq.DB.datascriptQuery") {
		t.Errorf("expected actionable error with both method names, got %q", *res.Error)
	}
}

func TestRunDoctor_Unreachable(t *testing.T) {
	c := httpapi.NewClient("http://127.0.0.1:1/api", "", &http.Client{Timeout: 500 * time.Millisecond})
	res := httpapi.RunDoctor(context.Background(), c)

	if res.Reachable {
		t.Error("expected reachable=false")
	}
	if res.Error == nil {
		t.Error("expected error to be set")
	}
}

// --- RunAdvancedQuery tests ---

func TestRunAdvancedQuery_DatascriptQuerySuccess(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req apiReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Method == "logseq.DB.datascriptQuery" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[["page-a"],["page-b"]]`)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunAdvancedQuery(context.Background(), c, "[:find ?n :where [?p :block/name ?n]]")

	if res.QueryMethod != "logseq.DB.datascriptQuery" {
		t.Errorf("expected query_method=logseq.DB.datascriptQuery, got %q", res.QueryMethod)
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

func TestRunAdvancedQuery_Failure(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"broken"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunAdvancedQuery(context.Background(), c, "[:find ?e :where [?e :block/uuid]]")

	if res.Error == nil {
		t.Fatal("expected error on failure")
	}
	if string(res.Results) != "null" {
		t.Errorf("expected null results, got %s", res.Results)
	}
}

// --- RunSimpleQuery tests ---

func TestRunSimpleQuery_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req apiReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Method != "logseq.DB.q" {
			t.Errorf("expected method logseq.DB.q, got %q", req.Method)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"block/name":"logseq"},{"block/name":"typescript"}]`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunSimpleQuery(context.Background(), c, "[[logseq]]")

	if res.Backend != "http" {
		t.Errorf("expected backend=http, got %q", res.Backend)
	}
	if res.InputKind != "simple" {
		t.Errorf("expected input_kind=simple, got %q", res.InputKind)
	}
	if res.QueryMethod != "logseq.DB.q" {
		t.Errorf("expected query_method=logseq.DB.q, got %q", res.QueryMethod)
	}
	if res.Error != nil {
		t.Errorf("expected no error, got %q", *res.Error)
	}
	if !json.Valid(res.Results) {
		t.Errorf("expected valid JSON results, got %s", res.Results)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", res.Warnings)
	}
}

func TestRunSimpleQuery_DispatchesToDBQ(t *testing.T) {
	// Verify simple queries are dispatched exclusively to logseq.DB.q,
	// never to logseq.DB.datascriptQuery.
	var gotMethod string
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req apiReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		gotMethod = req.Method
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[]`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	_ = httpapi.RunSimpleQuery(context.Background(), c, "(task now)")

	if gotMethod != "logseq.DB.q" {
		t.Errorf("expected method logseq.DB.q, got %q", gotMethod)
	}
}

func TestRunSimpleQuery_Failure(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"broken"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunSimpleQuery(context.Background(), c, "(task now)")

	if res.Error == nil {
		t.Fatal("expected error on failure")
	}
	if string(res.Results) != "null" {
		t.Errorf("expected null results on failure, got %s", res.Results)
	}
	if res.InputKind != "simple" {
		t.Errorf("expected input_kind=simple on failure, got %q", res.InputKind)
	}
}

func TestRunSimpleQuery_AuthFailure(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "bad-token", nil)
	res := httpapi.RunSimpleQuery(context.Background(), c, "[[logseq]]")

	if res.Error == nil {
		t.Fatal("expected error on auth failure")
	}
	if string(res.Results) != "null" {
		t.Errorf("expected null results on auth failure, got %s", res.Results)
	}
}

func TestRunSimpleQuery_NullResponseIsError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, ` null `)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunSimpleQuery(context.Background(), c, "{{query [[logseq]]}}")

	if res.Error == nil {
		t.Fatal("expected error on null response")
	}
	if !strings.Contains(*res.Error, "returned null") {
		t.Errorf("expected error containing 'returned null', got %q", *res.Error)
	}
	if res.QueryMethod != "" {
		t.Errorf("expected empty query_method on null response, got %q", res.QueryMethod)
	}
	if string(res.Results) != "null" {
		t.Errorf("expected null results on null response, got %s", res.Results)
	}
}

func TestRunSimpleQuery_ErrorEnvelopeIsError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":"Missing query property"}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	// Testing arbitrary unsupported string passing
	res := httpapi.RunSimpleQuery(context.Background(), c, "{{query [[logseq]]}}")

	if res.Error == nil {
		t.Fatal("expected error on 200 + error envelope JSON")
	}
	if !strings.Contains(*res.Error, `{"error":"Missing query property"}`) {
		t.Errorf("expected parsed error string, got %q", *res.Error)
	}
	if string(res.Results) != "null" {
		t.Errorf("expected explicitly null results, got %s", res.Results)
	}
}

func TestRunSimpleQuery_ErrorObjectEnvelopeIsError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":{"message":"Internal syntax error"}}`)
	})
	defer srv.Close()

	c := httpapi.NewClient(apiURL(srv), "", nil)
	res := httpapi.RunSimpleQuery(context.Background(), c, "foo")

	if res.Error == nil {
		t.Fatal("expected error on 200 + nested error envelope JSON")
	}
	if !strings.Contains(*res.Error, `{"error":{"message":"Internal syntax error"}}`) {
		t.Errorf("expected parsed error string, got %q", *res.Error)
	}
	if string(res.Results) != "null" {
		t.Errorf("expected explicitly null results, got %s", res.Results)
	}
}


func TestRunSimpleQuery_ExpressionShapes(t *testing.T) {
	// Verify various simple DSL expression shapes are forwarded correctly.
	tests := []struct {
		name string
		expr string
	}{
		{"page-ref", "[[logseq]]"},
		{"task-now", "(task now)"},
		{"and-combinator", "(and [[logseq]] #TypeScript)"},
		{"page-property", "(page-property type project)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotArgs []any
			srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				var req apiReq
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				gotArgs = req.Args
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[]`)
			})
			defer srv.Close()

			c := httpapi.NewClient(apiURL(srv), "", nil)
			res := httpapi.RunSimpleQuery(context.Background(), c, tt.expr)

			if res.Error != nil {
				t.Fatalf("unexpected error: %q", *res.Error)
			}
			if len(gotArgs) != 1 {
				t.Fatalf("expected 1 arg, got %d: %v", len(gotArgs), gotArgs)
			}
			// JSON decoding turns strings into interface{}, so compare as string.
			argStr, ok := gotArgs[0].(string)
			if !ok {
				t.Fatalf("expected string arg, got %T", gotArgs[0])
			}
			if argStr != tt.expr {
				t.Errorf("expected expr %q, got %q", tt.expr, argStr)
			}
		})
	}
}
