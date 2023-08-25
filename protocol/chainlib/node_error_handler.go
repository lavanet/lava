package chainlib

import (
	"context"
	"io"
	"net"
	"os"
	"regexp"
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

func HandleGenericExternalError(replyData string) error {
	// REGEX patterns
	infuraRateLimitPattern := regexp.MustCompile(`429 Too Many Requests.*daily request count exceeded, request rate limited`)

	switch {
	// Check Bad Gateway
	case strings.Contains(replyData, "502 Bad Gateway"):
		return utils.LavaFormatError("Provider Returned External Error: 502 Bad Gateway", nil, utils.Attribute{Key: "Reply:", Value: replyData})
		// Check for "Cannot read properties of undefined" errors
	case strings.Contains(replyData, "Cannot read properties of undefined"):
		// Extract the property being accessed, if mentioned
		var property string
		matches := regexp.MustCompile(`\(reading '(.+?)'\)`).FindStringSubmatch(replyData)
		if len(matches) > 1 {
			property = matches[1]
		}
		return utils.LavaFormatError("Provider Returned External Error: Attempted to access an undefined object property", nil, utils.Attribute{Key: "Property:", Value: property}, utils.Attribute{Key: "Reply:", Value: replyData})
		// Check for "got called with unhandled relay receiver" errors
	case strings.Contains(replyData, "got called with unhandled relay receiver"):
		// Extract requested and handled receivers
		requestedReceiverMatch := regexp.MustCompile(`{Key:requested_receiver Value:(.+?)}`).FindStringSubmatch(replyData)
		handledReceiversMatch := regexp.MustCompile(`{Key:handled_receivers Value:(.+?)}`).FindStringSubmatch(replyData)

		var requestedReceiver, handledReceivers string
		if len(requestedReceiverMatch) > 1 {
			requestedReceiver = requestedReceiverMatch[1]
		}
		if len(handledReceiversMatch) > 1 {
			handledReceivers = handledReceiversMatch[1]
		}
		return utils.LavaFormatError("Provider Returned External Error: Unhandled relay receiver", nil, utils.Attribute{Key: "Reqested Receiver:", Value: requestedReceiver}, utils.Attribute{Key: "Handled Receiver:", Value: handledReceivers}, utils.Attribute{Key: "Reply:", Value: replyData})
		// Check rate limit
	case infuraRateLimitPattern.MatchString(replyData):
		return utils.LavaFormatError("Provider Returned External Error: Infura Rate Limit Error - Too Many Requests.", nil, utils.Attribute{Key: "Reply:", Value: replyData})
		// Check unreachable network
	case strings.Contains(replyData, `"message":"Rpc Error"`) && strings.Contains(replyData, `connect: network is unreachable`):
		return utils.LavaFormatError("Provider Returned External Error:RPC Network Unreachable. The target service might be down or there might be a network issue.", nil, utils.Attribute{Key: "Reply:", Value: replyData})

	default:
	}
	return nil
}

// External Errors
func (geh *RestErrorHandler) HandleExternalError(replyData string) error {
	return nil
}

func (jeh *JsonRPCErrorHandler) HandleExternalError(replyData string) error {
	// switch {
	// case strings.Contains(replyData, `429 Too Many Requests`) && strings.Contains(replyData, `"daily request count exceeded, request rate limited"`):
	// 	errMsg := "Infura Rate Limit Error: Too Many Requests. Daily request count exceeded; request rate limited."
	// 	return utils.LavaFormatError(errMsg, nil, utils.Attribute{Key: "Reply:", Value: replyData})

	// default:
	// 	// Default handling if no specific condition is met
	// }
	return nil
}

func (teh *TendermintRPCErrorHandler) HandleExternalError(replyData string) error {
	// switch {
	// case strings.Contains(replyData, `"message":"Rpc Error"`) && strings.Contains(replyData, `connect: network is unreachable`):
	// 	errMsg := "JSON-RPC Error: RPC Network Unreachable. The target service might be down or there might be a network issue."
	// 	return utils.LavaFormatError(errMsg, nil, utils.Attribute{Key: "Reply:", Value: replyData})

	// }
	return nil
}

func (teh *GRPCErrorHandler) HandleExternalError(replyData string) error {
	// Add checks for GRPC API ...
	return nil
}
