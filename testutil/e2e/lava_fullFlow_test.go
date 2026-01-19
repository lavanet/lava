package e2e

import (
	"fmt"
	"testing"
	"time"
)

func TestLavaProtocol(t *testing.T) {
	// default timeout same as `go test`
	timeout := time.Minute * 10

	fmt.Printf("TestLavaProtocol: default timeout %s\n", timeout)
	if deadline, ok := t.Deadline(); ok {
		timeout = time.Until(deadline).Round(10 * time.Second)
		fmt.Printf("TestLavaProtocol: t.Deadline()=%s, adjusted timeout %s\n", deadline.Format(time.RFC3339), timeout)
	} else {
		fmt.Println("TestLavaProtocol: no t.Deadline() provided, using default timeout")
	}

	runProtocolE2E(timeout)
}

func TestLavaProtocolPayment(t *testing.T) {
	// default timeout same as `go test`
	timeout := time.Minute * 7

	if deadline, ok := t.Deadline(); ok {
		timeout = time.Until(deadline).Round(10 * time.Second)
	}

	runPaymentE2E(timeout)
}
