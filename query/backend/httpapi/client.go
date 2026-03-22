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
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient returns a Client with sensible defaults.
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
// and non-JSON responses.
func (c *Client) DoRaw(ctx context.Context, method string, args []any) (json.RawMessage, error) {
	body, err := json.Marshal(apiRequest{Method: method, Args: args})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{StatusCode: resp.StatusCode}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(raw))
	}

	// Validate that the response is valid JSON.
	if !json.Valid(raw) {
		return nil, fmt.Errorf("response is not valid JSON: %s", truncate(raw, 200))
	}

	return json.RawMessage(raw), nil
}

// ProbeDBQ checks whether logseq.DB.q is available by running a minimal query.
func (c *Client) ProbeDBQ(ctx context.Context) error {
	_, err := c.DoRaw(ctx, "logseq.DB.q", []any{"[:find ?e . :where [?e :block/uuid]]"})
	return err
}

// ProbeDatascriptQuery checks whether logseq.DB.datascriptQuery is available.
func (c *Client) ProbeDatascriptQuery(ctx context.Context) error {
	_, err := c.DoRaw(ctx, "logseq.DB.datascriptQuery", []any{"[:find ?e . :where [?e :block/uuid]]"})
	return err
}

// AuthError indicates the API returned a 401 or 403 status.
type AuthError struct {
	StatusCode int
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth failed with status %d", e.StatusCode)
}

// truncate returns at most n bytes of b as a string.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
