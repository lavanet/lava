package rpcsmartrouter

import (
	"errors"
	"net"
	"syscall"

	"github.com/lavanet/lava/v5/protocol/common"
)

// ClassifyDirectRPCError classifies a direct RPC error into a LavaError for
// internal use (logging, metrics, endpoint health). The original error is never
// modified — the router is a transparent hop for the user.
func ClassifyDirectRPCError(err error) *common.LavaError {
	if err == nil {
		return nil
	}

	// Connection-level errors — detected before inspecting the message
	var connError *common.LavaError
	if isConnectionRefused(err) {
		connError = common.LavaErrorConnectionRefused
	} else if isTimeout(err) {
		connError = common.LavaErrorConnectionTimeout
	}

	// Classify: connection error takes precedence, otherwise classify from message.
	// Smart router doesn't know chain family at this point — use EVM as default
	// since JSON-RPC is the most common transport for direct RPC.
	classified := common.ClassifyError(connError, common.ChainFamilyEVM, common.TransportJsonRPC, 0, err.Error())

	common.LogCodedError("direct RPC error", err, classified, "", 0, err.Error())

	return classified
}

func isConnectionRefused(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		var syscallErr *syscall.Errno
		if errors.As(netErr.Err, &syscallErr) {
			return *syscallErr == syscall.ECONNREFUSED
		}
	}
	return false
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
