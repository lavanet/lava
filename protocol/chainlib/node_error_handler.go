package chainlib

import (
	"context"
	"io"
	"net"
	"os"
	"syscall"

	"github.com/lavanet/lava/protocol/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lavanet/lava/utils"
)

type genericErrorHandler struct{}

func (geh *genericErrorHandler) handleConnectionError(err error) error {
	if err == net.ErrWriteToConnected {
		return utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Write to connected connection", nil)
	} else if err == net.ErrClosed {
		return utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Operation on closed connection", nil)
	} else if err == io.EOF {
		return utils.LavaFormatError("Provider Side Failed Sending Message, Reason: End of input stream reached", nil)
	} else if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
		return utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Network operation timed out", nil)
	} else if _, ok := err.(*net.DNSError); ok {
		return utils.LavaFormatError("Provider Side Failed Sending Message, Reason: DNS resolution failed", nil)
	} else if opErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := opErr.Err.(*os.SyscallError); ok && sysErr.Err == syscall.ECONNREFUSED {
			return utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Connection refused", nil)
		}
	}
	return nil // do not return here so the caller will return the error inside the data so it reaches the user when it doesn't match any specific cases
}

func (geh *genericErrorHandler) handleGenericErrors(ctx context.Context, nodeError error) error {
	if nodeError == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded {
		return utils.LavaFormatError("Provider Failed Sending Message", common.ContextDeadlineExceededError)
	}
	retError := geh.handleConnectionError(nodeError)
	if retError != nil {
		// printing the original error as  it was masked for the consumer to not see the private information such as ip address etc..
		utils.LavaFormatError("Original Node Error", nodeError)
	}
	return retError
}

func (geh *genericErrorHandler) handleCodeErrors(ctx context.Context, code codes.Code) error {
	if code == codes.DeadlineExceeded {
		return utils.LavaFormatError("Provider Failed Sending Message", common.ContextDeadlineExceededError)
	}
	switch code {
	case codes.PermissionDenied, codes.Canceled, codes.Aborted, codes.DataLoss, codes.Unauthenticated, codes.Unavailable:
		return utils.LavaFormatError("Provider Side Failed Sending Message, Reason: "+code.String(), nil)
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
}
