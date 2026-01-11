package lavasession

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/lavanet/lava/v5/protocol/common"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// DirectRPCProtocol represents the transport protocol for direct RPC connections
type DirectRPCProtocol string

const (
	DirectRPCProtocolHTTP  DirectRPCProtocol = "http"
	DirectRPCProtocolHTTPS DirectRPCProtocol = "https"
	DirectRPCProtocolWS    DirectRPCProtocol = "ws"  // WebSocket
	DirectRPCProtocolWSS   DirectRPCProtocol = "wss" // WebSocket Secure
	DirectRPCProtocolGRPC  DirectRPCProtocol = "grpc"
)

// DirectRPCConnection represents a direct connection to an RPC endpoint
// (no Lava provider-relay protocol involved)
type DirectRPCConnection interface {
	// SendRequest sends the already-built raw request bytes and returns raw response bytes.
	//
	// IMPORTANT: DirectRPCConnection is transport-only. It must not need to interpret JSON-RPC
	// method/params or chain semantics. Those remain in chainParser + chainMessage.
	SendRequest(ctx context.Context, data []byte, headers map[string]string) ([]byte, error)

	// GetProtocol returns the transport protocol being used
	GetProtocol() DirectRPCProtocol

	// Close closes the connection and cleans up resources
	Close() error

	// IsHealthy returns true if the connection is healthy
	IsHealthy() bool

	// GetURL returns the endpoint URL
	GetURL() string

	// GetNodeUrl returns the NodeUrl configuration (for timeout overrides, auth, etc.)
	GetNodeUrl() *common.NodeUrl
}

// HTTPDirectRPCResponse contains complete HTTP response data (Phase 4 REST support)
type HTTPDirectRPCResponse struct {
	StatusCode int                 // HTTP status code (200, 404, 500, etc.)
	Headers    map[string][]string // Response headers (http.Header compatible)
	Body       []byte              // Response body
}

// HTTPRequestParams defines HTTP request parameters for REST support (Phase 4)
type HTTPRequestParams struct {
	Method      string                  // HTTP method: GET, POST, PUT, DELETE
	URL         string                  // Full URL (will be auth-transformed)
	Body        []byte                  // Request body (nil for GET)
	Headers     []pairingtypes.Metadata // ✅ Preserves delete semantics (empty value = delete)
	ContentType string                  // Content-Type (only for requests with body)
}

// HTTPDirectRPCDoer is an HTTP-only extension interface for REST support (Phase 4)
// Only HTTPDirectRPCConnection implements this (not WebSocket/gRPC)
type HTTPDirectRPCDoer interface {
	DirectRPCConnection // Inherits base interface

	// DoHTTPRequest executes an HTTP request with variable method/URL
	// Returns: Complete HTTP response (status + headers + body)
	DoHTTPRequest(ctx context.Context, params HTTPRequestParams) (*HTTPDirectRPCResponse, error)
}

// HTTPDirectRPCConnection implements DirectRPCConnection for HTTP/HTTPS
type HTTPDirectRPCConnection struct {
	nodeUrl  common.NodeUrl
	protocol DirectRPCProtocol
	client   *http.Client
}

// WebSocketDirectRPCConnection implements DirectRPCConnection for WebSocket/WSS
type WebSocketDirectRPCConnection struct {
	nodeUrl  common.NodeUrl
	protocol DirectRPCProtocol
}

// GRPCDirectRPCConnection implements DirectRPCConnection for gRPC
type GRPCDirectRPCConnection struct {
	nodeUrl common.NodeUrl
}

// NewDirectRPCConnection creates a new direct RPC connection based on URL protocol
func NewDirectRPCConnection(
	ctx context.Context,
	nodeUrl common.NodeUrl,
	parallelConnections uint,
) (DirectRPCConnection, error) {
	protocol, err := DetectProtocol(nodeUrl.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to detect protocol: %w", err)
	}

	switch protocol {
	case DirectRPCProtocolHTTP, DirectRPCProtocolHTTPS:
		return &HTTPDirectRPCConnection{
			nodeUrl:  nodeUrl,
			protocol: protocol,
			client:   &http.Client{},
		}, nil

	case DirectRPCProtocolWS, DirectRPCProtocolWSS:
		// WebSocket support is handled via a dedicated subscription/streaming layer.
		// See WEBSOCKET_SUPPORT.md for the design (connection lifecycle differs from request/response).
		return &WebSocketDirectRPCConnection{
			nodeUrl:  nodeUrl,
			protocol: protocol,
		}, nil

	case DirectRPCProtocolGRPC:
		return &GRPCDirectRPCConnection{
			nodeUrl: nodeUrl,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// DetectProtocol detects the RPC protocol from URL scheme
func DetectProtocol(urlStr string) (DirectRPCProtocol, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "http":
		return DirectRPCProtocolHTTP, nil
	case "https":
		return DirectRPCProtocolHTTPS, nil
	case "ws":
		return DirectRPCProtocolWS, nil
	case "wss":
		return DirectRPCProtocolWSS, nil
	case "grpc", "grpcs":
		return DirectRPCProtocolGRPC, nil
	default:
		// Default to HTTPS for URLs without explicit scheme
		if scheme == "" {
			return DirectRPCProtocolHTTPS, nil
		}
		return "", fmt.Errorf("unsupported URL scheme: %s", scheme)
	}
}

// SendRequest implements DirectRPCConnection for HTTP/HTTPS
func (h *HTTPDirectRPCConnection) SendRequest(
	ctx context.Context,
	data []byte,
	headers map[string]string,
) ([]byte, error) {
	// NOTE: for JSON-RPC we use POST with a JSON body.
	// For REST-style APIs (e.g. Cosmos LCD), the chain parser must provide the correct URL path and HTTP method
	// (GET/POST/etc.). This transport layer sends bytes and returns bytes; method selection is driven by chain spec.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.nodeUrl.AuthConfig.AddAuthPath(h.nodeUrl.Url), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Apply auth headers from config + per-request headers from chainMessage/chainParser.
	for k, v := range h.nodeUrl.GetAuthHeaders() {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %w", err)
	}

	// Check HTTP status code and return error for non-2xx responses
	// Note: For JSON-RPC, status 200 with RPC error in body is common and valid
	// We return the body in both cases - the caller will check for RPC errors
	if resp.StatusCode >= 400 {
		// For 4xx/5xx errors, include status code in error
		return body, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       body,
		}
	}

	return body, nil
}

// HTTPStatusError represents an HTTP error response (4xx/5xx)
type HTTPStatusError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d %s", e.StatusCode, e.Status)
}

