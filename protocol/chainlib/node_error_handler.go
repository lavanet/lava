package chainlib

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v2/protocol/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lavanet/lava/v2/utils"
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

func (geh *genericErrorHandler) handleCodeErrors(code codes.Code) error {
	if code == codes.DeadlineExceeded {
		return utils.LavaFormatProduction("Provider Failed Sending Message", common.ContextDeadlineExceededError)
	}
	switch code {
	case codes.PermissionDenied, codes.Canceled, codes.Aborted, codes.DataLoss, codes.Unauthenticated, codes.Unavailable:
		return utils.LavaFormatProduction("Provider Side Failed Sending Message, Reason: "+code.String(), nil)
	}
	return nil
}

func (geh *genericErrorHandler) HandleStatusError(statusCode int, strict bool) error {
	return rpcclient.ValidateStatusCodes(statusCode, strict)
}

func (geh *genericErrorHandler) HandleJSONFormatError(replyData []byte) error {
	var jsonData map[string]interface{}
	err := json.Unmarshal(replyData, &jsonData)
	if err != nil {
		// if failed to parse might be an array of jsons
		var jsonArrayData []map[string]interface{}
		parsingErr := json.Unmarshal(replyData, &jsonArrayData)
		if parsingErr != nil {
			return utils.LavaFormatError("Rest reply is not in JSON format", err, utils.Attribute{Key: "reply.Data", Value: string(replyData)})
		}
	}
	return nil
}

func (geh *genericErrorHandler) ValidateRequestAndResponseIds(nodeMessageID json.RawMessage, replyMsgID json.RawMessage) error {
	reqId, idErr := rpcInterfaceMessages.IdFromRawMessage(nodeMessageID)
	if idErr != nil {
		return fmt.Errorf("Failed parsing ID " + idErr.Error())
	}
	respId, idErr := rpcInterfaceMessages.IdFromRawMessage(replyMsgID)
	if idErr != nil {
		return fmt.Errorf("Failed parsing ID " + idErr.Error())
	}
	if reqId != respId {
		return fmt.Errorf("ID mismatch error")
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
		return geh.handleCodeErrors(st.Code())
	}
	return geh.handleGenericErrors(ctx, nodeError)
}

type ErrorHandler interface {
	HandleNodeError(context.Context, error) error
	HandleStatusError(int, bool) error
	HandleJSONFormatError([]byte) error
	ValidateRequestAndResponseIds(json.RawMessage, json.RawMessage) error
}
