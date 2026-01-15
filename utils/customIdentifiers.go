package utils

import (
	"context"
)

type request_id_ctx_key struct{}

func WithRequestId(ctx context.Context, reqId string) context.Context {
	return context.WithValue(ctx, request_id_ctx_key{}, reqId)
}

func AppendRequestId(ctx context.Context, reqId string) context.Context {
	if ctx.Value(request_id_ctx_key{}) != nil || reqId == "" {
		return ctx
	}
	return context.WithValue(ctx, request_id_ctx_key{}, reqId)
}

func GetRequestId(ctx context.Context) (reqId string, found bool) {
	reqId, found = ctx.Value(request_id_ctx_key{}).(string)
	return reqId, found
}

type task_id_ctx_key struct{}

func WithTaskId(ctx context.Context, taskId string) context.Context {
	return context.WithValue(ctx, task_id_ctx_key{}, taskId)
}

func AppendTaskId(ctx context.Context, taskId string) context.Context {
	if ctx.Value(task_id_ctx_key{}) != nil || taskId == "" {
		return ctx
	}
	return context.WithValue(ctx, task_id_ctx_key{}, taskId)
}

func GetTaskId(ctx context.Context) (taskId string, found bool) {
	taskId, found = ctx.Value(task_id_ctx_key{}).(string)
	return taskId, found
}

type tx_id_ctx_key struct{}

func WithTxId(ctx context.Context, txId string) context.Context {
	return context.WithValue(ctx, tx_id_ctx_key{}, txId)
}

func AppendTxId(ctx context.Context, txId string) context.Context {
	if ctx.Value(tx_id_ctx_key{}) != nil || txId == "" {
		return ctx
	}
	return context.WithValue(ctx, tx_id_ctx_key{}, txId)
}

func GetTxId(ctx context.Context) (txId string, found bool) {
	txId, found = ctx.Value(tx_id_ctx_key{}).(string)
	return txId, found
}

// ExtractWantedHeadersFromCachedMap extracts specific headers from a pre-cached headers map
// and adds them to the Go context. This avoids repeated header lookups when headers
// are already cached via GetReqHeaders().
func ExtractWantedHeadersFromCachedMap(headers map[string][]string, ctx context.Context) context.Context {
	if reqId := getHeaderValue(headers, "X-Request-Id"); reqId != "" {
		ctx = WithRequestId(ctx, reqId)
	}

	if taskId := getHeaderValue(headers, "X-Task-Id"); taskId != "" {
		ctx = WithTaskId(ctx, taskId)
	}

	if txId := getHeaderValue(headers, "X-Tx-Id"); txId != "" {
		ctx = WithTxId(ctx, txId)
	}

	return ctx
}

// getHeaderValue extracts the first value for a header key from a cached headers map.
func getHeaderValue(headers map[string][]string, key string) string {
	if values, ok := headers[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}
