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

type Config struct {
	Strategy  Strategy
	IfaceName string // Required for StrategyXDP (e.g., "eth0", "lo")
	Size      uint32 // Number of slots in the Time Wheel
	Rate      int64  // Tokens/requests per interval
	Max       int64  // Maximum capacity or packet threshold
}

// New initializes the rate limiter using the provided Config.
func New(cfg Config) (Limiter, error) {
	switch cfg.Strategy {
	case StrategyMemory:
		// We extract the fields to pass to the internal package
		return wheel.New(cfg.Size, cfg.Rate, cfg.Max), nil

	case StrategyXDP:
		if cfg.IfaceName == "" {
			return nil, errors.New("network interface name is required for XDP strategy")
		}
		return wheel.NewXDP(cfg.IfaceName, cfg.Max)

	default:
		return nil, errors.New("unknown strategy")
	}
}
