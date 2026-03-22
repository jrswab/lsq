// Package cmd provides the CLI entrypoints for lsq subcommands.
package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
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
		fmt.Fprintln(stderr, "usage: lsq query <doctor|advanced> [flags]")
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

	// Validate backend.
	switch *backend {
	case "auto", "http":
		// Phase 1: "auto" resolves to "http".
	default:
		fmt.Fprintf(stderr, "unsupported backend %q: must be one of auto, http\n", *backend)
		return 1
	}

	// Dispatch subcommand.
	switch subcommand {
	case "doctor":
		return runDoctor(context.Background(), *format, *apiURL, *tokenEnv, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown query subcommand: %q\nusage: lsq query <doctor|advanced> [flags]\n", subcommand)
		return 1
	}
}

// runDoctor probes the Logseq HTTP API and prints the doctor result.
func runDoctor(ctx context.Context, format, apiURL, tokenEnv string, stdout, stderr io.Writer) int {
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
