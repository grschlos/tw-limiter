package wheel

import (
	"sync"
	"sync/atomic"
)

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

	interval int64  // one tick interval, ns
	slots    int    // num of slots in the wheel
	cursor   uint32 // current wheel "cursor" position
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
