//go:build linux
// +build linux

package wheel

import (
	"context"
	"fmt"
	"net"

	"github.com/cilium/ebpf/link"
	"github.com/grschlos/tw-limiter/internal/ebpf"
)

type XdpLimiter struct {
	objs *ebpf.BpfObjects
	link link.Link
}

func NewXDP(size uint32, rate, max int64) (*XdpLimiter, error) {
	objs := ebpf.BpfObjects{}
	if err := ebpf.LoadBpfObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("loading bpf objects: %w", err)
	}

	iface, err := net.InterfaceByName("lo")
	if err != nil {
		objs.Close()
		return nil, fmt.Errorf("lookup interface: %w", err)
	}

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
	}, nil
}

func (x *XdpLimiter) Allow(ctx context.Context, key string) (Result, error) {
	var value uint64
	// Simplified: converting string IP to uint32 (use net.ParseIP in real life)
	var localhost uint32 = 0x0100007f

	// Check our eBPF map
	err := x.objs.IpLimits.Lookup(&localhost, &value)
	if err != nil {
		return Result{Allowed: true}, nil // Not seen yet
	}

	// Return status based on what kernel is doing
	return Result{Allowed: value <= 100}, nil
}

func (x *XdpLimiter) Close() error {
	if x.link != nil {
		x.link.Close()
	}
	if x.objs != nil {
		x.objs.Close()
	}
	return nil
}
