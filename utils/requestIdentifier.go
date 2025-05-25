package utils

import (
	"context"
)

type request_id_ctx_key struct{}

func WithRequestId(ctx context.Context, reqId string) context.Context {
	return context.WithValue(ctx, request_id_ctx_key{}, reqId)
}

func GetRequestId(ctx context.Context) (reqId string, found bool) {
	reqId, found = ctx.Value(request_id_ctx_key{}).(string)
	return
}
