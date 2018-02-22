package main

import (
	"context"
	"net"
	"time"
)

func resolve(addr string, timeout time.Duration) ([]net.IPAddr, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return net.DefaultResolver.LookupIPAddr(ctx, addr)
}
