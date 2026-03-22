package query_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jrswab/lsq/query"
)

// --- helpers ---

func strPtr(s string) *string { return &s }

func mustJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nbody: %s", err, data)
	}
	return m
}

// --- DoctorResult JSON ---

func TestRenderResult_DoctorJSON_AllHealthy(t *testing.T) {
	dr := query.DoctorResult{
		Backend:   "http",
		Command:   "doctor",
		APIURL:    "http://127.0.0.1:12315/api",
		Reachable: true,
		Auth: query.DoctorAuth{
			Configured: true,
			Succeeded:  true,
		},
		Capabilities: query.DoctorCapabilities{
			DBQ:             true,
			DatascriptQuery: true,
		},
		Warnings: []string{},
		Error:    nil,
	}

	out, err := query.RenderResult("json", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := mustJSON(t, out)

	// Verify key fields match spec illustrative output.
	if m["backend"] != "http" {
		t.Errorf("backend=%v, want http", m["backend"])
	}
	if m["command"] != "doctor" {
		t.Errorf("command=%v, want doctor", m["command"])
	}
	if m["api_url"] != "http://127.0.0.1:12315/api" {
		t.Errorf("api_url=%v", m["api_url"])
	}
	if m["reachable"] != true {
		t.Errorf("reachable=%v", m["reachable"])
	}
	if m["error"] != nil {
		t.Errorf("error=%v, want null", m["error"])
	}

	// Warnings should be an empty array, not null.
	warnings, ok := m["warnings"].([]any)
	if !ok || warnings == nil {
		t.Errorf("warnings should be [], got %v", m["warnings"])
	}
	if len(warnings) != 0 {
		t.Errorf("warnings should be empty, got %v", warnings)
	}

	// Auth nested object.
	auth, ok := m["auth"].(map[string]any)
	if !ok {
		t.Fatalf("auth not an object: %v", m["auth"])
	}
	if auth["configured"] != true {
		t.Errorf("auth.configured=%v", auth["configured"])
	}
	if auth["succeeded"] != true {
		t.Errorf("auth.succeeded=%v", auth["succeeded"])
	}

	// Capabilities nested object.
	caps, ok := m["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities not an object: %v", m["capabilities"])
	}
	if caps["db_q"] != true {
		t.Errorf("capabilities.db_q=%v", caps["db_q"])
	}
	if caps["datascript_query"] != true {
		t.Errorf("capabilities.datascript_query=%v", caps["datascript_query"])
	}

	// Must end with newline.
	if out[len(out)-1] != '\n' {
		t.Error("json output should end with newline")
	}
}

