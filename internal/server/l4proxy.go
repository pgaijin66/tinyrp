package server

import (
	"io"
	"net"
	"sync"
)

// l4Proxy copies bytes bidirectionally between client and backend
// TCP connections. Go's io.Copy uses splice(2) on Linux when both
// sides are *net.TCPConn, giving true zero-copy kernel-to-kernel
// data transfer.
func l4Proxy(client, backend net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// client -> backend
	go func() {
		defer wg.Done()
		io.Copy(backend, client)
		// signal write-done so backend sees EOF
		if tc, ok := backend.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	// backend -> client
	go func() {
		defer wg.Done()
		io.Copy(client, backend)
		if tc, ok := client.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	wg.Wait()
}

// RunL4 starts a TCP proxy that forwards all connections to a single
// backend address. This is the fastest possible proxy mode since it
// operates at L4 with no HTTP parsing.
func RunL4(listenAddr, backendAddr string) error {
	ln, err := newReusePortListener(listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		client, err := ln.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer client.Close()
			backend, err := net.Dial("tcp", backendAddr)
			if err != nil {
				return
			}
			defer backend.Close()
			l4Proxy(client, backend)
		}()
	}
}
