// Package grpcwebcompat serves gRPC-Web requests with a canonically encoded
// trailer frame.
//
// gRPC-Web moves the response trailers (which carry grpc-status/grpc-message)
// out of the HTTP/2 TRAILERS frame and into the response body, as a final frame
// whose payload is an HTTP/1 style header block. github.com/improbable-eng/grpc-web
// builds that payload with http.Header.Write, which emits "key: value\r\n" -
// note the optional whitespace between the colon and the value. RFC 7230 allows
// that OWS and requires receivers to strip it, but a receiver that simply splits
// on ':' reads grpc-status as " 0" and fails with:
//
//	transport: malformed grpc-status: strconv.ParseInt: parsing " 0": invalid syntax
//
// which surfaces as a Trailers-Only response - an Internal error and no message
// at all for the caller.
//
// Envoy's grpc_web filter and the other reference implementations emit
// "key:value\r\n" with no OWS, which every parser accepts, so that is what this
// package writes. Values are also stripped of surrounding whitespace and of CR/LF
// (which would otherwise corrupt the framing of the trailer block itself).
//
// Everything that is not a gRPC-Web request - native gRPC, CORS pre-flights,
// gRPC-Websockets - is handed to the wrapped grpcweb server unchanged.
package grpcwebcompat

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md#protocol-differences-vs-grpc-over-http2
const (
	grpcContentType        = "application/grpc"
	grpcWebContentType     = "application/grpc-web"
	grpcWebTextContentType = "application/grpc-web-text"
)

// trailerFrameFlag marks the final frame of a gRPC-Web response as the trailer frame.
const trailerFrameFlag = 1 << 7

// WrapServer wraps a gRPC server for gRPC-Web the same way grpcweb.WrapServer does,
// except that gRPC-Web responses get a canonically encoded trailer frame.
func WrapServer(server *grpc.Server, options ...grpcweb.Option) http.Handler {
	return WrapHandler(server, grpcweb.WrapServer(server, options...))
}

// WrapHandler routes gRPC-Web requests to grpcHandler through this package's response
// writer and everything else to fallback, which is expected to be the equivalent
// grpcweb-wrapped handler.
func WrapHandler(grpcHandler, fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if IsGrpcWebRequest(req) {
			serveGrpcWeb(grpcHandler, resp, req)
			return
		}
		fallback.ServeHTTP(resp, req)
	})
}

// IsGrpcWebRequest reports whether req is a gRPC-Web request, using the same rule as
// grpcweb.WrappedGrpcServer.IsGrpcWebRequest so that the two stay in agreement about
// which requests this package takes over.
func IsGrpcWebRequest(req *http.Request) bool {
	return req.Method == http.MethodPost && strings.HasPrefix(req.Header.Get("content-type"), grpcWebContentType)
}

func serveGrpcWeb(grpcHandler http.Handler, resp http.ResponseWriter, req *http.Request) {
	nativeReq, isTextFormat := toNativeGrpcRequest(req)
	webResp := newResponseWriter(resp, isTextFormat)
	grpcHandler.ServeHTTP(webResp, nativeReq)
	webResp.finishRequest()
}

// toNativeGrpcRequest rewrites a gRPC-Web request into the plain gRPC request the
// grpc-go server expects.
func toNativeGrpcRequest(req *http.Request) (*http.Request, bool) {
	req.ProtoMajor = 2
	req.ProtoMinor = 0

	contentType := req.Header.Get("content-type")
	incomingContentType := grpcWebContentType
	isTextFormat := strings.HasPrefix(contentType, grpcWebTextContentType)
	if isTextFormat {
		// the body is base64 encoded; decode it while keeping the original Body closable
		req.Body = &readerCloser{reader: base64.NewDecoder(base64.StdEncoding, req.Body), closer: req.Body}
		incomingContentType = grpcWebTextContentType
	}
	req.Header.Set("content-type", strings.Replace(contentType, incomingContentType, grpcContentType, 1))

	// content-length describes the http1.1 payload, not the sum of the h2 DATA frame
	// lengths, and a mismatch makes the request malformed. Dropping it switches to
	// chunked encoding, which is the h2 default.
	req.Header.Del("content-length")

	return req, isTextFormat
}

// responseWriter turns the grpc-go server's plain gRPC response into a gRPC-Web one:
// the headers are copied through, and the trailers are appended to the body as a
// trailer frame instead of being sent as HTTP trailers.
type responseWriter struct {
	wroteHeaders bool
	wroteBody    bool
	headers      http.Header
	// wrapped must be flushed before returning so the base64 encoder, when used, emits
	// its final segment.
	wrapped http.ResponseWriter
	// contentType replaces the standard "application/grpc" content type.
	contentType string
}

func newResponseWriter(resp http.ResponseWriter, isTextFormat bool) *responseWriter {
	w := &responseWriter{
		headers:     make(http.Header),
		wrapped:     resp,
		contentType: grpcWebContentType,
	}
	if isTextFormat {
		w.wrapped = newBase64ResponseWriter(w.wrapped)
		w.contentType = grpcWebTextContentType
	}
	return w
}

func (w *responseWriter) Header() http.Header {
	return w.headers
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeaders {
		w.prepareHeaders()
	}
	w.wroteBody, w.wroteHeaders = true, true
	return w.wrapped.Write(b)
}

func (w *responseWriter) WriteHeader(code int) {
	w.prepareHeaders()
	w.wrapped.WriteHeader(code)
	w.wroteHeaders = true
}

