package rewardserver

import (
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type BadgerDB struct {
	providerAddr string
	specId       string
	shardString  string
	rewards      map[string]*entryWithTtl
	db           *badger.DB
	lock         sync.RWMutex
}

var _ DB = (*BadgerDB)(nil)

const persistThreshold = 100

func (mdb *BadgerDB) Key() string {
	return mdb.specId
}

func (mdb *BadgerDB) Save(key string, data []byte, ttl time.Duration) error {
	mdb.lock.Lock()
	defer mdb.lock.Unlock()

	mdb.rewards[key] = newEntry(data, ttl)

	if len(mdb.rewards) == persistThreshold {
		// err := mdb.saveAll(mdb.rewards)
		// if err != nil {
		// 	return err
		// }

		mdb.rewards = make(map[string]*entryWithTtl, persistThreshold)
	}

	return nil
}

func (mdb *BadgerDB) saveAll(rewards map[string]*entryWithTtl) error {
	err := mdb.db.Update(func(txn *badger.Txn) error {
		for key, data := range mdb.rewards {
			e := badger.NewEntry([]byte(key), data.data).WithTTL(data.remainingTtl())
			err := txn.SetEntry(e)
			if err != nil {
				return err
			}
		}

		return txn.Commit()
	})

	return err
}

func (mdb *BadgerDB) FindOne(key string) (one []byte, err error) {
	mdb.lock.RLock()
	defer mdb.lock.RUnlock()

	entry, found := mdb.rewards[key]
	if found && !entry.isExpired() {
		return entry.data, nil
	}

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

	mdb.lock.RLock()
	for key, value := range mdb.rewards {
		if value.isExpired() {
			continue
		}

		result[key] = value.data
	}
	mdb.lock.RUnlock()

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
	mdb.lock.Lock()
	defer mdb.lock.Unlock()

	delete(mdb.rewards, key)

	err := mdb.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})

	return err
}

func (mdb *BadgerDB) DeletePrefix(prefix string) error {
	mdb.lock.Lock()
	defer mdb.lock.Unlock()

	for key := range mdb.rewards {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		delete(mdb.rewards, key)
	}

	err := mdb.db.DropPrefix([]byte(prefix))
	if err != nil {
		return err
	}

	return err
}

func (mdb *BadgerDB) Close() error {
	if len(mdb.rewards) > 0 {
		err := mdb.saveAll(mdb.rewards)
		if err != nil {
			return err
		}
	}

	return mdb.db.Close()
}

func NewMemoryDB(specId string) *BadgerDB {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}

	return &BadgerDB{
		specId:  specId,
		rewards: make(map[string]*entryWithTtl, persistThreshold),
		db:      db,
	}
}

func NewLocalDB(storagePath, providerAddr string, specId string, shard uint) *BadgerDB {
	shardString := strconv.FormatUint(uint64(shard), 10)
	path := filepath.Join(storagePath, providerAddr, specId, shardString)
	Options := badger.DefaultOptions(path)
	// Options.Logger = utils.LoggerWrapper{LoggerName: "[Badger DB]: "} // replace the logger with lava logger
	Options.Logger = nil
	db, err := badger.Open(Options)
	if err != nil {
		panic(err)
	}

	return &BadgerDB{
		providerAddr: providerAddr,
		specId:       specId,
		shardString:  shardString,
		rewards:      make(map[string]*entryWithTtl, persistThreshold),
		db:           db,
	}
}

type entryWithTtl struct {
	expiresAt time.Time
	data      []byte
}

func newEntry(data []byte, ttl time.Duration) *entryWithTtl {
	return &entryWithTtl{
		expiresAt: time.Now().Add(ttl),
		data:      data,
	}
}

func (e *entryWithTtl) isExpired() bool {
	return e.expiresAt.Before(time.Now())
}

func (e *entryWithTtl) remainingTtl() time.Duration {
	return time.Until(e.expiresAt)
}
