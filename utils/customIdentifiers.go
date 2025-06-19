package utils

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

func UpdateAllCustomContextFields(originCtx context.Context, ctx context.Context) context.Context {
	var resCtx = ctx
	if taskId, found := GetTaskId(originCtx); found {
		resCtx = AppendTaskId(resCtx, taskId)
	}
	if reqId, found := GetRequestId(originCtx); found {
		resCtx = AppendRequestId(resCtx, reqId)
	}
	if txId, found := GetTxId(originCtx); found {
		resCtx = AppendTxId(resCtx, txId)
	}

	return resCtx
}

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
	return
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
	return
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
	return
}

func ExtractWantedHeadersAndUpdateContext(fiberCtx *fiber.Ctx, ctx context.Context) context.Context {
	reqId := fiberCtx.Get("x-request-id", "")
	taskId := fiberCtx.Get("x-task-id", "")
	txId := fiberCtx.Get("x-tx-id", "")

	if reqId != "" {
		ctx = WithRequestId(ctx, reqId)
	}

	if taskId != "" {
		ctx = WithTaskId(ctx, taskId)
	}

	if txId != "" {
		ctx = WithTxId(ctx, txId)
	}

	return ctx
}
