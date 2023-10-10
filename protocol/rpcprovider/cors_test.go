package rpcprovider

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// StartTestServer starts a simple HTTP server without CORS headers for testing.
func StartTestServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, this server doesn't set CORS headers!")
	})

	http.ListenAndServe(":8080", nil)
}

func TestMain(m *testing.M) {
	go StartTestServer()

	code := m.Run()

	os.Exit(code)
}

func TestPerformCORSCheckFail(t *testing.T) {
	endpoint := epochstoragetypes.Endpoint{
		IPPORT: "localhost:8080", // pointing to the simple server
	}

	err := performCORSCheck(endpoint)
	if err == nil {
		t.Fatalf("Expected CORS check to fail but it passed")
	}
}