// TestRenderResult_DoctorJSON_WithError preserves doctor errors in JSON output.
func TestRenderResult_DoctorJSON_WithError(t *testing.T) {
	dr := query.DoctorResult{
		Backend:  "http",
		Command:  "doctor",
		APIURL:   "http://127.0.0.1:12315/api",
		Warnings: []string{"logseq.DB.q not available"},
		Error:    strPtr("auth failed with status 401"),
	}

	out, err := query.RenderResult("json", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := mustJSON(t, out)

	if m["error"] != "auth failed with status 401" {
		t.Errorf("error=%v", m["error"])
	}
	warnings := m["warnings"].([]any)
	if len(warnings) != 1 || warnings[0] != "logseq.DB.q not available" {
		t.Errorf("warnings=%v", warnings)
	}
}

// TestRenderResult_DoctorJSON_NilWarningsBecomesEmptyArray normalizes nil warnings.
func TestRenderResult_DoctorJSON_NilWarningsBecomesEmptyArray(t *testing.T) {
	// When Warnings is nil (zero-value slice, not pre-initialized),
	// JSON output must still produce "warnings":[] not "warnings":null.
	// This tests the renderer's normalization guarantee, not the caller's discipline.
	dr := query.DoctorResult{
		Backend: "http",
		Command: "doctor",
		// Warnings deliberately omitted — this is the nil case.
	}

	out, err := query.RenderResult("json", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Raw string check: must not contain null.
	if strings.Contains(string(out), `"warnings":null`) {
		t.Errorf("nil warnings must serialize as [], got: %s", out)
	}
	// Parsed check: must be an empty array.
	m := mustJSON(t, out)
	warnings, ok := m["warnings"].([]any)
	if !ok {
		t.Fatalf("warnings is not an array: %T %v", m["warnings"], m["warnings"])
	}
	if len(warnings) != 0 {
		t.Errorf("expected empty warnings array, got %v", warnings)
	}
}

// --- AdvancedResult JSON ---

func TestRenderResult_AdvancedJSON_Success(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:     "http",
		InputKind:   "advanced",
		QueryMethod: "logseq.DB.q",
		Results:     json.RawMessage(`[["page-a"],["page-b"]]`),
		Warnings:    []string{},
		Error:       nil,
	}

	out, err := query.RenderResult("json", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := mustJSON(t, out)
	if m["backend"] != "http" {
		t.Errorf("backend=%v", m["backend"])
	}
	if m["input_kind"] != "advanced" {
		t.Errorf("input_kind=%v", m["input_kind"])
	}
	if m["query_method"] != "logseq.DB.q" {
		t.Errorf("query_method=%v", m["query_method"])
	}
	if m["error"] != nil {
		t.Errorf("error=%v, want null", m["error"])
	}

	results, ok := m["results"].([]any)
	if !ok {
		t.Fatalf("results is not an array: %T %v", m["results"], m["results"])
	}
	if len(results) != 2 {
		t.Errorf("results length=%d, want 2", len(results))
	}
}

// TestRenderResult_AdvancedJSON_EmptyResults keeps empty result arrays intact.
func TestRenderResult_AdvancedJSON_EmptyResults(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:     "http",
		InputKind:   "advanced",
		QueryMethod: "logseq.DB.q",
		Results:     json.RawMessage(`[]`),
		Warnings:    []string{},
	}

	out, err := query.RenderResult("json", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := mustJSON(t, out)
	results := m["results"].([]any)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %v", results)
	}
}

// TestRenderResult_AdvancedJSON_NullResults preserves null results in JSON output.
func TestRenderResult_AdvancedJSON_NullResults(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:     "http",
		InputKind:   "advanced",
		QueryMethod: "",
		Results:     json.RawMessage(`null`),
		Warnings:    []string{},
		Error:       strPtr("both methods failed"),
	}

	out, err := query.RenderResult("json", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := mustJSON(t, out)
	if m["results"] != nil {
		t.Errorf("expected null results, got %v", m["results"])
	}
	if m["error"] != "both methods failed" {
		t.Errorf("error=%v", m["error"])
	}
}

// TestRenderResult_AdvancedJSON_WithFallbackWarning preserves warning context.
func TestRenderResult_AdvancedJSON_WithFallbackWarning(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:     "http",
		InputKind:   "advanced",
		QueryMethod: "logseq.DB.datascriptQuery",
		Results:     json.RawMessage(`[["result"]]`),
		Warnings:    []string{"logseq.DB.q failed, used datascriptQuery fallback"},
	}

	out, err := query.RenderResult("json", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := mustJSON(t, out)
	warnings := m["warnings"].([]any)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

// TestRenderResult_AdvancedJSON_NilWarningsBecomesEmptyArray normalizes nil warnings.
func TestRenderResult_AdvancedJSON_NilWarningsBecomesEmptyArray(t *testing.T) {
	// Same normalization guarantee as DoctorResult: nil Warnings must
	// produce "warnings":[] not "warnings":null.
	ar := query.AdvancedResult{
		Backend:     "http",
		QueryMethod: "logseq.DB.q",
		Results:     json.RawMessage(`[]`),
		// Warnings deliberately omitted — nil case.
	}

	out, err := query.RenderResult("json", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(out), `"warnings":null`) {
		t.Errorf("nil warnings must serialize as [], got: %s", out)
	}
	m := mustJSON(t, out)
	warnings, ok := m["warnings"].([]any)
	if !ok {
		t.Fatalf("warnings is not an array: %T %v", m["warnings"], m["warnings"])
	}
	if len(warnings) != 0 {
		t.Errorf("expected empty warnings array, got %v", warnings)
	}
}

// --- NDJSON ---

func TestRenderResult_DoctorNDJSON(t *testing.T) {
	// Doctor result in ndjson is a single line (same as json).
	dr := query.DoctorResult{
		Backend:   "http",
		Command:   "doctor",
		APIURL:    "http://127.0.0.1:12315/api",
		Reachable: true,
		Warnings:  []string{},
	}

	out, err := query.RenderResult("ndjson", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 ndjson line for doctor, got %d", len(lines))
	}

	// Each line must be valid JSON.
	if !json.Valid([]byte(lines[0])) {
		t.Errorf("ndjson line is not valid JSON: %s", lines[0])
	}
}

// TestRenderResult_AdvancedNDJSON_ExpandsResults emits one NDJSON line per result row.
func TestRenderResult_AdvancedNDJSON_ExpandsResults(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:     "http",
		InputKind:   "advanced",
		QueryMethod: "logseq.DB.q",
		Results:     json.RawMessage(`[["page-a"],["page-b"],{"key":"val"}]`),
		Warnings:    []string{},
	}

	out, err := query.RenderResult("ndjson", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 ndjson lines (one per result), got %d: %v", len(lines), lines)
	}

	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %s", i, line)
		}
	}

	// Verify content of individual lines.
	if lines[0] != `["page-a"]` {
		t.Errorf("line 0: %s", lines[0])
	}
	if lines[2] != `{"key":"val"}` {
		t.Errorf("line 2: %s", lines[2])
	}
}

