package wheel

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const nsInSec = int64(time.Second)

// bucket keeps the current limit state for a particular user/key.
type bucket struct {
	tokens     int64 // Current amount of tokens available (we're using atomic)
	lastUpdate int64 // Nanoseconds since the last update
}

// Shard is a data segment, reducing the Mutex concurrency.
type Shard struct {
	mu      sync.RWMutex
	buckets map[string]*bucket

	// Padding to avoid False Sharing among shards.
	// 64 bytes is a typical x86 cache line size.
	_ [56]byte
}

// TimeWheel manages the shards and rotation logic.
type TimeWheel struct {
	shards    []Shard
	shardMask uint32 // mask for a quick shard index calculation (n-1)

	interval  int64  // one tick interval, ns
	rate      int64  // num of tokens per second
	maxTokens int64  // bucket capacity
	slots     int    // num of slots in the wheel
	cursor    uint32 // current wheel "cursor" position
}

func New(size uint32) *TimeWheel {
	// initializing all the shards at once
	shards := make([]Shard, size)
	for i := range shards {
		shards[i].buckets = make(map[string]*bucket)
	}
	return &TimeWheel{
		shards:    shards,
		shardMask: size - 1,
	}
}

func (tw *TimeWheel) getShard(key string) *Shard {
	// fast hash-function implementation(FNV-1a)
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	// apply bit mask
	return &tw.shards[hash&tw.shardMask]
}

func (tw *TimeWheel) Allow(ctx context.Context, key string) (Result, error) {
	shard := tw.getShard(key)
	now := time.Now().UnixNano()

	// 1. First try fast read under RLock
	shard.mu.RLock()
	b, ok := shard.buckets[key]
	shard.mu.RUnlock()

	if !ok {
		// 2. If there's no such bucket, it's created under the full
		// Lock
		shard.mu.Lock()
		// Double-check: whether it's been already created during
		// the Lock awaiting
		if b, ok = shard.buckets[key]; !ok {
			b = &bucket{
				tokens:     tw.maxTokens, // Start with the full bucket
				lastUpdate: now,
			}
			shard.buckets[key] = b
		}
		shard.mu.Unlock()
	}

	// 3. Work with bucket via atomics (minimizing lock contention)
	// This limits several requests concurrency by atomics
	return tw.processBucket(b, now), nil
}

func (tw *TimeWheel) processBucket(b *bucket, now int64) Result {
	for {
		// 1. Read current values using atomics
		oldTokens := atomic.LoadInt64(&b.tokens)
		oldLastUpdate := atomic.LoadInt64(&b.lastUpdate)

		// 2. Count how much time passed
		delta := now - oldLastUpdate
		if delta < 0 {
			delta = 0 // protection from system time hops
		}

		// Calculate a new balance
		refill := (delta * tw.rate) / nsInSec
		newTokens := oldTokens + refill
		if newTokens > tw.maxTokens {
			newTokens = tw.maxTokens
		}

		// 3. Try to get one token back
		allowed := false
		if newTokens > 0 {
			newTokens--
			allowed = true
		}

		// 4. Try to update bucket atomically
		if atomic.CompareAndSwapInt64(&b.tokens, oldTokens, newTokens) {
			// FIXME: make sure no updates happened (pack into a
			// single uint64 with loss of accuracy?)
			atomic.StoreInt64(&b.lastUpdate, now)

			return Result{
				Allowed:    allowed,
				Remaining:  int(newTokens),
				ResetAfter: time.Duration((tw.maxTokens - newTokens) * nsInSec / tw.rate),
			}
		}
	}
}
