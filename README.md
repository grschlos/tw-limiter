# TW-Limiter (Time Wheel Limiter)

A high-performance, **zero-allocation** rate limiter in Go, optimized for multi-core systems using Lock-Striping and CAS-loops.

> **Status:** **Core Implementation Complete**
> Achieving ~29ns latency with zero allocations in high-concurrency scenarios.

---

### Comparative Analysis

We compared **TW-Limiter** against the official `golang.org/x/time/rate` on an **AMD EPYC™ 7763** (4 cores allocated).

| Implementation | Latency (Parallel) | Performance Gain | Allocations |
| :--- | :--- | :--- | :--- |
| `x/time/rate` (Standard) | 102.0 ns/op | 1.0x (Baseline) | 0 allocs/op |
| **TW-Limiter** (This project) | **28.6 ns/op** | **~3.5x Faster** | **0 allocs/op** |

> **Key Takeaway:** TW-Limiter provides significantly higher throughput for multi-threaded applications by eliminating the global mutex bottleneck.

*Note: Benchmarks performed on AMD EPYC™ 7763. TW-Limiter is up to 2x faster under high contention.*

---

## Usage

Installation:

```bash
go get github.com/grschlos/tw-limiter
```

Basic example:

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/grschlos/tw-limiter"
)

func main() {
    // Initialize: 1024 shards, 100 req/sec, burst capacity of 10 tokens
    l := limiter.New(1024, 100, 10)
    defer l.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    // Check rate limit for a specific key (e.g., IP or API Key)
    res, err := l.Allow(ctx, "user-123")
    
    if errors.Is(err, limiter.ErrRateLimitExceeded) {
        fmt.Printf("Rate limit reached! Try again after %v\n", res.ResetAfter)
        return
    }

    if res.Allowed {
        fmt.Printf("Success! Remaining tokens: %d\n", res.Remaining)
    }
}
```

---

## Key Engineering Decisions

- **Lock-Free Fast Path:** Instead of holding a write-lock for every request, we use a **CAS (Compare-And-Swap)** loop for bucket updates. This allows multiple goroutines to update the same shard simultaneously if they hit different keys.
- **Cache-Line Alignment (False Sharing Protection):** Shards are padded to 64 bytes to ensure they reside on different L1/L2 cache lines, preventing CPU cache bouncing.
- **Lazy State Rotation:** Avoids background "stop-the-world" cleanup by calculating token drift on-the-fly ($O(1)$ complexity).
- **Contiguous Memory:** Shards are stored as a slice of values (`[]Shard`), significantly reducing GC scan time compared to a slice of pointers.

---

## Roadmap
- [x] Core Lock-Free logic.
- [x] Lock-striped sharding (1024+ shards).
- [x] Zero-allocation benchmarking.
- [ ] Redis integration for distributed state.
- [ ] eBPF-based XDP limiter for DDoS protection.
