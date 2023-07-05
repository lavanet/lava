package chainlib

import (
	"context"
	"io"
	"net"
	"os"
	"syscall"

	"github.com/lavanet/lava/utils"
)

type NodeErrorHandler struct {
}

func (neh *NodeErrorHandler) handleConnectionError(err error) error {
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
	return nil // Return original error if it doesn't match any specific cases
}

func (neh *NodeErrorHandler) HandleGenericErrors(ctx context.Context, nodeError error) error {
	if ctx.Err() == context.DeadlineExceeded {
		return utils.LavaFormatError("Provider Failed Sending Message", context.DeadlineExceeded)
	}
	return neh.handleConnectionError(nodeError)
}

func (neh *NodeErrorHandler) HandleRestErrors(ctx context.Context, nodeError error) error {
	if err := neh.HandleGenericErrors(ctx, nodeError); err != nil {
		// printing the original error as  it was masked for the consumer to not see the private information such as ip address etc..
		utils.LavaFormatError("Original Node Error", nodeError)
		return err
	}
	return nil
}

func (neh *NodeErrorHandler) HandleJsonRPCErrors(ctx context.Context, nodeError error) error {
	if err := neh.HandleGenericErrors(ctx, nodeError); err != nil {
		utils.LavaFormatError("Original Node Error", nodeError)
		return err
	}
	return nil
}

func (neh *NodeErrorHandler) HandleGRPCErrors(ctx context.Context, nodeError error) error {
	if err := neh.HandleGenericErrors(ctx, nodeError); err != nil {
		utils.LavaFormatError("Original Node Error", nodeError)
		return err
	}
	return nil
}

func (neh *NodeErrorHandler) HandleTendermintRPCErrors(ctx context.Context, nodeError error) error {
	if err := neh.HandleGenericErrors(ctx, nodeError); err != nil {
		utils.LavaFormatError("Original Node Error", nodeError)
		return err
	}
	return nil
}
