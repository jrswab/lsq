package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jrswab/lsq/query"
)

// RunDoctor probes the Logseq HTTP API and returns a structured doctor result.
//
// Reachability classification:
//   - reachable=false only for TransportError (connection refused, DNS, timeout)
//   - reachable=true for AuthError, MethodError, or any successful response
//
// This distinguishes "API unreachable" from "API reachable but query method
// unavailable/broken".
func RunDoctor(ctx context.Context, c *Client) query.DoctorResult {
	res := query.DoctorResult{
		Backend:  "http",
		Command:  "doctor",
		APIURL:   c.BaseURL,
		Warnings: []string{},
	}

	res.Auth.Configured = c.Token != ""

	// Probe DB.q
	dbqErr := c.ProbeDBQ(ctx)
	if dbqErr != nil {
		// Classify: transport → unreachable, auth → reachable but auth failed.
		if isTransport(dbqErr) {
			// DB.q hit a transport error. Try datascriptQuery to double-check
			// (in case it's a flaky connection, both should fail the same way).
			dsErr := c.ProbeDatascriptQuery(ctx)
			if dsErr != nil {
				if isTransport(dsErr) {
					// Both transport failures → unreachable.
					msg := dbqErr.Error()
					res.Error = &msg
					return res
				}
				// DB.q got transport error but datascript didn't — unusual.
				// The API is reachable (datascript got a real HTTP response).
				res.Reachable = true
				classifyAuth(dsErr, &res)
				if res.Error != nil {
					return res
				}
				// datascriptQuery reached the server but failed at method layer.
				msg := bothFailedMsg(dbqErr, dsErr)
				res.Error = &msg
				return res
			}
			// datascriptQuery succeeded → API reachable, DB.q is not.
			res.Reachable = true
			setAuthSucceeded(&res)
			res.Capabilities.DatascriptQuery = true
			res.Warnings = append(res.Warnings, "logseq.DB.q is not available; logseq.DB.datascriptQuery is available as fallback")
			return res
		}

		// DB.q returned a non-transport error → API is reachable.
		res.Reachable = true

		if isAuth(dbqErr) {
			res.Auth.Succeeded = false
			msg := dbqErr.Error()
			res.Error = &msg
			return res
		}

		// DB.q failed at method layer. Try datascriptQuery.
		dsErr := c.ProbeDatascriptQuery(ctx)
		if dsErr != nil {
			if isAuth(dsErr) {
				res.Auth.Succeeded = false
				msg := dsErr.Error()
				res.Error = &msg
				return res
			}
			// Both methods failed at method layer → reachable but no capabilities.
			setAuthSucceeded(&res)
			msg := bothFailedMsg(dbqErr, dsErr)
			res.Error = &msg
			return res
		}
		// datascriptQuery succeeded.
		setAuthSucceeded(&res)
		res.Capabilities.DatascriptQuery = true
		res.Warnings = append(res.Warnings, "logseq.DB.q is not available; logseq.DB.datascriptQuery is available as fallback")
		return res
	}

	// DB.q succeeded.
	res.Reachable = true
	setAuthSucceeded(&res)
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

// --- helpers ---

func isTransport(err error) bool {
	var te *TransportError
	return errors.As(err, &te)
}

func isAuth(err error) bool {
	var ae *AuthError
	return errors.As(err, &ae)
}

// classifyAuth updates result auth fields when an auth error is detected.
func classifyAuth(err error, res *query.DoctorResult) {
	if isAuth(err) {
		res.Auth.Succeeded = false
		msg := err.Error()
		res.Error = &msg
	}
}

// setAuthSucceeded marks auth.succeeded=true only when a token is configured.
func setAuthSucceeded(res *query.DoctorResult) {
	if res.Auth.Configured {
		res.Auth.Succeeded = true
	}
}

// bothFailedMsg produces an actionable error message when both query
// methods fail. It includes the underlying error text from each method
// so the user can distinguish "method not supported" from "server error"
// without having to re-run with --explain or debug logging.
func bothFailedMsg(dbqErr, dsErr error) string {
	return fmt.Sprintf(
		"both query methods failed: logseq.DB.q: %s; logseq.DB.datascriptQuery: %s",
		dbqErr, dsErr,
	)
}
