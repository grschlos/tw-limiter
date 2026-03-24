# TW-Limiter (Time Wheel Limiter)

A high-performance, distributed rate limiter implementation in Go, leveraging the **Time Wheel** algorithm for maximum efficiency in high-load environments.

> **Status:** **Work in Progress (Active Development)**
> Currently refining concurrency primitives and lock-free/sharded state management.

---

## Motivation

Standard rate limiters often rely on priority queues or heaps ($O(\log n)$ for insertion/deletion), which can become a bottleneck under extreme RPS. This project implements a **Hierarchical Time Wheel** to achieve:
- **$O(1)$** complexity for all basic operations.
- Minimal CPU overhead for timer management.
- High precision even with millions of active rate-limit buckets.

## Architectural Decisions

- **Lock-Striping:** To minimize mutex contention, the internal state is sharded. The number of shards is always a **power of two**, allowing for fast index calculation using bitwise `&` instead of `%`.
- **Zero-Allocation Focus:** Using `sync.Pool` for internal timer nodes to keep GC overhead at bay.
- **Distributed Ready:** Designed with a pluggable storage interface (starting with In-Memory, followed by Redis/eBPF integration).

## Tech Stack
- **Language:** Go 1.23+
- **Concurrency:** Native goroutines & atomic primitives.
- **Data Structures:** Circular buffers, sharded maps.

---

## Roadmap
- [ ] Core Time Wheel logic (Single-node).
- [ ] Lock-striped sharding implementation.
- [ ] Benchmarking vs standard `golang.org/x/time/rate`.
- [ ] Redis integration for distributed state.