func TestRenderResult_AdvancedNDJSON_CompactsPrettyPrintedItems(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:   "http",
		InputKind: "advanced",
		Results: json.RawMessage(`[
  {
    "key": "val"
  },
  [
    "nested"
  ]
]`),
		Warnings: []string{},
	}

	out, err := query.RenderResult("ndjson", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 ndjson lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != `{"key":"val"}` {
		t.Fatalf("expected compact object line, got %q", lines[0])
	}
	if lines[1] != `["nested"]` {
		t.Fatalf("expected compact array line, got %q", lines[1])
	}
}

// TestRenderResult_AdvancedNDJSON_ErrorFallsBackToEnvelope keeps error context visible.
func TestRenderResult_AdvancedNDJSON_ErrorFallsBackToEnvelope(t *testing.T) {
	// When there is an error, ndjson emits the full envelope as one line
	// so error/warning context is not lost.
	ar := query.AdvancedResult{
		Backend:  "http",
		Results:  json.RawMessage(`null`),
		Warnings: []string{},
		Error:    strPtr("transport error"),
	}

	out, err := query.RenderResult("ndjson", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 ndjson line for error result, got %d", len(lines))
	}

	m := mustJSON(t, []byte(lines[0]))
	if m["error"] != "transport error" {
		t.Errorf("error=%v", m["error"])
	}
}

// TestRenderResult_AdvancedNDJSON_EmptyResults emits the full envelope for empty arrays.
func TestRenderResult_AdvancedNDJSON_EmptyResults(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:     "http",
		QueryMethod: "logseq.DB.q",
		Results:     json.RawMessage(`[]`),
		Warnings:    []string{},
	}

	out, err := query.RenderResult("ndjson", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty array → no result lines, but we still get the envelope.
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	// 0 result items → falls through to envelope (results is empty).
	if !json.Valid([]byte(lines[0])) {
		t.Errorf("expected valid JSON line, got: %s", lines[0])
	}
}

// TestRenderResult_AdvancedNDJSON_ScalarResult emits the full envelope for scalar results.
func TestRenderResult_AdvancedNDJSON_ScalarResult(t *testing.T) {
	// If results is a scalar (not an array), emit the full envelope.
	ar := query.AdvancedResult{
		Backend:     "http",
		QueryMethod: "logseq.DB.q",
		Results:     json.RawMessage(`42`),
		Warnings:    []string{},
	}

	out, err := query.RenderResult("ndjson", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 ndjson line for scalar result, got %d", len(lines))
	}
}

// TestRenderResult_DoctorNDJSON_NilWarnings normalizes nil warnings in NDJSON output.
func TestRenderResult_DoctorNDJSON_NilWarnings(t *testing.T) {
	// Nil warnings in a doctor ndjson envelope must serialize as [] not null.
	dr := query.DoctorResult{
		Backend: "http",
		APIURL:  "http://127.0.0.1:12315/api",
		// Warnings deliberately omitted — nil case.
	}

	out, err := query.RenderResult("ndjson", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(out), `"warnings":null`) {
		t.Errorf("nil warnings in ndjson envelope must be [], got: %s", out)
	}
	m := mustJSON(t, out[:len(out)-1]) // strip trailing newline
	if _, ok := m["warnings"].([]any); !ok {
		t.Errorf("warnings should be [] in ndjson envelope, got %T %v", m["warnings"], m["warnings"])
	}
}

