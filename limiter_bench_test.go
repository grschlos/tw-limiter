package limiter_test

import (
	"context"
	"github.com/grschlos/tw-limiter"
	"testing"
)

// BenchmarkMemoryLimiter tests the standard Time Wheel performance.
// It shows how fast our lock-striped sharding is.
func BenchmarkMemoryLimiter(b *testing.B) {
	// 1024 shards, 1000 req/sec, burst 100
	l, _ := limiter.New(limiter.Config{
		Strategy: limiter.StrategyMemory,
		Size:     1024,
		Rate:     100,
		Max:      10,
	})
	defer l.Close()

	ctx := context.Background()
	key := "user-42"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = l.Allow(ctx, key)
		}
	})
}

// BenchmarkXDPLimiter_Overhead tests the Go-side overhead of the XDP strategy.
// In reality, XDP performance is measured in Mpps (Millions of packets per second)
// at the driver level, but here we show that the Go-layer is virtually free.
func BenchmarkXDPLimiter_Overhead(b *testing.B) {
	l, err := limiter.New(limiter.Config{
		Strategy:  limiter.StrategyXDP,
		IfaceName: "lo",
		Max:       100,
	})
	if err != nil {
		b.Skip("XDP not supported or no root privileges")
	}
	defer l.Close()

	ctx := context.Background()
	key := "127.0.0.1"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = l.Allow(ctx, key)
		}
	})
}
