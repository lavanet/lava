package rewardserver

import (
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/lavanet/lava/utils"
)

type BadgerDB struct {
	db *badger.DB
}

var _ DB = (*BadgerDB)(nil)

func (mdb *BadgerDB) Save(key string, data []byte, ttl time.Duration) error {
	err := mdb.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), data).WithTTL(ttl)
		return txn.SetEntry(e)
	})

	return err
}

func (mdb *BadgerDB) FindOne(key string) (one []byte, err error) {
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

func (mdb *BadgerDB) FindAll(providerAddress, specId string) (map[string][]byte, error) {
	result := make(map[string][]byte)

	err := mdb.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		opts.Prefix = []byte(providerAddress + specId)

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

func (mdb *BadgerDB) Delete(key string) error {
	err := mdb.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})

	return err
}

func (mdb *BadgerDB) DeletePrefix(prefix string) error {
	err := mdb.db.DropPrefix([]byte(prefix))
	if err != nil {
		return err
	}

	return err
}

func (mdb *BadgerDB) Close() error {
	return mdb.db.Close()
}

func NewMemoryDB() *BadgerDB {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}

	return &BadgerDB{
		db: db,
	}
}

func NewLocalDB(path string) *BadgerDB {
	Options := badger.DefaultOptions(path)
	Options.Logger = utils.LoggerWrapper{LoggerName: "[Badger DB]: "} // replace the logger with lava logger
	db, err := badger.Open(Options)
	if err != nil {
		panic(err)
	}

	return &BadgerDB{
		db: db,
	}
}
