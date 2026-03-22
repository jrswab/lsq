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
	raw, err := c.DoRaw(ctx, "logseq.DB.q", []any{"[:find ?e . :where [?e :block/uuid]]"})
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
// success/error envelope, so we use a conservative heuristic:
//
//   - If the response is a JSON object with a top-level "error" key whose
//     value is a non-null, non-empty string, treat it as failure.
//   - Everything else (arrays, scalars, objects without "error") is treated
//     as success.
//
// This catches the common pattern where Logseq returns {"error":"method not found"}
// while still accepting legitimate scalar/array results.
func isProbeSuccess(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	// Only inspect JSON objects; arrays and scalars are valid results.
	if raw[0] != '{' {
		return true
	}
	var envelope struct {
		Error *string `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false
	}
	// A non-nil, non-empty error string means failure.
	if envelope.Error != nil && *envelope.Error != "" {
		return false
	}
	return true
}

// TransportError indicates a connectivity-level failure (connection refused,
// DNS error, timeout). Callers can use errors.As to distinguish transport
// failures from method-layer or auth errors.
type TransportError struct {
	Cause error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("transport error: %v", e.Cause)
}

func (e *TransportError) Unwrap() error {
	return e.Cause
}

// AuthError indicates the API returned a 401 or 403 status.
type AuthError struct {
	StatusCode int
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth failed with status %d", e.StatusCode)
}

// MethodError indicates the API was reachable but the request failed at
// the method layer (non-200 status, invalid JSON, or error envelope).
type MethodError struct {
	StatusCode int
	Body       string
}

func (e *MethodError) Error() string {
	return fmt.Sprintf("method error (status %d): %s", e.StatusCode, e.Body)
}

// truncate returns at most n bytes of b as a string.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
