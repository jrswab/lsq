package query

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Supported output format names.
const (
	FormatJSON   = "json"
	FormatNDJSON = "ndjson"
	FormatText   = "text"
)

// DoctorAuth reports the authentication status.
type DoctorAuth struct {
	Configured bool `json:"configured"`
	Succeeded  bool `json:"succeeded"`
}

// DoctorCapabilities reports which query methods are available.
type DoctorCapabilities struct {
	DBQ             bool `json:"db_q"`
	DatascriptQuery bool `json:"datascript_query"`
}

// DoctorResult is the structured output of `lsq query doctor`.
//
// JSON field names and types match the spec's illustrative output:
//
//	{
//	  "backend": "http",
//	  "command": "doctor",
//	  "api_url": "http://127.0.0.1:12315/api",
//	  "reachable": true,
//	  "auth":         { "configured": true, "succeeded": true },
//	  "capabilities": { "db_q": true, "datascript_query": true },
//	  "warnings": [],
//	  "error": null
//	}
type DoctorResult struct {
	Backend      string             `json:"backend"`
	Command      string             `json:"command"`
	APIURL       string             `json:"api_url"`
	Reachable    bool               `json:"reachable"`
	Auth         DoctorAuth         `json:"auth"`
	Capabilities DoctorCapabilities `json:"capabilities"`
	Warnings     []string           `json:"warnings"`
	Error        *string            `json:"error"`
}

// AdvancedResult is the structured output of `lsq query advanced`.
//
// JSON field names and types match the spec's illustrative output:
//
//	{
//	  "backend": "http",
//	  "input_kind": "advanced",
//	  "query_method": "logseq.DB.q",
//	  "results": [],
//	  "warnings": [],
//	  "error": null
//	}
type AdvancedResult struct {
	Backend     string          `json:"backend"`
	InputKind   string          `json:"input_kind"`
	QueryMethod string          `json:"query_method"`
	Results     json.RawMessage `json:"results"`
	Warnings    []string        `json:"warnings"`
	Error       *string         `json:"error"`
}

// RenderResult serializes a result value into the requested output format.
//
// Supported formats:
//   - "json":   compact single-line JSON with deterministic key order
//   - "ndjson": newline-delimited JSON; for AdvancedResult the results array
//     is expanded into one JSON value per line; for all other types
//     the behavior is identical to "json" (one object = one line)
//   - "text":   human-readable summary; format varies by result type
//
// The result argument must be one of the known result types (DoctorResult,
// AdvancedResult) or a pointer to one. Returns an error for unsupported
// format names or unrecognized result types.
func RenderResult(format string, result any) ([]byte, error) {
	switch format {
	case FormatJSON:
		return renderJSON(result)
	case FormatNDJSON:
		return renderNDJSON(result)
	case FormatText:
		return renderText(result)
	default:
		return nil, fmt.Errorf("unsupported format %q: must be one of json, ndjson, text", format)
	}
}

// --- JSON ---

// renderJSON returns compact JSON with a trailing newline.
// Go's json.Marshal produces deterministic key order (struct field order).
func renderJSON(result any) ([]byte, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}
	return append(b, '\n'), nil
}

// --- NDJSON ---

// renderNDJSON returns newline-delimited JSON.
//
// For AdvancedResult: the top-level results array is expanded so each
// element is emitted as a separate JSON line. This is the primary use
// case for ndjson in query output — piping individual results to jq or
// other line-oriented tools.
//
// For all other result types: the entire result is emitted as a single
// newline-terminated JSON line (equivalent to json format).
func renderNDJSON(result any) ([]byte, error) {
	switch v := result.(type) {
	case AdvancedResult:
		return advancedNDJSON(v)
	case *AdvancedResult:
		if v == nil {
			return []byte("null\n"), nil
		}
		return advancedNDJSON(*v)
	default:
		return renderJSON(result)
	}
}

func advancedNDJSON(r AdvancedResult) ([]byte, error) {
	// If there is an error or results is null/empty, emit the envelope
	// as a single line so the error/warning context is not lost.
	if r.Error != nil || len(r.Results) == 0 || string(r.Results) == "null" {
		return renderJSON(r)
	}

	// Try to expand the results array into individual lines.
	var items []json.RawMessage
	if err := json.Unmarshal(r.Results, &items); err != nil {
		// results is not an array — emit the whole envelope as one line.
		return renderJSON(r)
	}

	// Zero items in the array — fall back to full envelope so the
	// consumer always gets at least one line of output.
	if len(items) == 0 {
		return renderJSON(r)
	}

	var buf []byte
	for _, item := range items {
		compact, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("ndjson item marshal: %w", err)
		}
		buf = append(buf, compact...)
		buf = append(buf, '\n')
	}
	return buf, nil
}

// --- Text ---

// renderText returns a human-readable summary.
func renderText(result any) ([]byte, error) {
	switch v := result.(type) {
	case DoctorResult:
		return doctorText(v), nil
	case *DoctorResult:
		if v == nil {
			return []byte("(nil doctor result)\n"), nil
		}
		return doctorText(*v), nil
	case AdvancedResult:
		return advancedText(v), nil
	case *AdvancedResult:
		if v == nil {
			return []byte("(nil advanced result)\n"), nil
		}
		return advancedText(*v), nil
	default:
		return nil, fmt.Errorf("unsupported result type %T for text format", result)
	}
}

func doctorText(r DoctorResult) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Backend:    %s\n", r.Backend))
	sb.WriteString(fmt.Sprintf("API URL:    %s\n", r.APIURL))
	sb.WriteString(fmt.Sprintf("Reachable:  %t\n", r.Reachable))

	if r.Auth.Configured {
		sb.WriteString(fmt.Sprintf("Auth:       configured, succeeded=%t\n", r.Auth.Succeeded))
	} else {
		sb.WriteString("Auth:       not configured\n")
	}

	sb.WriteString(fmt.Sprintf("DB.q:       %t\n", r.Capabilities.DBQ))
	sb.WriteString(fmt.Sprintf("Datascript: %t\n", r.Capabilities.DatascriptQuery))

	if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("Warnings:   %s\n", strings.Join(r.Warnings, "; ")))
	}
	if r.Error != nil {
		sb.WriteString(fmt.Sprintf("Error:      %s\n", *r.Error))
	}
	return []byte(sb.String())
}

func advancedText(r AdvancedResult) []byte {
	var sb strings.Builder

	if r.Error != nil {
		sb.WriteString(fmt.Sprintf("Error: %s\n", *r.Error))
		if len(r.Warnings) > 0 {
			sb.WriteString(fmt.Sprintf("Warnings: %s\n", strings.Join(r.Warnings, "; ")))
		}
		return []byte(sb.String())
	}

	// For text output, pretty-print the results JSON.
	if len(r.Results) > 0 && string(r.Results) != "null" {
		var pretty json.RawMessage
		if json.Unmarshal(r.Results, &pretty) == nil {
			indented, err := json.MarshalIndent(pretty, "", "  ")
			if err == nil {
				sb.Write(indented)
				sb.WriteByte('\n')
			} else {
				sb.Write(r.Results)
				sb.WriteByte('\n')
			}
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("Warnings: %s\n", strings.Join(r.Warnings, "; ")))
	}

	return []byte(sb.String())
}
