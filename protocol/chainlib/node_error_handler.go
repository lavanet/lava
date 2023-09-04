package chainlib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/lavanet/lava/protocol/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lavanet/lava/utils"
)

type genericErrorHandler struct{}

func (geh *genericErrorHandler) handleConnectionError(err error) error {
	if err == net.ErrWriteToConnected {
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: Write to connected connection", nil)
	} else if err == net.ErrClosed {
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: Operation on closed connection", nil)
	} else if err == io.EOF {
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: End of input stream reached", nil)
	} else if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: Network operation timed out", nil)
	} else if _, ok := err.(*net.DNSError); ok {
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: DNS resolution failed", nil)
	} else if opErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := opErr.Err.(*os.SyscallError); ok && sysErr.Err == syscall.ECONNREFUSED {
			return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: Connection refused", nil)
		}
	} else if strings.Contains(err.Error(), "http: server gave HTTP response to HTTPS client") {
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: misconfigured http endpoint as https", nil)
	}
	return nil // do not return here so the caller will return the error inside the data so it reaches the user when it doesn't match any specific cases
}

func (geh *genericErrorHandler) handleGenericErrors(ctx context.Context, nodeError error) error {
	if nodeError == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded {
		return utils.LavaFormatProduction("Provider Failed Sending Message", common.ContextDeadlineExceededError)
	}
	retError := geh.handleConnectionError(nodeError)
	if retError != nil {
		// printing the original error as  it was masked for the consumer to not see the private information such as ip address etc..
		utils.LavaFormatProduction("Original Node Error", nodeError)
	}
	return retError
}

func (geh *genericErrorHandler) handleCodeErrors(ctx context.Context, code codes.Code) error {
	if code == codes.DeadlineExceeded {
		return utils.LavaFormatProduction("Provider Failed Sending Message", common.ContextDeadlineExceededError)
	}
	switch code {
	case codes.PermissionDenied, codes.Canceled, codes.Aborted, codes.DataLoss, codes.Unauthenticated, codes.Unavailable:
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: "+code.String(), nil)
	}
	return nil
}

type RestErrorHandler struct{ genericErrorHandler }

// Validating if the error is related to the provider connection or not
// returning nil if its not one of the expected connectivity error types
func (rne *RestErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	return rne.handleGenericErrors(ctx, nodeError)
}

type JsonRPCErrorHandler struct{ genericErrorHandler }

func (jeh *JsonRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	return jeh.handleGenericErrors(ctx, nodeError)
}

type TendermintRPCErrorHandler struct{ genericErrorHandler }

func (tendermintErrorHandler *TendermintRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	return tendermintErrorHandler.handleGenericErrors(ctx, nodeError)
}

type GRPCErrorHandler struct{ genericErrorHandler }

func (geh *GRPCErrorHandler) HandleNodeError(ctx context.Context, nodeError error) error {
	st, ok := status.FromError(nodeError)
	if ok {
		// Get the error message from the gRPC status
		return geh.handleCodeErrors(ctx, st.Code())
	}
	return geh.handleGenericErrors(ctx, nodeError)
}

type ErrorHandler interface {
	HandleNodeError(context.Context, error) error
	HandleExternalError(replyData string) error
}

type JsonResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func extractNestedCode(message string) (int, error) {
	idx := strings.Index(message, "\"code\":")
	if idx == -1 {
		return 0, utils.LavaFormatError("code field not found", nil)
	}

	start := idx + len("\"code\":")
	end := strings.Index(message[start:], ",")
	if end == -1 {
		end = len(message) - start
	}

	codeStr := message[start : start+end]
	nestedCode, err := strconv.Atoi(strings.TrimSpace(codeStr))
	if err != nil {
		return 0, utils.LavaFormatError("invalid code field", err)
	}

	return nestedCode, nil
}

// External Errors
func (jeh *JsonRPCErrorHandler) HandleExternalError(replyData string) error {
	// Try to parse the reply into a JsonRPCResponse
	var jsonResponse JsonResponse
	err := json.Unmarshal([]byte(replyData), &jsonResponse)
	if err != nil {
		return utils.LavaFormatProduction("Unparsable external provider error detected.", err)
	}
	// Check if there is an "error" in the response
	if jsonResponse.Error.Code != 0 {
		nestedCode, err := extractNestedCode(jsonResponse.Error.Message)
		if err != nil {
			return utils.LavaFormatProduction("Cannot extract nested error code.", err)
		}
		// Check if this internal error code is in our map of allowed errors
		_, exists := AllowedErrorsMap["jsonrpc"][fmt.Sprintf("%d", nestedCode)]
		if !exists {
			// If the error code is not allowed, return a node error
			errMsg := fmt.Sprintf("Disallowed provider error code: %d", nestedCode)
			return utils.LavaFormatProduction(errMsg, nil)
		}
	}
	return nil
}

func (geh *RestErrorHandler) HandleExternalError(replyData string) error {
	return nil
}

func (teh *TendermintRPCErrorHandler) HandleExternalError(replyData string) error {
	return nil
}

func (teh *GRPCErrorHandler) HandleExternalError(replyData string) error {
	return nil
}
