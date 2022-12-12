package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/go-chi/chi"
)

var (
	port     = flag.String("p", "3333", "port to listen on")
	upstream = flag.String("upstream", "localhost:9999", "allowed upstream")
)

func main() {
	flag.Parse()
	r := chi.NewRouter()

	// Chi does not accept an empty Path, so we use '/'. We could have avoided this by
	// using the http.ServerMux directly or by a PR to Chi allowing Connect with no
	// path in URL - as [CONNECT](https://httpwg.org/specs/rfc9110.html#CONNECT) does
	// not use it
	r.Connect("/", proxyConnection)
	log.Println("Proxy server is running on port", *port)
	http.ListenAndServe(":"+*port, r)
}

func proxyConnection(w http.ResponseWriter, r *http.Request) {
	if r.URL.Host != *upstream { // apply some logic to ensure this is a valid target
		http.Error(w, "invalid CONNECT target", http.StatusBadRequest)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	backend, err := dial(*upstream)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	defer backend.Close()

	// [Spec](https://httpwg.org/specs/rfc9110.html#CONNECT) says:
	// Any 2xx (Successful) response indicates that the sender (and all inbound proxies)
	// will switch to tunnel mode immediately after the response header section; data
	// received after that header section is from the server identified by the request
	// target. Any response other than a successful response indicates that the tunnel
	// has not yet been formed.

	// @todo consider adding headers to reply? Body is not allowed
	w.WriteHeader(http.StatusOK)

	// from this point on, the connection is in "TCP mode"
	log.Println("splicing", r.RemoteAddr, "to", *upstream)
	conn, _, err := hj.Hijack()
	defer conn.Close()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	splice(conn, backend)
}
