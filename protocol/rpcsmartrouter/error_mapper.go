package rpcsmartrouter

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"

	"github.com/lavanet/lava/v5/protocol/lavasession"
)

// MapDirectRPCError maps direct RPC errors to user-friendly errors
func MapDirectRPCError(err error, protocol lavasession.DirectRPCProtocol) error {
	if err == nil {
		return nil
	}

	// Connection errors
	if isConnectionRefused(err) {
		return fmt.Errorf("RPC endpoint unavailable (connection refused): %w", err)
	}

	if isTimeout(err) {
		return fmt.Errorf("RPC request timeout: %w", err)
	}

	// Protocol-specific error handling
	switch protocol {
	case lavasession.DirectRPCProtocolHTTP, lavasession.DirectRPCProtocolHTTPS:
		return mapHTTPError(err)
	case lavasession.DirectRPCProtocolGRPC:
		return mapGRPCError(err)
	case lavasession.DirectRPCProtocolWS, lavasession.DirectRPCProtocolWSS:
		return mapWebSocketError(err)
	}

	return err
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

func mapHTTPError(err error) error {
	// Check for HTTP status codes (429, 503, etc.)
	errStr := err.Error()
	if strings.Contains(errStr, "429") {
		return fmt.Errorf("RPC rate limit exceeded: %w", err)
	}
	if strings.Contains(errStr, "503") {
		return fmt.Errorf("RPC service unavailable: %w", err)
	}
	if strings.Contains(errStr, "500") {
		return fmt.Errorf("RPC internal server error: %w", err)
	}
	if strings.Contains(errStr, "502") || strings.Contains(errStr, "504") {
		return fmt.Errorf("RPC gateway error: %w", err)
	}
	return err
}

func mapGRPCError(err error) error {
	// Map gRPC status codes
	// TODO: Add gRPC-specific error mapping when gRPC support is implemented
	return err
}

func mapWebSocketError(err error) error {
	// Map WebSocket-specific errors
	// TODO: Add WebSocket-specific error mapping when WS support is implemented
	return err
}
