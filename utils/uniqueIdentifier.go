package utils

import (
	"context"
	"math/rand"
)

type unique_identifier_ctx_key struct{}

func GenerateUniqueIdentifier() uint64 {
	return rand.Uint64()
}

func WithUniqueIdentifier(ctx context.Context, guid uint64) context.Context {
	return context.WithValue(ctx, unique_identifier_ctx_key{}, guid)
}

func AppendUniqueIdentifier(ctx context.Context, guid uint64) context.Context {
	if ctx.Value(unique_identifier_ctx_key{}) != nil || guid == 0 {
		// do not add a guid if it exists
		return ctx
	}
	return context.WithValue(ctx, unique_identifier_ctx_key{}, guid)
}

func GetUniqueIdentifier(ctx context.Context) (guid uint64, found bool) {
	guid, found = ctx.Value(unique_identifier_ctx_key{}).(uint64)
	if !found {
		return 0, false
	}
	return guid, found
}
