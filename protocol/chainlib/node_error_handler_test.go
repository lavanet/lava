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

func TestHandleExternalErrorJSONRPC(t *testing.T) {
	jeh := &JsonRPCErrorHandler{}

	// 1. Well-formed error response, with an allowed error code
	allowedErrorResponse := `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"429 Too Many Requests: {\"code\":-32001,\"message\":\"some allowed error\",\"data\":{\"some_field\":\"some_value\"}}"}}`
	err := jeh.HandleExternalError(allowedErrorResponse)
	if err != nil {
		t.Errorf("Expected nil error for allowed error code, got: %v", err)
	}

	// 2. Well-formed error response, but with a disallowed error code
	disallowedErrorResponse := `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"429 Too Many Requests: {\"code\":-32005,\"message\":\"some disallowed error\",\"data\":{\"some_field\":\"some_value\"}}"}}`
	err = jeh.HandleExternalError(disallowedErrorResponse)
	if err == nil {
		t.Errorf("Expected non-nil error for disallowed error code")
	}

	// 3. Ill-formed error response
	illFormedResponse := `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"some random string that doesn't contain a nested code"}}`
	err = jeh.HandleExternalError(illFormedResponse)
	if err == nil {
		t.Errorf("Expected non-nil error for ill-formed response")
	}

	// 4. Well-formed successful relay with "result" field
	successfulRelayResponse := `{"jsonrpc":"2.0","id":1,"result":{"key":"value"}}`
	err = jeh.HandleExternalError(successfulRelayResponse)
	if err != nil {
		t.Errorf("Expected nil error for successful relay, got: %v", err)
	}
}

func TestHandleExternalErrorForREST(t *testing.T) {
	handler := RestErrorHandler{}

	// Error response 1: "height must be greater than 0, but got -3"
	replyData1 := `{"code":2,"message":"height must be greater than 0, but got -3","details":[]}`
	err := handler.HandleExternalError(replyData1)
	if err != nil {
		t.Errorf("Expected nil, got %s", err.Error())
	}

	// Error response 2: "requested block height is bigger then the chain length"
	replyData2 := `{"code":3,"message":"requested block height is bigger then the chain length","details":[]}`
	err = handler.HandleExternalError(replyData2)
	if err != nil {
		t.Errorf("Expected nil, got %s", err.Error())
	}

	// Error response 3: Unexpected error code
	replyData3 := `{"code":999,"message":"unknown error","details":[]}`
	err = handler.HandleExternalError(replyData3)
	if err == nil {
		t.Errorf("Expected an error, got nil")
	}

	// Successful response 2: Another simplified example of a successful response
	successfulReply2 := `{"block_id":{"hash":"anotherHash","part_set_header":{"total":1,"hash":"anotherHash"}}}`
	err = handler.HandleExternalError(successfulReply2)
	if err != nil {
		t.Errorf("Expected nil for a successful response, got %s", err.Error())
	}

	// Malformed response: Neither a successful nor an error response
	malformedReply := `{"unknownField":"unknownValue"}`
	err = handler.HandleExternalError(malformedReply)
	if err == nil {
		t.Errorf("Expected an error for a malformed response, got nil")
	}
}
