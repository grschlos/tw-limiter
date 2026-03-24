package limiter

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// Result is a Limit Check result.
// Useful for data transmission via HTTP-headers (X-RateLimit-Limit)
type Result struct {
	Allowed    bool          // Is the request allowed
	Remaining  int           // How many tokens left in the current time window
	ResetAfter time.Duration // Time before the limit is reset
}

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
