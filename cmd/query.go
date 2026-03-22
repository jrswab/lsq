// Package cmd provides the CLI entrypoints for lsq subcommands.
package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jrswab/lsq/query"
	"github.com/jrswab/lsq/query/backend/httpapi"
)

// Default values for query flags.
const (
	DefaultAPIURL     = "http://127.0.0.1:12315/api"
	DefaultTokenEnv   = "LOGSEQ_API_TOKEN"
	DefaultHTTPTimeout = 10 * time.Second
)

// RunQuery is the top-level entrypoint for `lsq query ...`.
//
// It parses the query subcommand and flags, then dispatches to the
// appropriate handler. Returns an exit code (0 = success, nonzero = failure).
//
// args should be the arguments after "query", e.g. for `lsq query doctor --format json`
// args would be ["doctor", "--format", "json"].
func RunQuery(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: lsq query <doctor|advanced|simple> [flags]")
		return 1
	}

	subcommand := args[0]
	flagArgs := args[1:]

	fs := flag.NewFlagSet("lsq query "+subcommand, flag.ContinueOnError)
	fs.SetOutput(stderr)

	backend := fs.String("backend", "auto", "Execution backend: auto, http")
	format := fs.String("format", "text", "Output format: text, json, ndjson")
	apiURL := fs.String("api-url", DefaultAPIURL, "Logseq HTTP API endpoint URL")
	tokenEnv := fs.String("api-token-env", DefaultTokenEnv, "Environment variable name holding the API bearer token")
	queryStr := fs.String("query", "", "Raw advanced query text (for advanced subcommand)")
	queryFile := fs.String("file", "", "Path to file containing query text (for advanced subcommand)")
	expr := fs.String("expr", "", "Raw simple DSL expression (for simple subcommand)")
	_ = fs.Bool("explain", false, "Show verbose diagnostic output (reserved)")

	if err := fs.Parse(flagArgs); err != nil {
		// flag.ContinueOnError: Parse already printed the error to stderr.
		return 1
	}

	// Validate format early.
	switch *format {
	case query.FormatJSON, query.FormatNDJSON, query.FormatText:
		// OK
	default:
		fmt.Fprintf(stderr, "unsupported format %q: must be one of text, json, ndjson\n", *format)
		return 1
	}

	// Dispatch subcommand.
	switch subcommand {
	case "doctor":
		return runDoctor(context.Background(), *format, *backend, *apiURL, *tokenEnv, stdout, stderr)
	case "advanced":
		return runAdvanced(context.Background(), *format, *backend, *apiURL, *tokenEnv, *queryStr, *queryFile, stdout, stderr)
	case "simple":
		return runSimple(context.Background(), *format, *backend, *apiURL, *tokenEnv, *expr, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown query subcommand: %q\nusage: lsq query <doctor|advanced|simple> [flags]\n", subcommand)
		return 1
	}
}

// runDoctor probes the Logseq HTTP API and prints the doctor result.
func runDoctor(ctx context.Context, format, backend, apiURL, tokenEnv string, stdout, stderr io.Writer) int {
	if err := validateHTTPBackend(backend); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Resolve the token from the environment variable.
	token := ""
	if tokenEnv != "" {
		token = os.Getenv(tokenEnv)
	}

	client := httpapi.NewClient(apiURL, token, &http.Client{
		Timeout: DefaultHTTPTimeout,
	})

	result := httpapi.RunDoctor(ctx, client)

	out, err := query.RenderResult(format, result)
	if err != nil {
		fmt.Fprintf(stderr, "render error: %v\n", err)
		return 1
	}

	stdout.Write(out)

	// Exit non-zero if the doctor found an error
	// (API unreachable, auth failed, both methods failed, etc.)
	if result.Error != nil {
		return 1
	}
	return 0
}

// runAdvanced executes a raw advanced query through the HTTP API.
func runAdvanced(ctx context.Context, format, backend, apiURL, tokenEnv, queryStr, queryFile string, stdout, stderr io.Writer) int {
	if err := validateHTTPBackend(backend); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Normalize both inputs: trim whitespace so a blank-only value is
	// treated as absent, matching the file-input trimming behavior.
	queryStr = strings.TrimSpace(queryStr)
	queryFile = strings.TrimSpace(queryFile)

	// Validate input: exactly one of --query or --file is required.
	hasQuery := queryStr != ""
	hasFile := queryFile != ""
	if !hasQuery && !hasFile {
		fmt.Fprintln(stderr, "error: one of --query or --file is required")
		return 1
	}
	if hasQuery && hasFile {
		fmt.Fprintln(stderr, "error: --query and --file are mutually exclusive")
		return 1
	}

	// Read query from file if needed.
	if hasFile {
		data, err := os.ReadFile(queryFile)
		if err != nil {
			fmt.Fprintf(stderr, "error reading query file: %v\n", err)
			return 1
		}
		queryStr = strings.TrimSpace(string(data))
		if queryStr == "" {
			fmt.Fprintln(stderr, "error: query file is empty")
			return 1
		}
	}

	token := ""
	if tokenEnv != "" {
		token = os.Getenv(tokenEnv)
	}

	client := httpapi.NewClient(apiURL, token, &http.Client{
		Timeout: DefaultHTTPTimeout,
	})

	result := httpapi.RunAdvancedQuery(ctx, client, queryStr)

	out, err := query.RenderResult(format, result)
	if err != nil {
		fmt.Fprintf(stderr, "render error: %v\n", err)
		return 1
	}

	stdout.Write(out)

	if result.Error != nil {
		return 1
	}
	return 0
}

// runSimple executes a raw simple DSL expression through the HTTP API.
func runSimple(ctx context.Context, format, backend, apiURL, tokenEnv, expr string, stdout, stderr io.Writer) int {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		fmt.Fprintln(stderr, "error: --expr is required for simple queries")
		return 1
	}

	token := ""
	if tokenEnv != "" {
		token = os.Getenv(tokenEnv)
	}

	client := httpapi.NewClient(apiURL, token, &http.Client{
		Timeout: DefaultHTTPTimeout,
	})

	// Defer dispatch logic out of the command layer and down to the router.
	execHTTP := func(ctx context.Context, expr string) query.AdvancedResult {
		return httpapi.RunSimpleQuery(ctx, client, expr)
	}

	result, err := query.RunSimple(ctx, backend, expr, execHTTP)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	out, err := query.RenderResult(format, result)
	if err != nil {
		fmt.Fprintf(stderr, "render error: %v\n", err)
		return 1
	}

	stdout.Write(out)

	if result.Error != nil {
		return 1
	}
	return 0
}

func validateHTTPBackend(backend string) error {
	switch backend {
	case "auto", "http":
		return nil
	default:
		return fmt.Errorf("unsupported backend %q: must be one of auto, http", backend)
	}
}
