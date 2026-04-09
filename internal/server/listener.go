package server

import (
	"context"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// newReusePortListener creates a TCP listener with SO_REUSEPORT set.
// This lets multiple processes bind the same port, with the kernel
// distributing connections across them.
func newReusePortListener(addr string) (net.Listener, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				if opErr != nil {
					return
				}
				// disable Nagle's algorithm for lower latency
				_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NODELAY, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}
	return lc.Listen(context.Background(), "tcp", addr)
}
