package main

import (
	"context"
	"io"
	"net"
	"sync"
	"time"
)

func dial(upstream string) (net.Conn, error) {
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	c, err := d.DialContext(ctx, "tcp", upstream)

	if cancel != nil {
		cancel()
	}
	if err != nil {
		return nil, err
	}
	return c, err
}

func splice(src, dst net.Conn) {
	done := &sync.WaitGroup{}
	done.Add(2)
	go iocopy(done, src, dst)
	go iocopy(done, dst, src)
	done.Wait()
}

// copy the bytes around...
func iocopy(wg *sync.WaitGroup, src, dst net.Conn) {
	io.Copy(src, dst)
	dst.(*net.TCPConn).CloseWrite()
	wg.Done()
}