func (w *responseWriter) Flush() {
	// WriteHeader followed by a Flush would otherwise force a 200 response even when
	// there is no payload at all.
	if w.wroteHeaders || w.wroteBody {
		flushWriter(w.wrapped)
	}
}

// prepareHeaders copies the headers the gRPC handler set onto the real response
// writer, translating the content type and exposing the response headers to browsers.
func (w *responseWriter) prepareHeaders() {
	flushed := w.wrapped.Header()
	for key, values := range w.headers {
		if strings.EqualFold(key, "trailer") {
			// the trailer declaration is meaningless once trailers move into the body
			continue
		}
		key = strings.Replace(key, http.TrailerPrefix, "", 1)
		if strings.EqualFold(key, "content-type") {
			replaced := make([]string, len(values))
			for i, value := range values {
				replaced[i] = strings.Replace(value, grpcContentType, w.contentType, 1)
			}
			values = replaced
		}
		flushed[http.CanonicalHeaderKey(key)] = values
	}

	exposed := make([]string, 0, len(flushed)+2)
	for key := range flushed {
		exposed = append(exposed, key)
	}
	sort.Strings(exposed)
	// grpc-status and grpc-message travel in the body, but browsers still expect them
	// to be listed here, matching grpcweb's behaviour.
	exposed = append(exposed, "grpc-status", "grpc-message")
	flushed.Set(http.CanonicalHeaderKey("access-control-expose-headers"), strings.Join(exposed, ", "))
}

func (w *responseWriter) finishRequest() {
	if w.wroteHeaders || w.wroteBody {
		w.writeTrailerFrame()
		return
	}
	w.WriteHeader(http.StatusOK)
	flushWriter(w.wrapped)
}

// writeTrailerFrame appends the trailer frame that terminates a gRPC-Web response.
func (w *responseWriter) writeTrailerFrame() {
	payload := encodeTrailers(w.extractTrailers())
	frame := make([]byte, 5, 5+len(payload))
	frame[0] = trailerFrameFlag
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	frame = append(frame, payload...)
	w.wrapped.Write(frame)
	flushWriter(w.wrapped)
}

// extractTrailers returns the headers the gRPC handler set after the response headers
// had already been flushed - grpc-status, grpc-message and anything the handler sent
// as an undeclared trailer.
func (w *responseWriter) extractTrailers() http.Header {
	flushed := w.wrapped.Header()
	trailers := make(http.Header)
	for key, values := range w.headers {
		if strings.EqualFold(key, "trailer") {
			continue
		}
		if strings.Contains(key, http.TrailerPrefix) {
			key = strings.Replace(key, http.TrailerPrefix, "", 1)
		} else if _, alreadySent := flushed[http.CanonicalHeaderKey(key)]; alreadySent {
			continue
		}
		// the gRPC-Web spec requires lower-case trailer names, and names that differ only
		// in case have to merge rather than overwrite one another
		key = strings.ToLower(key)
		trailers[key] = append(trailers[key], values...)
	}
	return trailers
}

// encodeTrailers writes the trailer block as "key:value\r\n" pairs. The colon is not
// followed by whitespace: RFC 7230 permits it, but receivers that split on ':' without
// trimming then read grpc-status as " 0" and reject the response.
func encodeTrailers(trailers http.Header) []byte {
	keys := make([]string, 0, len(trailers))
	for key := range trailers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	buf := new(bytes.Buffer)
	for _, key := range keys {
		for _, value := range trailers[key] {
			buf.WriteString(key)
			buf.WriteByte(':')
			buf.WriteString(sanitizeFieldValue(value))
			buf.WriteString("\r\n")
		}
	}
	return buf.Bytes()
}

// sanitizeFieldValue strips the surrounding optional whitespace a receiver would have
// to trim anyway, and replaces CR/LF, which would otherwise split one trailer into
// several or truncate the block.
func sanitizeFieldValue(value string) string {
	if strings.ContainsAny(value, "\r\n") {
		value = strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
	}
	return strings.TrimSpace(value)
}

// base64ResponseWriter writes base64 encoded payloads for the grpc-web-text format.
// Flush must be called to make the encoder emit its final segment.
type base64ResponseWriter struct {
	wrapped http.ResponseWriter
	encoder io.WriteCloser
}

func newBase64ResponseWriter(wrapped http.ResponseWriter) http.ResponseWriter {
	w := &base64ResponseWriter{wrapped: wrapped}
	w.newEncoder()
	return w
}

func (w *base64ResponseWriter) newEncoder() {
	w.encoder = base64.NewEncoder(base64.StdEncoding, w.wrapped)
}

func (w *base64ResponseWriter) Header() http.Header {
	return w.wrapped.Header()
}

func (w *base64ResponseWriter) Write(b []byte) (int, error) {
	return w.encoder.Write(b)
}

func (w *base64ResponseWriter) WriteHeader(code int) {
	w.wrapped.WriteHeader(code)
}

func (w *base64ResponseWriter) Flush() {
	// closing the encoder flushes it; gRPC-Web allows multiple padded base64 parts
	if err := w.encoder.Close(); err != nil {
		// Flush cannot report an error
		grpclog.Errorf("ignoring error flushing base64 encoder: %v", err)
	}
	w.newEncoder()
	flushWriter(w.wrapped)
}

func flushWriter(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// readerCloser pairs an io.Reader with the io.Closer of the reader it decorates.
type readerCloser struct {
	reader io.Reader
	closer io.Closer
}

func (r *readerCloser) Read(dest []byte) (int, error) {
	return r.reader.Read(dest)
}

func (r *readerCloser) Close() error {
	return r.closer.Close()
}
