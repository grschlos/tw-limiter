package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grschlos/tw-limiter"
)

func main() {
	// 1. Define command-line flags to avoid hardcoding
	iface := flag.String("iface", "lo", "Network interface to attach XDP program")
	targetIP := flag.String("ip", "127.0.0.1", "Target IPv4 address to monitor")
	maxPackets := flag.Int64("max", 100, "Maximum allowed packets per second")
	flag.Parse()

	fmt.Printf("Starting XDP filter on interface: %s\n", *iface)
	fmt.Printf("Monitoring IP: %s with limit: %d pkts/sec\n", *targetIP, *maxPackets)

	// 2. Initialize the limiter using the dynamic configuration
	l, err := limiter.New(limiter.Config{
		Strategy:  limiter.StrategyXDP,
		IfaceName: *iface,
		Max:       *maxPackets,
	})
	if err != nil {
		log.Fatalf("Failed to initialize XDP limiter: %v", err)
	}
	defer l.Close()

	// 3. Setup signal handling for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("XDP filter is active. Press Ctrl+C to stop.")

	// 4. Monitoring loop: periodically check the status of the target IP from the kernel
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			fmt.Println("\nDetaching XDP program and exiting...")
			return
		case <-ticker.C:
			res, err := l.Allow(ctx, *targetIP)
			if err != nil {
				log.Printf("Error checking status for %s: %v", *targetIP, err)
				continue
			}

			if !res.Allowed {
				log.Printf("ALERT: Limit exceeded for %s! Kernel is dropping packets.", *targetIP)
			} else {
				// Optional: log status if packets are being tracked but not yet limited
				log.Printf("Status for %s: OK", *targetIP)
			}
		}
	}
}