// TestRenderResult_AdvancedNDJSON_WarningsPreventsExpansion preserves warning context.
func TestRenderResult_AdvancedNDJSON_WarningsPreventsExpansion(t *testing.T) {
	// When AdvancedResult has both results AND warnings, ndjson must emit
	// the full envelope as one line instead of expanding the results array.
	// This ensures warning context (e.g. datascriptQuery fallback) is never
	// silently dropped in the output stream.
	ar := query.AdvancedResult{
		Backend:     "http",
		InputKind:   "advanced",
		QueryMethod: "logseq.DB.datascriptQuery",
		Results:     json.RawMessage(`[["page-a"],["page-b"]]`),
		Warnings:    []string{"logseq.DB.q failed, used datascriptQuery fallback"},
	}

	out, err := query.RenderResult("ndjson", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 envelope line (not expanded results) when warnings present, got %d lines: %v", len(lines), lines)
	}

	// The single line must be the full envelope with warnings preserved.
	m := mustJSON(t, []byte(lines[0]))
	warnings, ok := m["warnings"].([]any)
	if !ok || len(warnings) != 1 {
		t.Errorf("expected 1 warning in ndjson envelope, got %v", m["warnings"])
	}
	if warnings[0] != "logseq.DB.q failed, used datascriptQuery fallback" {
		t.Errorf("unexpected warning text: %v", warnings[0])
	}
	// Results must also be present in the envelope.
	results, ok := m["results"].([]any)
	if !ok || len(results) != 2 {
		t.Errorf("expected 2 results in ndjson envelope, got %v", m["results"])
	}
	if m["query_method"] != "logseq.DB.datascriptQuery" {
		t.Errorf("unexpected query_method: %v", m["query_method"])
	}
}

// --- Text format ---

func TestRenderResult_DoctorText_AllHealthy(t *testing.T) {
	dr := query.DoctorResult{
		Backend:   "http",
		Command:   "doctor",
		APIURL:    "http://127.0.0.1:12315/api",
		Reachable: true,
		Auth: query.DoctorAuth{
			Configured: true,
			Succeeded:  true,
		},
		Capabilities: query.DoctorCapabilities{
			DBQ:             true,
			DatascriptQuery: true,
		},
		Warnings: []string{},
	}

	out, err := query.RenderResult("text", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := string(out)
	for _, want := range []string{
		"Backend:    http",
		"API URL:    http://127.0.0.1:12315/api",
		"Reachable:  true",
		"DB.q:       true",
		"Datascript: true",
		"Auth:       configured, succeeded=true",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text output missing %q\ngot:\n%s", want, text)
		}
	}

	// Should NOT contain Warnings or Error lines when empty/nil.
	if strings.Contains(text, "Warnings:") {
		t.Errorf("should not show Warnings line when empty\ngot:\n%s", text)
	}
	if strings.Contains(text, "Error:") {
		t.Errorf("should not show Error line when nil\ngot:\n%s", text)
	}
}

