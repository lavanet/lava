package grpcwebcompat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testMethod = "/lavanet.test.Echo/Echo"

// rawCodec passes message bytes through untouched, the way the gRPC gateway's proxy
// codec does, so these tests do not need generated protobuf types.
type rawCodec struct{}

func (rawCodec) Marshal(v interface{}) ([]byte, error) {
	payload, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("rawCodec cannot marshal %T", v)
	}
	return payload, nil
}

func (rawCodec) Unmarshal(data []byte, v interface{}) error {
	buffer, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("rawCodec cannot unmarshal into %T", v)
	}
	*buffer = data
	return nil
}
func (rawCodec) Name() string   { return "lava/test-raw-codec" }
func (rawCodec) String() string { return rawCodec{}.Name() }

// newEchoServer returns a gRPC server that echoes the request back, sets a response
// header and a trailer, and then fails with handlerErr when it is not nil. The message
// is sent before the error so the status has to travel in the trailer frame; a handler
// that fails without sending anything produces a Trailers-Only response instead.
func newEchoServer(handlerErr error) *grpc.Server {
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		var reqBytes []byte
		if err := stream.RecvMsg(&reqBytes); err != nil {
			return err
		}
		// a value carrying the optional whitespace a misbehaving upstream may have added
		stream.SetTrailer(metadata.Pairs("x-padded-trailer", "  padded  "))
		if err := stream.SetHeader(metadata.Pairs("x-node-height", "12345")); err != nil {
			return err
		}
		if err := stream.SendMsg(append([]byte("echo-"), reqBytes...)); err != nil {
			return err
		}
		return handlerErr
	}
	return grpc.NewServer(grpc.UnknownServiceHandler(handler), grpc.ForceServerCodec(rawCodec{}))
}

// newFailFastServer returns a gRPC server that fails before writing anything, which is
// what makes the wrapper emit a Trailers-Only response.
func newFailFastServer(handlerErr error) *grpc.Server {
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		var reqBytes []byte
		if err := stream.RecvMsg(&reqBytes); err != nil {
			return err
		}
		return handlerErr
	}
	return grpc.NewServer(grpc.UnknownServiceHandler(handler), grpc.ForceServerCodec(rawCodec{}))
}

// grpcFrame wraps payload in a gRPC length prefixed frame.
func grpcFrame(payload []byte) []byte {
	frame := make([]byte, 5, 5+len(payload))
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	return append(frame, payload...)
}

// splitFrames walks a gRPC-Web response body and separates the message frames from
// the trailer frame (the one whose most significant flag bit is set).
func splitFrames(t *testing.T, body []byte) (messages [][]byte, trailers []byte) {
	t.Helper()
	for len(body) > 0 {
		require.GreaterOrEqual(t, len(body), 5, "truncated frame header")
		flags := body[0]
		length := int(binary.BigEndian.Uint32(body[1:5]))
		require.GreaterOrEqual(t, len(body), 5+length, "truncated frame payload")
		payload := body[5 : 5+length]
		if flags&trailerFrameFlag != 0 {
			require.Nil(t, trailers, "more than one trailer frame")
			trailers = payload
		} else {
			messages = append(messages, payload)
		}
		body = body[5+length:]
	}
	return messages, trailers
}

// decodeGrpcWebText decodes a grpc-web-text body, which is a concatenation of
// independently padded base64 segments.
func decodeGrpcWebText(t *testing.T, body []byte) []byte {
	t.Helper()
	var decoded []byte
	for len(body) > 0 {
		require.Zero(t, len(body)%4, "base64 body is not a multiple of 4 bytes")
		segmentEnd := len(body)
		for i := 0; i+4 <= len(body); i += 4 {
			if bytes.ContainsRune(body[i:i+4], '=') {
				segmentEnd = i + 4
				break
			}
		}
		segment, err := base64.StdEncoding.DecodeString(string(body[:segmentEnd]))
		require.NoError(t, err)
		decoded = append(decoded, segment...)
		body = body[segmentEnd:]
	}
	return decoded
}

