package main

import (
	"flag"
	"io"
	"log"
	"net"
)

func main() {
	port := flag.String("p", "9999", "port to listen on ")
	flag.Parse()

	server, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalln(err)
	}
	defer server.Close()

	log.Println("Echo server is running on port", *port)

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("Failed to accept connection", err)
			continue
		}
		log.Println("connection accepted", conn.RemoteAddr())

		go func(conn net.Conn) {
			defer func() {
				conn.Close()
			}()
			io.Copy(conn, conn)
		}(conn)
	}
}
