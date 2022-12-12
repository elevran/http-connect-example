package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var (
	proxy  = flag.String("proxy", "localhost:3333", "proxy address")
	target = flag.String("target", "localhost:9999", "target server to connect to")
)

func main() {
	flag.Parse()

	addr, port, err := net.SplitHostPort(*proxy)
	if port == "" {
		*proxy = net.JoinHostPort(addr, "80")
	}
	log.Println("dialing proxy", *proxy)
	c, err := dial(*proxy)
	if err != nil {
		log.Fatalln("dial proxy failed", err)
	}
	defer c.Close()

	// @todo there's probably a cleaner way of doing this, other than
	// writing the string directly (e.g., maybe setting &http.Request{} fields?)
	// Having http.Request allow adding any relevant headers to req.Header
	// Body not expected according to spec

	// NOTE trailing '/' in request path. This is a Chi router limitation :-(
	// Anther option would be to use a different method (e.g., POST) with "normal"
	// path handling and an Upgrade header. This would not be standard compliant, however
	if _, err = fmt.Fprintf(c, "%s %s/ HTTP/1.1\r\nHost:%s\r\n\r\n",
		http.MethodConnect, *target, *proxy); err != nil {
		log.Fatalln("failed to write request", err)
	}

	br := bufio.NewReader(c)
	reply, err := http.ReadResponse(br, nil)
	if err != nil {
		log.Fatalf("reading HTTP response from CONNECT to %s via proxy %s failed: %v",
			*target, *proxy, err)
	}

	if reply.StatusCode != http.StatusOK {
		log.Printf("proxy error from %s while dialing %s: %v", *proxy, addr, reply.Status)
		bytes, err := io.ReadAll(reply.Body)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Body:", string(bytes))
		reply.Body.Close()
		return
	}

	// It's safe to discard the bufio.Reader here and use the original TCP connection
	// directly. According to the Spec, there must not be a body. But we can
	// double-check to confirm.
	// We can still use the Headers to pass information from the remote
	if br.Buffered() > 0 {
		log.Printf("unexpected %d bytes of buffered data from CONNECT proxy %q",
			br.Buffered(), *proxy)
	}

	// From here on it's TCP
	if err = ping(c); err != nil {
		log.Println("ping target failed", err)
	}
}

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

// this just to show that we have a TCP connection on hand, not HTTP...
func ping(c net.Conn) error {
	b := make([]byte, 16)

	for i := 1; i <= 5; i++ {
		t0 := time.Now()
		if _, err := c.Write([]byte("ping")); err != nil {
			return err
		}
		if _, err := c.Read(b); err != nil {
			return err
		}
		log.Println("ping #", i, "completed in", time.Now().Sub(t0))
	}
	return nil
}
