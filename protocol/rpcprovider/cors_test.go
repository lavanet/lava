package rpcprovider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func StartTestServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, this server doesn't set CORS headers!")
	})
	err := http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", mux)
	if err != nil {
		log.Fatalf("Failed to start server 8080: %s", err.Error())
	}
}

func StartTestServerWithOriginHeader() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, "Hello, this server sets Access-Control-Allow-Origin but not x-grpc-web!")
	})
	err := http.ListenAndServeTLS(":8081", "cert.pem", "key.pem", mux)
	if err != nil {
		log.Fatalf("Failed to start server 8081: %s", err.Error())
	}
}

func StartTestServerWithXGrpcWeb() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-grpc-web")
		fmt.Fprint(w, "Hello, this server sets Access-Control-Allow-Origin and x-grpc-web but not lava-sdk-relay-timeout!")
	})
	err := http.ListenAndServeTLS(":8082", "cert.pem", "key.pem", mux)
	if err != nil {
		log.Fatalf("Failed to start server 8082: %s", err.Error())
	}
}

func StartTestServerWithAllHeaders() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-grpc-web, lava-sdk-relay-timeout")
		fmt.Fprint(w, "Hello, this server sets all required headers!")
	})
	err := http.ListenAndServeTLS(":8083", "cert.pem", "key.pem", mux)
	if err != nil {
		log.Fatalf("Failed to start server 8083: %s", err.Error())
	}
}

func TestMain(m *testing.M) {
	err := CreateSelfSignedCertificate("cert.pem", "key.pem", 365*24*time.Hour)
	if err != nil {
		panic(err)
	}

	go StartTestServer()
	go StartTestServerWithOriginHeader()
	go StartTestServerWithXGrpcWeb()
	go StartTestServerWithAllHeaders()
	time.Sleep(10 * time.Millisecond) // allow the servers to finish starting
	code := m.Run()

	os.Exit(code)
}

func TestPerformCORSCheckFail(t *testing.T) {
	endpoint := epochstoragetypes.Endpoint{
		IPPORT: "localhost:8080",
	}

	err := PerformCORSCheck(endpoint)
	require.Error(t, err, "Expected CORS check to fail but it passed")
	require.True(t, strings.Contains(err.Error(), "CORS check failed"), "Expected CORS related error message", err)
}

func TestPerformCORSCheckFailXGrpcWeb(t *testing.T) {
	endpoint := epochstoragetypes.Endpoint{
		IPPORT: "localhost:8081",
	}

	err := PerformCORSCheck(endpoint)
	require.Error(t, err, "Expected CORS check to fail but it passed")
	require.True(t, strings.Contains(err.Error(), "x-grpc-web"), "Expected error to relate to x-grpc-web")
}

func TestPerformCORSCheckFailLavaSdkRelayTimeout(t *testing.T) {
	endpoint := epochstoragetypes.Endpoint{
		IPPORT: "localhost:8082",
	}

	err := PerformCORSCheck(endpoint)
	require.Error(t, err, "Expected CORS check to fail but it passed")
	require.True(t, strings.Contains(err.Error(), "lava-sdk-relay-timeout"), "Expected error to relate to lava-sdk-relay-timeout")
}

func TestPerformCORSCheckSuccess(t *testing.T) {
	endpoint := epochstoragetypes.Endpoint{
		IPPORT: "localhost:8083", // pointing to the server with all headers
	}

	err := PerformCORSCheck(endpoint)
	require.NoError(t, err, "Expected CORS check to pass but it failed")
}

func CreateSelfSignedCertificate(certPath, keyPath string, validFor time.Duration) error {
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Lava Network"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		return err
	}

	keyFile, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}

	err = pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	if err != nil {
		return err
	}

	return nil
}
