// Package httpapi implements the Logseq HTTP API transport for query execution.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// apiRequest is the JSON body sent to POST /api.
type apiRequest struct {
	Method string `json:"method"`
	Args   []any  `json:"args"`
}

// Client talks to the local Logseq HTTP API.
//
// BaseURL is the full API endpoint URL including the path, e.g.
// "http://127.0.0.1:12315/api". DoRaw posts directly to this URL
// without appending any path suffix. This matches the spec's --api-url
// flag semantic and the illustrative doctor output
// ("api_url": "http://127.0.0.1:12315/api").
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient returns a Client with sensible defaults.
// baseURL must be the full API endpoint (e.g. "http://127.0.0.1:12315/api").
// If httpClient is nil, http.DefaultClient is used.
func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: httpClient,
	}
}

// DoRaw sends an API call and returns the raw JSON response body.
// It returns an error for transport failures, non-200 status codes,
// and non-JSON responses. Transport-level errors (connection refused,
// DNS failure, timeout) are wrapped in TransportError so callers can
// distinguish reachability from method-layer failures.
func (c *Client) DoRaw(ctx context.Context, method string, args []any) (json.RawMessage, error) {
	body, err := json.Marshal(apiRequest{Method: method, Args: args})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &TransportError{Cause: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &TransportError{Cause: fmt.Errorf("read response: %w", err)}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{StatusCode: resp.StatusCode}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &MethodError{
			StatusCode: resp.StatusCode,
			Body:       truncate(raw, 200),
		}
	}

	// Validate that the response is valid JSON.
	if !json.Valid(raw) {
		return nil, &MethodError{
			StatusCode: resp.StatusCode,
			Body:       truncate(raw, 200),
		}
	}

	return json.RawMessage(raw), nil
}

// ProbeDBQ checks whether logseq.DB.q is available by running a minimal query.
//
// A probe succeeds only when DoRaw returns valid JSON AND the response does
// not look like a Logseq error envelope (see isProbeSuccess). This prevents
// false positives when the server returns 200 + {"error": "..."}.
func (c *Client) ProbeDBQ(ctx context.Context) error {
	raw, err := c.DoRaw(ctx, "logseq.DB.q", []any{"[[logseq]]"})
	if err != nil {
		return err
	}
	if !isProbeSuccess(raw) {
		return &MethodError{StatusCode: 200, Body: truncate(raw, 200)}
	}
	return nil
}

// ProbeDatascriptQuery checks whether logseq.DB.datascriptQuery is available.
func (c *Client) ProbeDatascriptQuery(ctx context.Context) error {
	raw, err := c.DoRaw(ctx, "logseq.DB.datascriptQuery", []any{"[:find ?e . :where [?e :block/uuid]]"})
	if err != nil {
		return err
	}
	if !isProbeSuccess(raw) {
		return &MethodError{StatusCode: 200, Body: truncate(raw, 200)}
	}
	return nil
}

// isProbeSuccess applies the narrowest defensible validation for a probe
// response. The Logseq HTTP API does not have a formally documented
// success/error envelope, so we use a conservative heuristic.
//
// Decision table (applied in order):
//
//	| Response shape                              | Result  | Rationale
//	|---------------------------------------------|---------|------------------------------------------
//	| empty body                                  | FAIL    | no data returned
//	| non-object (array, number, string, bool)    | SUCCESS | legitimate query results
//	| unparseable object                           | FAIL    | corrupted response
//	| {"error": "non-empty string"}               | FAIL    | string error envelope
//	| {"error": {"message": "..."}}               | FAIL    | structured error object
//	| {"error": 123} (non-null, non-string)       | FAIL    | non-string error value
//	| {"error": ""} or {"error": null}            | SUCCESS | empty/null error = no error
//	| {"ok": false, ...} or {"success": false, ...}| FAIL   | boolean failure indicator
//	| object without "error" key                  | SUCCESS | legitimate object result
//
// This heuristic is intentionally isolated so it can be revised after
// real Logseq API validation (see ValidateAgainstReal).
//
// TODO(real-api): After testing against a real Logseq instance, refine
// this function based on observed success/error response shapes.
func isProbeSuccess(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return false
	}
	if bytes.Equal(raw, []byte("null")) {
		return false
	}
	// Non-object responses (arrays, scalars) are valid query results.
	if raw[0] != '{' {
		return true
	}

	// Parse the object into a generic map so we can inspect the "error"
	// value regardless of its JSON type (string, object, number, etc.).
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}

	// Check for boolean failure indicators: {"ok":false} or {"success":false}.
	if boolFlagFalse(obj, "ok") || boolFlagFalse(obj, "success") {
		return false
	}

	// Check the "error" field if present.
	errRaw, hasError := obj["error"]
	if !hasError {
		return true
	}

	// null → no error.
	if string(errRaw) == "null" {
		return true
	}

	// Try to unmarshal as string. Empty string → no error.
	var errStr string
	if json.Unmarshal(errRaw, &errStr) == nil {
		return errStr == ""
	}

	// Error is present and is neither null nor an empty string
	// (could be an object like {"message":"..."}, a number, etc.).
	// Treat any non-null, non-empty-string "error" value as failure.
	return false
}

