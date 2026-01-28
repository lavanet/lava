// Mock RPC server for Smart Router Direct RPC testing.
// Returns configurable HTTP status and delay so you can test retry, health, and 4xx behavior.
//
// Usage: go run ./scripts/mock_rpc_server [flags]
// Flags: -port  Listen port (default 19999)
//
// When testing via Smart Router, use the control API (router does not forward X-Mock-* headers).
// If ALL providers point at the same mock, the router sends N parallel requests; use sticky
// so every request gets the same status until you reset:
//
//	curl "http://127.0.0.1:19999/control?status=500&sticky=1"  # ALL requests get 500 until reset
//	curl "http://127.0.0.1:19999/control?status=200"          # reset to success (clears sticky)
//
// Timeout handling: use wait (seconds) to delay the response and test per-request timeouts:
//
//	curl "http://127.0.0.1:19999/control?wait=5"   # next response sleeps 5s (or use delay=5)
//	curl "http://127.0.0.1:19999/control?wait=10&sticky=1"  # all responses wait 10s until reset
//
// Then send the RPC through the router; all parallel requests to the mock will get 500.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const defaultPort = 19999

// Valid JSON-RPC response for eth_blockNumber (hex block number)
const okBody = `{"jsonrpc":"2.0","id":1,"result":"method does not exist"}` + "\n"

var (
	nextStatus = 200
	nextDelay  = 0
	sticky     = false // when true, status/delay stay until you set status=200
	mu         sync.Mutex
)

func main() {
	port := flag.Int("port", defaultPort, "listen port")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("[MockRPC] listening on %s", addr)
	log.Printf("[MockRPC] control: GET /control?status=200|400|429|500 ; add &sticky=1 so ALL requests get that status until status=200")
	log.Printf("[MockRPC] control: GET /control?wait=N (or delay=N) to sleep N seconds before response (test timeout handling)")
	log.Printf("[MockRPC] default: 200 + eth_blockNumber-style JSON")

	http.HandleFunc("/control", control)
	http.HandleFunc("/", handle)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[MockRPC] listen failed: %v", err)
	}
}

// control sets the next response status or delay. Use when testing via Smart Router
// (router does not forward X-Mock-* headers). GET /control?status=500 then send
// request through router; mock will return 500.
func control(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if s := r.URL.Query().Get("status"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && (n == 200 || n == 400 || n == 429 || n == 500) {
			nextStatus = n
			nextDelay = 0
			if n == 200 {
				sticky = false
				log.Printf("[MockRPC] control: status=200 (reset, sticky cleared)")
			} else {
				sticky = r.URL.Query().Get("sticky") == "1" || r.URL.Query().Get("sticky") == "true"
				if sticky {
					log.Printf("[MockRPC] control: STICKY status=%d (all requests get this until you set status=200)", n)
				} else {
					log.Printf("[MockRPC] control: next response status=%d", n)
				}
			}
		}
	}
	// delay or wait (seconds) before responding — for timeout handling tests
	for _, key := range []string{"delay", "wait"} {
		if d := r.URL.Query().Get(key); d != "" {
			if n, err := strconv.Atoi(d); err == nil && n >= 0 && n <= 300 {
				nextDelay = n
				if r.URL.Query().Get("sticky") == "1" || r.URL.Query().Get("sticky") == "true" {
					sticky = true
					log.Printf("[MockRPC] control: STICKY wait=%ds (all requests wait until you set status=200)", n)
				} else {
					log.Printf("[MockRPC] control: next response wait=%ds", n)
				}
				break
			}
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(200)
	fmt.Fprintf(w, "ok status=%d wait=%ds sticky=%v\n", nextStatus, nextDelay, sticky)
}

func handle(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	status := nextStatus
	delaySec := nextDelay
	if !sticky && (nextStatus != 200 || nextDelay > 0) {
		nextStatus = 200
		nextDelay = 0
	}
	mu.Unlock()

	if delaySec > 0 {
		log.Printf("[MockRPC] request %s %s -> status=%d delay=%ds (sleeping)", r.Method, r.URL.Path, status, delaySec)
		time.Sleep(time.Duration(delaySec) * time.Second)
	} else {
		log.Printf("[MockRPC] request %s %s -> status=%d", r.Method, r.URL.Path, status)
	}

	if s := r.Header.Get("X-Mock-Status"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && (n == 200 || n == 400 || n == 429 || n == 500) {
			status = n
		}
	}
	// Per-request wait (seconds) — X-Mock-Wait or X-Mock-Delay, for timeout/override tests
	for _, h := range []string{"X-Mock-Wait", "X-Mock-Delay"} {
		if d := r.Header.Get(h); d != "" {
			if n, err := strconv.Atoi(d); err == nil && n >= 0 && n <= 300 {
				time.Sleep(time.Duration(n) * time.Second)
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	switch status {
	case 200:
		w.Write([]byte(okBody))
	case 400:
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"invalid request"}}` + "\n"))
	case 429:
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32005,"message":"rate limit exceeded"}}` + "\n"))
	case 500:
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"internal error"}}` + "\n"))
	default:
		w.Write([]byte(okBody))
	}
}
