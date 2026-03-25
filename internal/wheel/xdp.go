//go:build linux
// +build linux

package wheel

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/cilium/ebpf/link"
	"github.com/grschlos/tw-limiter/internal/ebpf"
)

// configVal mirrors the memory layout of 'struct config_val' in the C code.
// It is used to pass dynamic configuration to the kernel via BPF maps.
type configVal struct {
	MaxPackets uint64
	IntervalNs uint64
}

// XdpLimiter represents the kernel-level rate limiting strategy.
type XdpLimiter struct {
	objs *ebpf.BpfObjects
	link link.Link
	max  uint64
}

// ipToUint32 converts an IPv4 string to a uint32 in Network Byte Order (Big Endian).
func ipToUint32(ipStr string) (uint32, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, errors.New("invalid IP format")
	}

	ip = ip.To4()
	if ip == nil {
		return 0, errors.New("only IPv4 is supported by this XDP program")
	}

	return binary.BigEndian.Uint32(ip), nil
}

// NewXDP initializes the eBPF rate limiter and applies the dynamic configuration.
func NewXDP(ifaceName string, max int64) (*XdpLimiter, error) {
	objs := ebpf.BpfObjects{}
	if err := ebpf.LoadBpfObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("loading bpf objects: %w", err)
	}

	// 1. Prepare and inject the dynamic configuration into the kernel.
	// The index is always 0 because config_map is an ARRAY map with max_entries=1.
	var key uint32 = 0
	cfg := configVal{
		MaxPackets: uint64(max),
		IntervalNs: 1_000_000_000, // Hardcoded to 1 second for now, can be extracted to Config later
	}

	if err := objs.ConfigMap.Put(&key, &cfg); err != nil {
		objs.Close()
		return nil, fmt.Errorf("failed to put config into bpf map: %w", err)
	}

	// 2. Resolve the network interface by name.
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("lookup interface %s: %w", ifaceName, err)
	}

	// 3. Attach the eBPF program to the XDP hook of the interface.
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.CountPackets,
		Interface: iface.Index,
	})
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("attach xdp: %w", err)
	}

	return &XdpLimiter{
		objs: &objs,
		link: l,
		max:  uint64(max),
	}, nil
}

// Allow checks the current state of the IP address in the kernel map.
// The kernel actively drops packets itself; this method is primarily for status monitoring in Go.
func (x *XdpLimiter) Allow(ctx context.Context, key string) (Result, error) {
	ipKey, err := ipToUint32(key)
	if err != nil {
		return Result{Allowed: true}, err
	}

	var counter struct {
		LastReset uint64
		Count     uint64
	}

	err = x.objs.IpLimits.Lookup(&ipKey, &counter)
	if err != nil {
		// Key not found means no packets have been seen yet
		return Result{Allowed: true}, nil
	}

	// Compare against our dynamically set threshold
	return Result{Allowed: counter.Count <= x.max}, nil
}

// Close removes the XDP program from the network interface and cleans up resources.
func (x *XdpLimiter) Close() error {
	if x.link != nil {
		x.link.Close()
	}
	if x.objs != nil {
		return x.objs.Close()
	}
	return nil
}
