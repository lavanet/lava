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
	errorCode, err := extractRPCNestedCode(errorMessage)
	if err != nil {
		return utils.LavaFormatProduction("Unparsable external JsonRPC error detected", err, utils.Attribute{Key: "errorMessage", Value: errorMessage})
	}
	// Check if this internal error code is in our map of allowed errors
	if allowedErrors, ok := AllowedErrorsMap["jsonrpc"]; !ok {
		return utils.LavaFormatProduction("Allowed errors for json-RPC is not configured", nil)
	} else if _, exists := allowedErrors[strconv.Itoa(errorCode)]; !exists {
		return utils.LavaFormatProduction(fmt.Sprintf("Disallowed provider error code: %d", errorCode), nil)
	}

	utils.LavaFormatInfo("Allowed error detected in JSONRPC response:", utils.Attribute{Key: "Allowed Error", Value: errorMessage})
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
	// Extract the nested error code from the error message
	errorCode, err := extractRPCNestedCode(errorMessage)
	if err != nil {
		return utils.LavaFormatProduction("Unparsable external TendermintRPC error detected", err, utils.Attribute{Key: "errorMessage", Value: errorMessage})
	}
	// Check if this internal error code is in our map of allowed errors
	if allowedErrors, ok := AllowedErrorsMap["tendermintrpc"]; !ok {
		return utils.LavaFormatProduction("Allowed errors for tendermintRPC is not configured", nil)
	} else if _, exists := allowedErrors[strconv.Itoa(errorCode)]; !exists {
		return utils.LavaFormatProduction(fmt.Sprintf("Disallowed provider error code: %d", errorCode), nil)
	}

	utils.LavaFormatInfo("Allowed error detected in tendermintRPC response:", utils.Attribute{Key: "Allowed Error", Value: errorMessage})
	return nil
}

func (geh *GRPCErrorHandler) HandleExternalError(replyData string) error {
	return nil
}

func extractRPCNestedCode(message string) (int, error) {
	var jsonSubStr string
	if strings.Contains(message, "\\\"") {
		// The message contains escaped characters, try to unquote it
		unquotedMessage, err := strconv.Unquote("\"" + message + "\"")
		if err != nil {
			return 0, utils.LavaFormatProduction("Failed to unquote the message", err)
		}

		// Find where the JSON starts and ends in the unquoted message
		jsonStart := strings.Index(unquotedMessage, "{\"code\":")
		if jsonStart == -1 {
			return 0, utils.LavaFormatProduction("Cannot find JSON object in message", nil)
		}
		jsonSubStr = unquotedMessage[jsonStart:]
		jsonEnd := strings.LastIndex(jsonSubStr, "}")
		if jsonEnd == -1 {
			return 0, utils.LavaFormatProduction("Cannot find the end of the JSON object in message", nil)
		}

		// Extract the JSON substring
		jsonSubStr = jsonSubStr[:jsonEnd+1]
	} else {
		// The message seems to already be a JSON object
		jsonSubStr = message
	}

	// Now try to unmarshal it
	var parsedMessage map[string]interface{}
	if err := json.Unmarshal([]byte(jsonSubStr), &parsedMessage); err != nil {
		return 0, utils.LavaFormatProduction("Cannot unmarshal the JSON substring", err)
	}

	// Extract the code from the parsed JSON
	code, exists := parsedMessage["code"].(float64)
	if !exists {
		return 0, utils.LavaFormatProduction("Code field not found", nil)
	}

	return int(code), nil
}
