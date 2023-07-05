package rewardserver

type DB interface {
	Save(key string, data []byte) error
	FindOne(key string) ([]byte, error)
	FindAll() (map[string][]byte, error)
	Delete(key string) error
	DeletePrefix(prefix string) error
	Close() error
}
