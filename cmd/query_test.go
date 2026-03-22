package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jrswab/lsq/cmd"
)

func TestRunQuery_DoctorJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if m["reachable"] != true {
		t.Errorf("reachable=%v", m["reachable"])
	}
	if m["command"] != "doctor" {
		t.Errorf("command=%v", m["command"])
	}
}

func TestRunQuery_DoctorNDJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "ndjson", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 ndjson line, got %d", len(lines))
	}
	if !json.Valid([]byte(lines[0])) {
		t.Errorf("not valid JSON: %s", lines[0])
	}
}

func TestRunQuery_DoctorText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "text", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Reachable:  true") {
		t.Errorf("missing 'Reachable:  true' in output:\n%s", stdout.String())
	}
}

func TestRunQuery_DoctorDefaultFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	// No --format flag → should default to text.
	code := cmd.RunQuery(
		[]string{"doctor", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Reachable:") {
		t.Errorf("expected text output (default format), got:\n%s", stdout.String())
	}
}

func TestRunQuery_UnsupportedFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--format", "xml"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unsupported format") {
		t.Errorf("expected format error in stderr, got: %s", stderr.String())
	}
}

func TestRunQuery_UnsupportedBackend(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--backend", "file"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unsupported backend") {
		t.Errorf("expected backend error in stderr, got: %s", stderr.String())
	}
}

func TestRunQuery_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery([]string{}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage in stderr, got: %s", stderr.String())
	}
}

func TestRunQuery_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"magic"},
		&stdout, &stderr,
	)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown query subcommand") {
		t.Errorf("expected unknown subcommand error, got: %s", stderr.String())
	}
}

func TestRunQuery_BackendAutoResolvesToHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `42`)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := cmd.RunQuery(
		[]string{"doctor", "--backend", "auto", "--format", "json", "--api-url", srv.URL + "/api"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
	}

	var m map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Backend in result is always "http" in Phase 1.
	if m["backend"] != "http" {
		t.Errorf("expected backend=http, got %v", m["backend"])
	}
}
