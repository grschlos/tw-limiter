//go:build !linux
// +build !linux

package wheel

import (
	"context"
	"errors"
)

type XdpLimiter struct{}

func NewXDP(ifaceName string, max int64) (*XdpLimiter, error) {
	return nil, errors.New("XDP strategy is only supported on Linux")
}

func (x *XdpLimiter) Allow(ctx context.Context, key string) (Result, error) {
	return Result{}, nil
}

func (x *XdpLimiter) Close() error {
	return nil
}
