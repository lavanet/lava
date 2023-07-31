package rewardserver

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/lavanet/lava/utils"
)

type BadgerDB struct {
	providerAddr string
	specId       string
	shardString  string
	db           *badger.DB
}

var _ DB = (*BadgerDB)(nil)

func (mdb *BadgerDB) Key() string {
	return mdb.specId
}

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

func (mdb *BadgerDB) FindAll() (map[string][]byte, error) {
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

func NewMemoryDB(specId string) *BadgerDB {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}

	return &BadgerDB{
		specId: specId,
		db:     db,
	}
}

func NewLocalDB(storagePath, providerAddr string, specId string, shard uint) *BadgerDB {
	shardString := strconv.FormatUint(uint64(shard), 10)
	path := filepath.Join(storagePath, providerAddr, specId, shardString)
	Options := badger.DefaultOptions(path)
	Options.Logger = utils.LoggerWrapper{LoggerName: "[Badger DB]: "} // replace the logger with lava logger
	db, err := badger.Open(Options)
	if err != nil {
		panic(err)
	}

	return &BadgerDB{
		providerAddr: providerAddr,
		specId:       specId,
		shardString:  shardString,
		db:           db,
	}
}
