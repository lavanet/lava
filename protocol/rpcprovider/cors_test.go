package rpcprovider

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
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
	require.NotNil(t, err, "Expected CORS check to fail but it passed")
	require.True(t, strings.Contains(err.Error(), "CORS check failed"), "Expected CORS related error message")
}
