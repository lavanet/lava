package chainlib

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/require"
)

func TestNodeErrorHandlerGenericErrors(t *testing.T) {
	ctx := context.Background()
	neh := &genericErrorHandler{}

	// Test context.DeadlineExceeded error
	withTimeout, cancel := context.WithTimeout(ctx, 0)
	defer cancel()
	err := neh.handleGenericErrors(withTimeout, nil)
	expectedError := utils.LavaFormatError("Provider Failed Sending Message", context.DeadlineExceeded)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.ErrWriteToConnected error
	err = neh.handleGenericErrors(ctx, net.ErrWriteToConnected)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Write to connected connection", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.ErrClosed error
	err = neh.handleGenericErrors(ctx, net.ErrClosed)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Operation on closed connection", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test io.EOF error
	err = neh.handleGenericErrors(ctx, io.EOF)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: End of input stream reached", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.OpError with timeout error
	opErr := &net.OpError{
		Op:     "dummy",
		Net:    "dummy",
		Source: nil,
		Addr:   nil,
		Err:    os.ErrDeadlineExceeded,
	}
	err = neh.handleGenericErrors(ctx, opErr)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Network operation timed out", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.DNSError error
	dnsErr := &net.DNSError{
		Err: "dummy",
	}
	err = neh.handleGenericErrors(ctx, dnsErr)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: DNS resolution failed", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.OpError with connection refused error
	opErr = &net.OpError{
		Op:     "dummy",
		Net:    "dummy",
		Source: nil,
		Addr:   nil,
		Err: &os.SyscallError{
			Syscall: "dummy",
			Err:     syscall.ECONNREFUSED,
		},
	}
	err = neh.handleGenericErrors(ctx, opErr)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Connection refused", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test non-matching error
	err = neh.handleGenericErrors(ctx, errors.New("dummy error"))
	require.Equal(t, err, nil)
}

func TestHandleGenericExternalError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "503 Service Unavailable",
			input: `<html><body><h1>503 Service Unavailable</h1>\nNo server is available to handle this request.\n</body></html>\n\n`,
		},
		{
			name:  "HTTP Response Error",
			input: `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"Post \"https://evmos-grpc.w3coins.io:8545\": http: server gave HTTP response to HTTPS client"}}`,
		},
		{
			name:  "Rate Limit Error",
			input: `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"429 Too Many Requests: {\"code\":-32005,\"message\":\"daily request count exceeded, request rate limited\",\"data\":{\"rate\":{\"allowed_rps\":1,\"backoff_seconds\":30,\"current_rps\":1.4333333333333333},\"see\":\"https://infura.io/dashboard\"}}"}}`,
		},
		{
			name:  "Rate Limit Error - 2",
			input: `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"429 Too Many Requests: {\n  \"message\":\"API rate limit exceeded\"\n}"}}`,
		},
		{
			name: "502 Bad Gateway Error",
			input: `
			aptosRelayParse SyntaxError: Unexpected token '<', "<html>
			<h"... is not valid JSON
				at JSON.parse (<anonymous>)
				at Object.aptosRelayParse (page-2e73b2e0edee9f5d.js:1:2191)
				at j (page-2e73b2e0edee9f5d.js:1:2988) <html>
			<head><title>502 Bad Gateway</title></head>
			<body>
			<center><h1>502 Bad Gateway</h1></center>
			<hr><center>nginx/1.18.0 (Ubuntu)</center>
			</body>
			</html>`,
		},
		{
			name: "Undefined Property Error - with property",
			input: `error sending relay TypeError: Cannot read properties of undefined (reading 'header')
			at Object.cosmosRelayParse (page-2e73b2e0edee9f5d.js:1:2138)
			at j (page-2e73b2e0edee9f5d.js:1:2988)`,
		},
		{
			name: "Unhandled Relay Receiver Error",
			input: `
			{"Error_Received":"{\"Error_GUID\":\"GUID_9390299768\",\"Error\":\"Failed all retries -- [{Key:GUID Value:15668354495498650348} {Key:errors Value:[rpc error: code = Unknown desc = got called with unhandled relay receiver -- [{Key:requested_receiver Value:SOLANATjsonrpc} {Key:handled_receivers Value:LAV1grpc,LAV1tendermintrpc,LAV1rest}] No pairings available.]}]\"}"}`,
		},
		{
			name:  "RPC Network Unreachable Error",
			input: `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"Rpc Error","data":"Post \"http://127.0.0.0:45757\": dial tcp 127.0.0.0:45757: connect: network is unreachable"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandleGenericExternalError(tt.input)
			if err == nil {
				t.Fatalf("Expected an error but got nil")
			}
		})
	}
}
