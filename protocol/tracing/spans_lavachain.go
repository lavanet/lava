package tracing

import (
	"context"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// tracerName is the OTel tracer name (instrumentation library identifier)
// used by all helpers in this package. Distinct from the smartrouter scope
// to make span source readily identifiable in the trace UI.
const tracerName = "github.com/lavanet/lava/v5/protocol/tracing"

// WrapClientConn wraps a gogoproto-style ClientConn so every Invoke and
// NewStream call emits a `lavachain.Query` span. Used at the seam between
// Lava code and the Cosmos SDK client.Context — Cosmos SDK adapts a
// gRPC-shaped interface to a Tendermint RPC ABCI transport, so an
// otelgrpc.UnaryClientInterceptor (which only fires on real grpc.Dial calls)
// does not apply here.
func WrapClientConn(inner grpc1.ClientConn) grpc1.ClientConn {
	return &tracedClientConn{inner: inner}
}

type tracedClientConn struct {
	inner grpc1.ClientConn
}

func (t *tracedClientConn) Invoke(
	ctx context.Context,
	method string,
	args, reply interface{},
	opts ...grpc.CallOption,
) error {
	ctx, span := otel.Tracer(tracerName).Start(ctx, SpanLavaChainQuery,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String(attrLavaChainMethod, method)),
	)
	defer span.End()

	err := t.inner.Invoke(ctx, method, args, reply, opts...)
	span.SetAttributes(attribute.String(attrLavaChainCode, status.Code(err).String()))
	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

func (t *tracedClientConn) NewStream(
	ctx context.Context,
	desc *grpc.StreamDesc,
	method string,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, SpanLavaChainQuery,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String(attrLavaChainMethod, method)),
	)
	stream, err := t.inner.NewStream(ctx, desc, method, opts...)
	if err != nil {
		span.SetAttributes(attribute.String(attrLavaChainCode, status.Code(err).String()))
		span.SetStatus(otelcodes.Error, err.Error())
		span.RecordError(err)
		span.End()
		return nil, err
	}
	// Stream lifetime is owned by the caller. This goroutine parks on
	// stream.Context().Done() and ends the span when the stream closes;
	// its lifetime is bounded by the stream's lifetime, NOT a leak. For
	// long-lived state-tracker subscription streams the goroutine stays
	// parked until binary shutdown — that's intentional and correct.
	go func() {
		<-stream.Context().Done()
		span.SetAttributes(attribute.String(attrLavaChainCode, codes.OK.String()))
		span.End()
	}()
	return stream, nil
}

// StartLavaChainTx starts a span around a Cosmos SDK transaction broadcast
// to lavad. Caller sets broadcast/commit timings and tx hash via
// RecordLavaChainTxResult as those values become available, then
// `defer span.End()`.
func StartLavaChainTx(ctx context.Context, msgType string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, SpanLavaChainSubmitTx,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String(attrLavaChainMsgType, msgType)),
	)
}

// RecordLavaChainTxResult attaches the broadcast/commit timing and result
// data to a `lavachain.SubmitTx` span. Safe to call on a non-recording span.
func RecordLavaChainTxResult(span trace.Span, broadcastMs, commitMs int64, height int64, txHash string) {
	span.SetAttributes(
		attribute.Int64(attrLavaChainBroadcastMs, broadcastMs),
		attribute.Int64(attrLavaChainCommitMs, commitMs),
		attribute.Int64(attrLavaChainHeight, height),
		attribute.String(attrLavaChainTxHash, txHash),
	)
}
