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
	// 1 Well-formed error response, with an allowed error code
	allowedErrorResponse := `{"code":-32602,"message":"invalid argument 0: json: cannot unmarshal hex string without 0x prefix into Go value of type common.Hash"}`
	err := jeh.HandleExternalError(allowedErrorResponse)
	if err != nil {
		t.Errorf("Expected nil error for allowed error code, got: %v", err)
	}
	// 2. Well-formed error response, with an disallowed error code
	disallowedErrorResponse := `429 Too Many Requests: {\"code\":-32005,\"message\":\"daily request count exceeded, request rate limited\",\"data\":{\"rate\":{\"allowed_rps\":1,\"backoff_seconds\":30,\"current_rps\":1.4333333333333333},\"see\":\"https://infura.io/dashboard\"}}`
	err = jeh.HandleExternalError(disallowedErrorResponse)
	if err == nil {
		t.Errorf("Expected not nil error for allowed error code, got: %v", err)
	}
	// 3 Ill-formed error response
	illErrorResponse := `<head><title>502 Bad Gateway</title></head>
	<body>
	<center><h1>502 Bad Gateway</h1></center>
	<hr><center>nginx/1.18.0 (Ubuntu)</center>
	</body>
	</html>`
	err = jeh.HandleExternalError(illErrorResponse)
	if err == nil {
		t.Errorf("Expected not nil error for allowed error code, got: %v", err)
	}
}

func TestHandleExternalErrorTendermintRPC(t *testing.T) {
	jeh := &JsonRPCErrorHandler{}
	// 1 Well-formed error response, with an allowed error code
	allowedErrorResponse := `{"code":-32602,"message":"Invalid params","data":"error converting http params to arguments: invalid character 'B' after top-level value"}`
	err := jeh.HandleExternalError(allowedErrorResponse)
	if err != nil {
		t.Errorf("Expected nil error for allowed error code, got: %v", err)
	}
}

func TestHandleExternalErrorForREST(t *testing.T) {
	handler := RestErrorHandler{}

	// Error response 1: "height must be greater than 0, but got -3"
	replyData1 := `{"code":14,"message":"invalid address","details":[]}`
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

	// Ill-formed error response
	illData := ` 429 Too Many Requests: <html>
	<head><title>429 Too Many Requests</title></head>
	<body>
	<center><h1>429 Too Many Requests</h1></center>
	<hr><center>nginx/1.18.0 (Ubuntu)</center>
	</body>
	</html>`
	err = handler.HandleExternalError(illData)
	if err == nil {
		t.Errorf("Expected not nil for an ill response, got %s", err.Error())
	}

}
