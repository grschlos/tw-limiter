package limiter

import (
	"context"
	"errors"
	"github.com/grschlos/tw-limiter/internal/wheel"
)

type Strategy int

const (
	StrategyMemory Strategy = iota
	StrategyXDP
)

var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrNotSupported      = errors.New("strategy not supported on this OS")
)

// Use Type Aliasing to keep Result visible through the root package
type Result = wheel.Result

type Limiter interface {
	Allow(ctx context.Context, key string) (Result, error)
	Close() error
}

// New is the only function the user needs to call.
func New(strategy Strategy, size uint32, rate, max int64) (Limiter, error) {
	switch strategy {
	case StrategyMemory:
		return wheel.New(size, rate, max), nil
	case StrategyXDP:
		return wheel.NewXDP(size, rate, max)
	default:
		return nil, errors.New("unknown strategy")
	}
}