// doGrpcWebRequest performs a gRPC-Web call against handler and returns the response
// along with the decoded body.
func doGrpcWebRequest(t *testing.T, handler http.Handler, contentType string, payload []byte) (*http.Response, []byte) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	body := grpcFrame(payload)
	if contentType == grpcWebTextContentType {
		body = []byte(base64.StdEncoding.EncodeToString(body))
	}
	req, err := http.NewRequest(http.MethodPost, server.URL+testMethod, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("content-type", contentType)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if contentType == grpcWebTextContentType {
		raw = decodeGrpcWebText(t, raw)
	}
	return resp, raw
}

// TestEncodeTrailers_NoOptionalWhitespace is the regression test for the reported
// failure: a trailer block written as "grpc-status: 0" is read as " 0" by receivers
// that split on ':' without trimming, which rejects the call with
// "transport: malformed grpc-status" and drops the message the caller already got.
func TestEncodeTrailers_NoOptionalWhitespace(t *testing.T) {
	for _, contentType := range []string{grpcWebContentType + "+proto", grpcWebTextContentType} {
		t.Run(contentType, func(t *testing.T) {
			resp, body := doGrpcWebRequest(t, WrapServer(newEchoServer(nil)), contentType, []byte("ping"))
			require.Equal(t, http.StatusOK, resp.StatusCode)

			messages, trailers := splitFrames(t, body)
			require.Equal(t, [][]byte{[]byte("echo-ping")}, messages)
			require.NotNil(t, trailers, "response is missing its trailer frame")

			require.Contains(t, string(trailers), "grpc-status:0\r\n")
			require.NotContains(t, string(trailers), "grpc-status: ")
			// the padded trailer value is trimmed rather than passed on
			require.Contains(t, string(trailers), "x-padded-trailer:padded\r\n")

			// response headers are still copied through, with the gRPC-Web content type
			require.Equal(t, "12345", resp.Header.Get("X-Node-Height"))
			require.Equal(t, contentType, resp.Header.Get("Content-Type"))
			require.Contains(t, resp.Header.Get("Access-Control-Expose-Headers"), "grpc-status")
		})
	}
}

// TestEncodeTrailers_NonOKStatus checks a non-OK status is encoded the same way once the
// response body has already started.
func TestEncodeTrailers_NonOKStatus(t *testing.T) {
	handlerErr := status.Error(codes.NotFound, "no such block")
	resp, body := doGrpcWebRequest(t, WrapServer(newEchoServer(handlerErr)), grpcWebContentType+"+proto", []byte("ping"))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	messages, trailers := splitFrames(t, body)
	require.Equal(t, [][]byte{[]byte("echo-ping")}, messages)
	require.NotNil(t, trailers)
	require.Contains(t, string(trailers), "grpc-status:5\r\n")
	require.Contains(t, string(trailers), "grpc-message:no such block\r\n")
	require.NotContains(t, string(trailers), ": ")
}

// TestResponseWriter_TrailersOnly pins the shape of a call that fails before writing anything:
// the gRPC-Web spec allows the trailers to be sent as response headers with no body, and
// net/http serializes those, so they never pick up the stray whitespace.
func TestResponseWriter_TrailersOnly(t *testing.T) {
	handlerErr := status.Error(codes.NotFound, "no such block")
	resp, body := doGrpcWebRequest(t, WrapServer(newFailFastServer(handlerErr)), grpcWebContentType+"+proto", []byte("ping"))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, body)
	require.Equal(t, "5", resp.Header.Get("Grpc-Status"))
	require.Equal(t, "no such block", resp.Header.Get("Grpc-Message"))
	// the transport's trailer declaration must not leak into a response that has none
	require.Empty(t, resp.Header.Get("Trailer"))
}