func (h *HTTPDirectRPCConnection) GetProtocol() DirectRPCProtocol {
	return h.protocol
}

func (h *HTTPDirectRPCConnection) Close() error {
	return nil
}

func (h *HTTPDirectRPCConnection) IsHealthy() bool {
	return true // health tracking is done at the endpoint/QoS layer
}

func (h *HTTPDirectRPCConnection) GetURL() string {
	return h.nodeUrl.Url
}

func (h *HTTPDirectRPCConnection) GetNodeUrl() *common.NodeUrl {
	return &h.nodeUrl
}

// DoHTTPRequest implements HTTPDirectRPCDoer for REST support (Phase 4)
// Executes HTTP request with variable method (GET/POST/PUT/DELETE)
func (h *HTTPDirectRPCConnection) DoHTTPRequest(
	ctx context.Context,
	params HTTPRequestParams,
) (*HTTPDirectRPCResponse, error) {
	// Build request body reader
	var bodyReader io.Reader
	if params.Body != nil {
		bodyReader = bytes.NewReader(params.Body)
	}

	// ✅ Apply auth path transformation (e.g., prepend /api/v1 if configured)
	fullURL := h.nodeUrl.AuthConfig.AddAuthPath(params.URL)

	req, err := http.NewRequestWithContext(ctx, params.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}

	// ✅ Apply NodeUrl auth headers (API keys, bearer tokens, etc.)
	h.nodeUrl.SetAuthHeaders(ctx, req.Header.Set)

	// ✅ Apply IP forwarding if needed
	h.nodeUrl.SetIpForwardingIfNecessary(ctx, req.Header.Set)

	// ✅ Apply per-request headers with delete semantics
	for _, header := range params.Headers {
		if header.Value == "" {
			req.Header.Del(header.Name) // Empty value = delete header
		} else {
			req.Header.Set(header.Name, header.Value)
		}
	}

	// Set Content-Type for requests with body
	if params.Body != nil && params.ContentType != "" {
		req.Header.Set("Content-Type", params.ContentType)
	}

	// Send request
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response: %w", err)
	}

	// ✅ Return complete response (status + headers + body)
	// Don't return error for 4xx/5xx - client needs the response
	return &HTTPDirectRPCResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}

// SendRequest implements DirectRPCConnection for WebSocket/WSS
func (w *WebSocketDirectRPCConnection) SendRequest(
	ctx context.Context,
	data []byte,
	headers map[string]string,
) ([]byte, error) {
	return nil, fmt.Errorf("WebSocket SendRequest not implemented; use subscription/streaming flow")
}

func (w *WebSocketDirectRPCConnection) GetProtocol() DirectRPCProtocol {
	return w.protocol
}

func (w *WebSocketDirectRPCConnection) Close() error {
	return nil
}

func (w *WebSocketDirectRPCConnection) IsHealthy() bool {
	return true // health tracking is done at the endpoint/QoS layer
}

func (w *WebSocketDirectRPCConnection) GetURL() string {
	return w.nodeUrl.Url
}

func (w *WebSocketDirectRPCConnection) GetNodeUrl() *common.NodeUrl {
	return &w.nodeUrl
}

// SendRequest implements DirectRPCConnection for gRPC
func (g *GRPCDirectRPCConnection) SendRequest(
	ctx context.Context,
	data []byte,
	headers map[string]string,
) ([]byte, error) {
	// TODO: Implement gRPC request sending
	// This will depend on the specific gRPC service definition
	return nil, fmt.Errorf("gRPC direct connections not yet implemented")
}

func (g *GRPCDirectRPCConnection) GetProtocol() DirectRPCProtocol {
	return DirectRPCProtocolGRPC
}

func (g *GRPCDirectRPCConnection) Close() error {
	return nil
}

func (g *GRPCDirectRPCConnection) IsHealthy() bool {
	return true // health tracking is done at the endpoint/QoS layer
}

func (g *GRPCDirectRPCConnection) GetURL() string {
	return g.nodeUrl.Url
}

func (g *GRPCDirectRPCConnection) GetNodeUrl() *common.NodeUrl {
	return &g.nodeUrl
}
