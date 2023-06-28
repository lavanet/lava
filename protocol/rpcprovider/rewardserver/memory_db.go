package rewardserver

import (
	"context"

	"github.com/dgraph-io/badger/v4"
)

type MemoryDB struct {
	db *badger.DB
}

var _ DB = (*MemoryDB)(nil)

func (mdb *MemoryDB) Save(ctx context.Context, key string, data []byte) error {
	err := mdb.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})

	return err
}

func (mdb *MemoryDB) FindOne(ctx context.Context, key string) (one []byte, err error) {
	err = mdb.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		one, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return
}

func (mdb *MemoryDB) FindAll(ctx context.Context) (map[string][]byte, error) {
	result := make(map[string][]byte)

	err := mdb.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			result[key] = value
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (mdb *MemoryDB) Delete(ctx context.Context, key string) error {
	err := mdb.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})

	return err
}

func (mdb *MemoryDB) DeletePrefix(ctx context.Context, prefix string) error {
	err := mdb.db.DropPrefix([]byte(prefix))
	if err != nil {
		return err
	}

	return err
}

func NewMemoryDB() *MemoryDB {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		// TODO: what do we do if the database cannot open?
		panic(err)
	}

	return &MemoryDB{
		db: db,
	}
}
