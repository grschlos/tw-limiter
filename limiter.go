package limiter

import (
	"context"
	"github.com/grschlos/tw-limiter/internal"
)

type Result = wheel.Result

var ErrRateLimitExceeded = wheel.ErrRateLimitExceeded

// Limiter is a basic interface for the service.
// Such a design allows for both In-Memory and Redis-based implementations.
type Limiter interface {
	// Allow shows whether the request can proceed for a
	// particular user or app (for instance, IP or API-key).
	// Returns Result or error (if a DB connection is lost, or whatever).
	// Return by value to get rid of unnecessary heap allocations
	Allow(ctx context.Context, key string) (Result, error)

	// Finalization (tickers termination)
	Close() error
}

// Creates new Limiter instance. The implementation is in internal/wheel.
func New(size uint32, rate, maxTokens int64) Limiter {
	return wheel.New(size, rate, maxTokens)
}
