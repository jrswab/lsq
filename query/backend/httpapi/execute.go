package httpapi

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jrswab/lsq/query"
)

// RunDoctor probes the Logseq HTTP API and returns a structured doctor result.
func RunDoctor(ctx context.Context, c *Client) query.DoctorResult {
	res := query.DoctorResult{
		Backend:  "http",
		Command:  "doctor",
		APIURL:   c.BaseURL,
		Warnings: []string{},
	}

	res.Auth.Configured = c.Token != ""

	// Probe reachability + DB.q
	if err := c.ProbeDBQ(ctx); err != nil {
		var authErr *AuthError
		if errors.As(err, &authErr) {
			res.Reachable = true
			res.Auth.Succeeded = false
			msg := authErr.Error()
			res.Error = &msg
			return res
		}
		// Check if datascriptQuery works (maybe the q method is unsupported,
		// but the connection is fine).
		if err2 := c.ProbeDatascriptQuery(ctx); err2 != nil {
			// Both failed — likely unreachable or something else.
			var authErr2 *AuthError
			if errors.As(err2, &authErr2) {
				res.Reachable = true
				res.Auth.Succeeded = false
				msg := authErr2.Error()
				res.Error = &msg
				return res
			}
			msg := err.Error()
			res.Error = &msg
			return res
		}
		// datascriptQuery succeeded, so API is reachable but DB.q is not available.
		res.Reachable = true
		if res.Auth.Configured {
			res.Auth.Succeeded = true
		}
		res.Capabilities.DatascriptQuery = true
		res.Warnings = append(res.Warnings, "logseq.DB.q is not available; logseq.DB.datascriptQuery is available as fallback")
		return res
	}

	// DB.q succeeded.
	res.Reachable = true
	if res.Auth.Configured {
		res.Auth.Succeeded = true
	}
	res.Capabilities.DBQ = true

	// Also probe datascriptQuery for completeness.
	if err := c.ProbeDatascriptQuery(ctx); err == nil {
		res.Capabilities.DatascriptQuery = true
	}

	return res
}

// RunAdvancedQuery executes a raw advanced query through the HTTP API.
// It tries logseq.DB.q first and falls back to logseq.DB.datascriptQuery.
func RunAdvancedQuery(ctx context.Context, c *Client, queryStr string) query.AdvancedResult {
	res := query.AdvancedResult{
		Backend:   "http",
		InputKind: "advanced",
		Warnings:  []string{},
	}

	// Try DB.q first.
	raw, err := c.DoRaw(ctx, "logseq.DB.q", []any{queryStr})
	if err == nil {
		res.QueryMethod = "logseq.DB.q"
		res.Results = raw
		return res
	}

	// Fallback to datascriptQuery.
	raw, err2 := c.DoRaw(ctx, "logseq.DB.datascriptQuery", []any{queryStr})
	if err2 == nil {
		res.QueryMethod = "logseq.DB.datascriptQuery"
		res.Results = raw
		res.Warnings = append(res.Warnings, "logseq.DB.q failed, used datascriptQuery fallback")
		return res
	}

	// Both failed.
	msg := err.Error()
	res.Error = &msg
	res.Results = json.RawMessage("null")
	return res
}
