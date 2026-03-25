# TW-Limiter: Hybrid Multi-Layer Rate Limiting in Go & eBPF

[![Go Reference](https://pkg.go.dev/badge/github.com/grschlos/tw-limiter.svg)](https://pkg.go.dev/github.com/grschlos/tw-limiter)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![eBPF](https://img.shields.io/badge/eBPF-Kernel_Powered-black?logo=linux)](#)

A high-performance, dual-layer rate limiting library written in Go. It combines a highly concurrent **Userspace Time Wheel** with a **Kernel-level eBPF/XDP filter** to provide granular API throttling and robust infrastructure protection against DDoS attacks.

## Architecture Overview

`tw-limiter` operates on two distinct layers, allowing you to choose the right strategy for your use case:

1. **Application Layer (StrategyMemory):**
   Utilizes a lock-striped Time Wheel algorithm in Userspace. It achieves $O(1)$ complexity with zero allocations in the hot path, making it perfect for fine-grained business logic and API routing.
   
2. **Infrastructure Layer (StrategyXDP):**
   Leverages Linux eBPF/XDP (eXpress Data Path) to drop malicious packets directly at the NIC driver level, bypassing the Linux network stack entirely. Ideal for extreme load mitigation with virtually zero CPU overhead in the Go application.

---

## Benchmarks

### Environment 1: Server (High-End)
* **CPU:** AMD EPYC 7763 (Milan, Zen 3)
* **OS:** Linux (amd64)

| Benchmark | Latency | Allocations | Description |
| :--- | :--- | :--- | :--- |
| **Memory (Parallel)** | **28.91 ns/op** | 0 B/op | Different shards (No contention) |
| **Memory (Contention)** | **51.77 ns/op** | 0 B/op | Same shard (High contention) |
| **XDP Strategy** | **< 1.00 ns/op** | 0 B/op | Kernel-level bypass |
| *Standard Go Rate Limiter* | *~99.46 ns/op* | *0 B/op* | *Standard `x/time/rate`* |

### Environment 2: Laptop (Mobile)
* **CPU:** Intel(R) Core(TM) i7-8565U
* **OS:** Linux (amd64)

| Benchmark | Latency | Description |
| :--- | :--- | :--- |
| **Memory (Contention)** | **~68.38 ns/op** | Reliable performance on older hardware |
| **Standard Go Rate Limiter** | **~153.60 ns/op** | Significant overhead compared to TW-Limiter |

---

## Features
* **Lock-free hot path**: Minimal contention using shard-based locking.
* **eBPF/XDP Integration**: Drops malicious traffic before it reaches the OS network stack.
* **Zero Allocations**: Highly optimized for high-throughput systems.
* **Cross-Platform**: Uses Go Build Tags. Compiles seamlessly on Windows/macOS (with fallback stubs) while enabling XDP on Linux.
* **Type-Safe API**: Clean encapsulation using Go `internal` packages and Type Aliasing.

## Installation

```bash
go get [github.com/grschlos/tw-limiter](https://github.com/grschlos/tw-limiter)
```

---

## Usage

# 1. Application-Level Rate Limiting (Cross-Platform)

Best for API throttling and business logic. Works on Linux, macOS, and Windows.

```go
package main

import (
	"context"
	"log"
	"github.com/grschlos/tw-limiter"
)

func main() {
	// Initialize using Memory Strategy (Time Wheel)
	l, err := limiter.New(limiter.Config{
		Strategy: limiter.StrategyMemory,
		Size:     1024, // 1024 shards for high concurrency
		Rate:     100,  // 100 requests per interval
		Max:      10,   // Max burst capacity
	})
	if err != nil {
		log.Fatalf("failed to create limiter: %v", err)
	}
	defer l.Close()

	// Check limit for a specific key
	res, err := l.Allow(context.Background(), "user-123")
	if err == nil && res.Allowed {
		// Handle request
	}
}
```

# 2. Kernel-Level XDP Filtering (Linux + Root required)

Best for DDoS protection and high-load packet filtering.

```go
package main

import (
	"context"
	"log"
	"github.com/grschlos/tw-limiter"
)

func main() {
	// Initialize using XDP Strategy
	l, err := limiter.New(limiter.Config{
		Strategy:  limiter.StrategyXDP,
		IfaceName: "eth0", // Attach to physical interface
		Max:       100,    // Allow 100 packets/sec per IP in Kernel
	})
	if err != nil {
		log.Fatalf("failed to load XDP: %v (are you root?)", err)
	}
	defer l.Close()

	// Check status of a specific IP from the eBPF map
	res, err := l.Allow(context.Background(), "192.168.1.50")
	if err == nil && !res.Allowed {
		log.Println("Kernel is currently dropping packets from this IP")
	}
}
```
---

## Verifying XDP Protection (Flood Test)

To see the eBPF filter in action on your local machine, you can run a flood ping test:

1. Run the included example attached to the loopback interface (lo):
```bash
sudo go run examples/xdp-filter/main.go
```

2. In a separate terminal, initiate a flood ping to exceed the 100 packets/sec limit:
```bash
sudo ping -f 127.0.0.1
```

3. **Observe the results:** You will see a significant packet loss percentage in the `ping` statistics (which is normally 0% on localhost), proving that the XDP program is dropping excess packets at the lowest system level.

---


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
- [x] eBPF-based XDP limiter for DDoS protection.
- [ ] Redis integration for distributed state.
