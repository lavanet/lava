package rewardserver

import "context"

type DB interface {
	Save(ctx context.Context, key string, data []byte) error
	FindOne(ctx context.Context, key string) ([]byte, error)
	FindAll(ctx context.Context) (map[string][]byte, error)
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}
