package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grschlos/tw-limiter"
)

func main() {
	// Use our high-level public API
	// StrategyXDP handles eBPF loading and attaching internally
	l, err := limiter.New(limiter.StrategyXDP, 0, 1000, 100)
	if err != nil {
		log.Fatalf("failed to initialize XDP limiter: %v (are you root?)", err)
	}
	defer l.Close()

	log.Println("XDP Limiter active on 'lo'. Try: ping 127.0.0.1")

	// Setup signal handling for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// In XDP mode, Allow() is mainly a check/status call
	// because the kernel is already doing the heavy lifting.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Just an example of calling the API
			res, err := l.Allow(ctx, "127.0.0.1")
			if err != nil {
				log.Printf("Error checking limits: %v", err)
				continue
			}

			if res.Allowed {
				log.Println("Traffic is flowing (under threshold)...")
			} else {
				log.Println("!!! Kernel is currently dropping packets from this IP")
			}

		case <-stop:
			log.Println("Shutting down and detaching XDP program...")
			return
		}
	}
}