// TestWrapServer_NativeGrpcPassthrough makes sure requests that are not gRPC-Web still reach the
// wrapped gRPC server untouched.
func TestWrapServer_NativeGrpcPassthrough(t *testing.T) {
	grpcServer := newEchoServer(nil)
	httpServer := &http.Server{Handler: h2c.NewHandler(WrapServer(grpcServer), &http2.Server{})}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go httpServer.Serve(lis)
	t.Cleanup(func() { httpServer.Close() })

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(rawCodec{})))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var respBytes []byte
	var trailer metadata.MD
	require.NoError(t, conn.Invoke(ctx, testMethod, []byte("ping"), &respBytes, grpc.Trailer(&trailer)))
	require.Equal(t, []byte("echo-ping"), respBytes)
	require.Equal(t, []string{"  padded  "}, trailer.Get("x-padded-trailer"))
}

// TestResponseContentType_NotReflectedFromRequest checks the response content type is assembled
// from this package's constants. The request header is attacker controlled, so echoing
// it back is a cross-site scripting sink.
func TestResponseContentType_NotReflectedFromRequest(t *testing.T) {
	hostile := grpcWebContentType + `+<script>alert(1)</script>`
	resp, _ := doGrpcWebRequest(t, WrapServer(newEchoServer(nil)), hostile, []byte("ping"))

	require.Equal(t, grpcWebContentType, resp.Header.Get("Content-Type"))
	require.NotContains(t, resp.Header.Get("Content-Type"), "script")
}

func TestResponseContentType_SubTypes(t *testing.T) {
	playbook := []struct {
		name      string
		base      string
		requested string
		expected  string
	}{
		{"bare", grpcWebContentType, grpcWebContentType, grpcWebContentType},
		{"proto", grpcWebContentType, grpcWebContentType + "+proto", grpcWebContentType + "+proto"},
		{"json", grpcWebContentType, grpcWebContentType + "+json", grpcWebContentType + "+json"},
		{"text", grpcWebTextContentType, grpcWebTextContentType, grpcWebTextContentType},
		{"text proto", grpcWebTextContentType, grpcWebTextContentType + "+proto", grpcWebTextContentType + "+proto"},
		// anything outside the known sub-types collapses to the base rather than echoing
		{"unknown sub-type", grpcWebContentType, grpcWebContentType + "+<script>", grpcWebContentType},
		{"empty", grpcWebContentType, "", grpcWebContentType},
	}
	for _, tt := range playbook {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, responseContentType(tt.base, tt.requested))
		})
	}
}

func TestIsGrpcWebRequest_MethodAndContentType(t *testing.T) {
	playbook := []struct {
		name        string
		method      string
		contentType string
		expected    bool
	}{
		{"grpc-web", http.MethodPost, grpcWebContentType, true},
		{"grpc-web proto", http.MethodPost, grpcWebContentType + "+proto", true},
		{"grpc-web text", http.MethodPost, grpcWebTextContentType, true},
		{"native grpc", http.MethodPost, grpcContentType, false},
		{"not a post", http.MethodGet, grpcWebContentType, false},
		// case sensitivity is deliberate: it mirrors the fallback's own predicate
		{"mixed case", http.MethodPost, "Application/Grpc-Web", false},
	}
	for _, tt := range playbook {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, testMethod, nil)
			req.Header.Set("content-type", tt.contentType)
			require.Equal(t, tt.expected, isGrpcWebRequest(req))
		})
	}
}

func TestSanitizeFieldValue_Whitespace(t *testing.T) {
	playbook := []struct {
		name     string
		value    string
		expected string
	}{
		{"leading space", " 0", "0"},
		{"trailing tab", "0\t", "0"},
		// CR and LF each become a space, matching net/http's own header serialization
		{"embedded crlf", "a\r\nb", "a  b"},
		{"all whitespace", "   ", ""},
		{"inner space kept", "keep me", "keep me"},
	}
	for _, tt := range playbook {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, sanitizeFieldValue(tt.value))
		})
	}
}
