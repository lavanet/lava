package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeClientConn struct {
	invokeErr error
	method    string
}

func (f *fakeClientConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	f.method = method
	return f.invokeErr
}

func (f *fakeClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	f.method = method
	return nil, f.invokeErr
}

func setupTestTracer(t *testing.T) *tracetest.SpanRecorder {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	prevTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		_ = tp.Shutdown(context.Background())
	})
	return sr
}

func TestWrapClientConn_InvokeSuccessRecordsSpan(t *testing.T) {
	sr := setupTestTracer(t)
	inner := &fakeClientConn{}
	wrapped := WrapClientConn(inner)

	err := wrapped.Invoke(context.Background(), "/lavanet.lava.pairing.Query/GetPairing", nil, nil)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanLavaChainQuery, spans[0].Name())

	attrs := spans[0].Attributes()
	method, ok := findStringAttr(attrs, attrLavaChainMethod)
	require.True(t, ok)
	require.Equal(t, "/lavanet.lava.pairing.Query/GetPairing", method)

	code, ok := findStringAttr(attrs, attrLavaChainCode)
	require.True(t, ok)
	require.Equal(t, codes.OK.String(), code)
}

func TestWrapClientConn_InvokeErrorRecordsCode(t *testing.T) {
	sr := setupTestTracer(t)
	inner := &fakeClientConn{invokeErr: status.Error(codes.Unavailable, "node unreachable")}
	wrapped := WrapClientConn(inner)

	err := wrapped.Invoke(context.Background(), "/foo.Bar/Baz", nil, nil)
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)

	code, ok := findStringAttr(spans[0].Attributes(), attrLavaChainCode)
	require.True(t, ok)
	require.Equal(t, codes.Unavailable.String(), code)
}

func TestStartLavaChainTx_RecordsMsgType(t *testing.T) {
	sr := setupTestTracer(t)

	ctx, span := StartLavaChainTx(context.Background(), "/lavanet.lava.pairing.MsgRelayPayment")
	require.NotNil(t, ctx)
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanLavaChainSubmitTx, spans[0].Name())

	msg, ok := findStringAttr(spans[0].Attributes(), attrLavaChainMsgType)
	require.True(t, ok)
	require.Equal(t, "/lavanet.lava.pairing.MsgRelayPayment", msg)
}

func TestStartLavaChainTx_RecordError(t *testing.T) {
	sr := setupTestTracer(t)
	_, span := StartLavaChainTx(context.Background(), "/foo.Bar/MsgX")
	RecordError(span, errors.New("simulate failed"))
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.NotEqual(t, "", spans[0].Status().Description)
}

// findStringAttr returns the string value of the first attribute matching key.
func findStringAttr(attrs []attribute.KeyValue, key string) (string, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value.AsString(), true
		}
	}
	return "", false
}
