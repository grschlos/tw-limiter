package wheel

import (
	"context"
	"fmt"
	"testing"

	"golang.org/x/time/rate"
)

// BenchmarkAllowParallel checks the performance at high concurrency.
// We use different keys to work with different shards
func BenchmarkAllowParallel(b *testing.B) {
	// Initializing the limiter: 1024 shards, 1000 runs/sec, 100
	// tokens/bucket
	tw := &TimeWheel{
		shards:    make([]Shard, 1024),
		shardMask: 1023,
		rate:      1000,
		maxTokens: 100,
	}
	for i := range tw.shards {
		tw.shards[i].buckets = make(map[string]*bucket)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Every worker generates its own key to simulate real workload
		key := fmt.Sprintf("user-%p", pb)
		for pb.Next() {
			_, _ = tw.Allow(ctx, key)
		}
	})
}

// BenchmarkAllowContention simulates the worst-case-scenario: all the requests
// hit the same key.
func BenchmarkAllowContention(b *testing.B) {
	tw := New(1024, 1000, 100)
	tw.rate = 1000
	tw.maxTokens = 100
	ctx := context.Background()
	key := "static-key"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = tw.Allow(ctx, key)
		}
	})
}

// Standard Limiter (for comparison)
func BenchmarkStandardRateLimiter(b *testing.B) {
	limiter := rate.NewLimiter(rate.Limit(1000), 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = limiter.Allow()
		}
	})
}