// TestRenderResult_DoctorText_Unreachable reports transport failures in text output.
func TestRenderResult_DoctorText_Unreachable(t *testing.T) {
	dr := query.DoctorResult{
		Backend:  "http",
		APIURL:   "http://127.0.0.1:12315/api",
		Warnings: []string{},
		Error:    strPtr("transport error: connection refused"),
	}

	out, err := query.RenderResult("text", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := string(out)
	if !strings.Contains(text, "Reachable:  false") {
		t.Errorf("expected Reachable: false, got:\n%s", text)
	}
	if !strings.Contains(text, "Error:") {
		t.Errorf("expected Error line, got:\n%s", text)
	}
	if !strings.Contains(text, "Auth:       not configured") {
		t.Errorf("expected 'not configured' auth, got:\n%s", text)
	}
}

// TestRenderResult_DoctorText_WithWarnings renders warning lines in text output.
func TestRenderResult_DoctorText_WithWarnings(t *testing.T) {
	dr := query.DoctorResult{
		Backend:   "http",
		APIURL:    "http://127.0.0.1:12315/api",
		Reachable: true,
		Warnings:  []string{"warn1", "warn2"},
	}

	out, err := query.RenderResult("text", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := string(out)
	if !strings.Contains(text, "Warnings:   warn1; warn2") {
		t.Errorf("expected warnings, got:\n%s", text)
	}
}

// TestRenderResult_AdvancedText_Success pretty-prints successful advanced results.
func TestRenderResult_AdvancedText_Success(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:     "http",
		QueryMethod: "logseq.DB.q",
		Results:     json.RawMessage(`[["page-a"],["page-b"]]`),
		Warnings:    []string{},
	}

	out, err := query.RenderResult("text", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := string(out)
	// Text format pretty-prints the results.
	if !strings.Contains(text, "page-a") {
		t.Errorf("expected results in text, got:\n%s", text)
	}
	// Should not include Warnings line when empty.
	if strings.Contains(text, "Warnings:") {
		t.Errorf("should not show Warnings line when empty\ngot:\n%s", text)
	}
}

func TestRenderResult_AdvancedText_WhitespaceNullResults(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:  "http",
		Results:  json.RawMessage(" \n null \t "),
		Warnings: []string{"kept"},
	}

	out, err := query.RenderResult("text", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(out), "null") {
		t.Fatalf("did not expect null payload to be rendered, got %q", string(out))
	}
	if !strings.Contains(string(out), "Warnings: kept") {
		t.Fatalf("expected warnings to remain visible, got %q", string(out))
	}
}

// TestRenderResult_AdvancedText_Error reports advanced execution failures.
func TestRenderResult_AdvancedText_Error(t *testing.T) {
	ar := query.AdvancedResult{
		Backend:  "http",
		Results:  json.RawMessage(`null`),
		Warnings: []string{"fallback attempted"},
		Error:    strPtr("both methods failed"),
	}

	out, err := query.RenderResult("text", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := string(out)
	if !strings.Contains(text, "Error: both methods failed") {
		t.Errorf("expected error line, got:\n%s", text)
	}
	if !strings.Contains(text, "Warnings: fallback attempted") {
		t.Errorf("expected warnings line, got:\n%s", text)
	}
}

// --- Unsupported format ---

func TestRenderResult_UnsupportedFormat(t *testing.T) {
	dr := query.DoctorResult{Warnings: []string{}}
	_, err := query.RenderResult("xml", dr)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestRenderResult_UnsupportedTypeJSON rejects unsupported result types for JSON output.
func TestRenderResult_UnsupportedTypeJSON(t *testing.T) {
	if _, err := query.RenderResult("json", struct{ Name string }{"bad"}); err == nil {
		t.Fatal("expected unsupported result type error")
	}
}

// TestRenderResult_UnsupportedTypeNDJSON rejects unsupported result types for NDJSON output.
func TestRenderResult_UnsupportedTypeNDJSON(t *testing.T) {
	if _, err := query.RenderResult("ndjson", []string{"bad"}); err == nil {
		t.Fatal("expected unsupported result type error")
	}
}

// --- Pointer receivers ---

func TestRenderResult_PointerDoctorResult(t *testing.T) {
	dr := &query.DoctorResult{
		Backend:  "http",
		Warnings: []string{},
	}
	out, err := query.RenderResult("json", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(out) {
		t.Errorf("expected valid JSON, got: %s", out)
	}
}

// TestRenderResult_PointerAdvancedResult accepts pointer inputs for advanced results.
func TestRenderResult_PointerAdvancedResult(t *testing.T) {
	ar := &query.AdvancedResult{
		Backend:  "http",
		Results:  json.RawMessage(`[]`),
		Warnings: []string{},
	}
	out, err := query.RenderResult("json", ar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(out) {
		t.Errorf("expected valid JSON, got: %s", out)
	}
}

// --- Deterministic output ---

func TestRenderResult_JSONKeyOrder(t *testing.T) {
	// Verify that JSON field order matches struct declaration order
	// (Go's json.Marshal guarantees this for structs).
	dr := query.DoctorResult{
		Backend:  "http",
		Command:  "doctor",
		APIURL:   "http://127.0.0.1:12315/api",
		Warnings: []string{},
	}

	out, err := query.RenderResult("json", dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := string(out)
	backendIdx := strings.Index(s, `"backend"`)
	commandIdx := strings.Index(s, `"command"`)
	apiURLIdx := strings.Index(s, `"api_url"`)
	warningsIdx := strings.Index(s, `"warnings"`)
	errorIdx := strings.Index(s, `"error"`)

	if backendIdx > commandIdx || commandIdx > apiURLIdx || apiURLIdx > warningsIdx || warningsIdx > errorIdx {
		t.Errorf("JSON keys not in expected order:\n%s", s)
	}
}
