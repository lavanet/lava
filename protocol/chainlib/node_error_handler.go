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
	HandleExternalError(errorMessage string) error
}

// HandleExternalError handles external errors for JSON-RPC calls.
func (jeh *JsonRPCErrorHandler) HandleExternalError(errorMessage string) error {
	// Extract the nested error code from the error message
	nestedCode, err := extractRPCNestedCode(errorMessage)
	if err != nil {
		return utils.LavaFormatProduction("Disallowed error detected in relay response", err, utils.Attribute{Key: "errorMessage:", Value: errorMessage})
	}
	// Check if this internal error code is in our map of allowed errors
	allowedErrors, ok := AllowedErrorsMap["jsonrpc"]
	if !ok {
		return utils.LavaFormatProduction("Allowed errors for json-RPC is not configured", nil)
	}
	_, exists := allowedErrors[fmt.Sprintf("%d", nestedCode)]
	if !exists {
		// If the error code is not allowed, return a node error
		errMsg := fmt.Sprintf("Disallowed provider error code: %d", nestedCode)
		return utils.LavaFormatProduction(errMsg, nil)
	}
	return nil
}

type RESTErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (geh *RestErrorHandler) HandleExternalError(replyData string) error {
	var restError RESTErrorResponse
	var jsonData map[string]interface{}
	// Try to unmarshal the data to see if it is a RESTErrorResponse
	err := json.Unmarshal([]byte(replyData), &restError)
	if err == nil && restError.Code != 0 && restError.Message != "" {
		// Check if the code is in the allowed errors list
		allowedErrors, ok := AllowedErrorsMap["rest"]
		if !ok {
			return utils.LavaFormatProduction("allowed errors for REST not configured", nil)
		}
		if _, exists := allowedErrors[fmt.Sprintf("%d", restError.Code)]; !exists {
			return utils.LavaFormatProduction(fmt.Sprintf("Disallowed provider error code: %d", restError.Code), nil)
		}
		utils.LavaFormatInfo("Allowed error detected in REST response:", utils.Attribute{Key: "Allowed Error", Value: restError})
		return nil
	}

	// Try to unmarshal the data to see if it is a general JSON map
	err = json.Unmarshal([]byte(replyData), &jsonData)
	if err != nil {
		return utils.LavaFormatProduction("Unparsable external REST error detected.", err)
	}

	return nil
}

func (te *TendermintRPCErrorHandler) HandleExternalError(errorMessage string) error {
	nestedCode, err := extractRPCNestedCode(errorMessage)
	if err != nil {
		return utils.LavaFormatProduction("Disallowed error detected in relay response", err, utils.Attribute{Key: "errorMessage:", Value: errorMessage})
	}
	// Check if this internal error code is in our map of allowed errors
	allowedErrors, ok := AllowedErrorsMap["tendermint"]
	if !ok {
		return utils.LavaFormatProduction("Allowed errors for tendermint-RPC is not configured", nil)
	}
	_, exists := allowedErrors[fmt.Sprintf("%d", nestedCode)]
	if !exists {
		// If the error code is not allowed, return a node error
		errMsg := fmt.Sprintf("Disallowed provider error code: %d", nestedCode)
		return utils.LavaFormatProduction(errMsg, nil)
	}
	return nil
}

func (geh *GRPCErrorHandler) HandleExternalError(replyData string) error {
	return nil
}

func extractRPCNestedCode(message string) (int, error) {
	idx := strings.Index(message, "\"code\":")
	if idx == -1 {
		return 0, utils.LavaFormatError("cannot extract nested error code, code field not found", nil)
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
