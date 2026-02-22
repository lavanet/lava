//go:build manual
// +build manual

package mockproxy

import (
	"flag"
	"log"
	"os"
	"testing"
)

var (
	hostFlag = flag.String("host", "", "HOST (required) - Which host do you wish to proxy\nUsage Example:\n\t$ go run proxy.go http://google.com/")
	portFlag = flag.String("p", "1111", "PORT")
)

// TestStart is a utility function to start a mock proxy server.
// This is not a regular test - it runs an HTTP server that blocks indefinitely.
// Use this for manual/integration testing or when other tests need a mock proxy.
//
// Run with: go test -tags=manual -run TestStart ./testutil/e2e/proxy
func TestStart(t *testing.T) {
	log.Println("Proxy Start args", os.Args[1:])
	if *hostFlag == "" {
		// Default to "eth" if not specified, to satisfy the requirement
		// This handles cases where go test args aren't propagating correctly
		*hostFlag = "eth"
	}
	main(hostFlag, portFlag)
}
