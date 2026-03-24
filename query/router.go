package query

import (
	"context"
	"fmt"
)

// HTTPSimpleExecutor defines the execution contract for raw simple DSL queries via HTTP.
// This func type is injected to preserve the separation of routing logic while
// avoiding an import cycle between the query and httpapi packages.
type HTTPSimpleExecutor func(ctx context.Context, expr string) AdvancedResult

// RunSimple evaluates the backend policy and routes raw simple queries to the execution engine.
//
// HTTP-only routing mechanics:
//   - "auto" resolves to the HTTP executor
//   - "http" routes to the HTTP executor
//   - "file" or any other value yields an error
func RunSimple(ctx context.Context, backend string, expr string, execHTTP HTTPSimpleExecutor) (AdvancedResult, error) {
	switch backend {
	case "auto", "http":
		if execHTTP == nil {
			return AdvancedResult{}, fmt.Errorf("http execution engine not provided")
		}
		// Dispatch to the correct backend function (provided as a closure, e.g., httpapi.RunSimpleQuery)
		return execHTTP(ctx, expr), nil
	default:
		return AdvancedResult{}, fmt.Errorf("unsupported backend %q for simple query: must be one of auto, http", backend)
	}
}
