package e2e

import (
	"fmt"
	"testing"
	"time"
)

func TestLavaProtocol(t *testing.T) {
	// default timeout same as `go test`
	timeout := time.Minute * 10

	if deadline, ok := t.Deadline(); ok {
		timeout = time.Until(deadline).Round(10 * time.Second)
	}

	runProtocolE2E(timeout)
}

func TestLavaSDK(t *testing.T) {
	fmt.Println("Starting SDK tests, what will happen, you will not see... until it's too late")
	// default timeout same as `go test`
	timeout := time.Minute * 10

	if deadline, ok := t.Deadline(); ok {
		timeout = time.Until(deadline).Round(10 * time.Second)
	}

	runSDKE2E(timeout)
}

func TestLavaProtocolPayment(t *testing.T) {
	// default timeout same as `go test`
	timeout := time.Minute * 7

	if deadline, ok := t.Deadline(); ok {
		timeout = time.Until(deadline).Round(10 * time.Second)
	}

	runPaymentE2E(timeout)
}
