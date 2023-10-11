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

func StartTestServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, this server doesn't set CORS headers!")
	})
	http.ListenAndServe(":8080", mux)
}

func StartTestServerWithOriginHeader() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, "Hello, this server sets Access-Control-Allow-Origin but not x-grpc-web!")
	})
	http.ListenAndServe(":8081", mux)
}

func TestMain(m *testing.M) {
	go StartTestServer()
	go StartTestServerWithOriginHeader()

	code := m.Run()

	os.Exit(code)
}

func TestPerformCORSCheckFail(t *testing.T) {
	endpoint := epochstoragetypes.Endpoint{
		IPPORT: "localhost:8080",
	}

	err := performCORSCheck(endpoint)
	require.NotNil(t, err, "Expected CORS check to fail but it passed")
	require.True(t, strings.Contains(err.Error(), "CORS check failed"), "Expected CORS related error message")
}

func TestPerformCORSCheckFailXGrpcWeb(t *testing.T) {
	endpoint := epochstoragetypes.Endpoint{
		IPPORT: "localhost:8081",
	}

	err := performCORSCheck(endpoint)
	require.NotNil(t, err, "Expected CORS check to fail but it passed")
	require.True(t, strings.Contains(err.Error(), "x-grpc-web"), "Expected error to relate to x-grpc-web")
}