// boolFlagFalse returns true if obj[key] exists and is the JSON boolean false.
func boolFlagFalse(obj map[string]json.RawMessage, key string) bool {
	raw, ok := obj[key]
	if !ok {
		return false
	}
	var val bool
	if json.Unmarshal(raw, &val) != nil {
		return false
	}
	return !val
}

// TransportError indicates a connectivity-level failure (connection refused,
// DNS error, timeout). Callers can use errors.As to distinguish transport
// failures from method-layer or auth errors.
type TransportError struct {
	Cause error
}

// Error formats the transport failure for display.
func (e *TransportError) Error() string {
	return fmt.Sprintf("transport error: %v", e.Cause)
}

// Unwrap returns the underlying network error.
func (e *TransportError) Unwrap() error {
	return e.Cause
}

// AuthError indicates the API returned a 401 or 403 status.
type AuthError struct {
	StatusCode int
}

// Error formats the authentication failure for display.
func (e *AuthError) Error() string {
	return fmt.Sprintf("auth failed with status %d", e.StatusCode)
}

// MethodError indicates the API was reachable but the request failed at
// the method layer (non-200 status, invalid JSON, or error envelope).
type MethodError struct {
	StatusCode int
	Body       string
}

// Error formats the method-layer failure for display.
func (e *MethodError) Error() string {
	return fmt.Sprintf("method error (status %d): %s", e.StatusCode, e.Body)
}

// ValidateAgainstReal documents the checks that must be performed against a
// real Logseq HTTP API instance to confirm the probe heuristic is correct.
// This function is a no-op in production; it serves as an explicit seam
// for future runtime validation and as a living checklist.
//
// When validating against a real Logseq instance, verify:
//
// TODO(real-api): 1. What HTTP status does Logseq return for an unsupported
//
//	method name? (e.g. "logseq.DB.nonexistent") — expected: 200 + error
//	envelope, or non-200?
//
// TODO(real-api): 2. What is the exact JSON shape of a successful DB.q
//
//	response? Is it always an array, or can it be a scalar/object?
//
// TODO(real-api): 3. What is the exact JSON shape of a DB.q error response?
//
//	Is it {"error":"string"}, {"error":{"message":"..."}}, or something
//	else?
//
// TODO(real-api): 4. Does logseq.DB.datascriptQuery use the same
//
//	request/response envelope as logseq.DB.q?
//
// TODO(real-api): 5. Does the Logseq API require auth by default, or only
//
//	when explicitly enabled? What status code is returned on auth failure?
//
// TODO(real-api): 6. Is the probe query [:find ?e . :where [?e :block/uuid]]
//
//	safe and fast on large graphs? Should we use a different probe query?
func ValidateAgainstReal() {
	// Intentional no-op. This function exists as a documentation seam.
	// After real-instance testing, the TODOs above should be resolved and
	// isProbeSuccess updated accordingly.
}

// truncate returns at most n bytes of b as a string.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
