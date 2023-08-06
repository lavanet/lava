package rewardserver

import "time"

type DB interface {
	Key() string
	Save(key string, data []byte, ttl time.Duration) error
	FindOne(key string) ([]byte, error)
	FindAll() (map[string][]byte, error)
	Delete(key string) error
	DeletePrefix(prefix string) error
	Close() error
}
