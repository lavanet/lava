package rewardserver

import (
	"sync"
)

type ThreadSafeDB struct {
	innerDB DB
	lock    sync.RWMutex
}

var _ DB = (*ThreadSafeDB)(nil)

func (threadSafeDB *ThreadSafeDB) Key() string {
	threadSafeDB.lock.RLock()
	defer threadSafeDB.lock.RUnlock()
	return threadSafeDB.innerDB.Key()
}

func (threadSafeDB *ThreadSafeDB) Save(dbEntry *DBEntry) error {
	threadSafeDB.lock.Lock()
	defer threadSafeDB.lock.Unlock()
	return threadSafeDB.innerDB.Save(dbEntry)
}

func (threadSafeDB *ThreadSafeDB) BatchSave(dbEntries []*DBEntry) error {
	threadSafeDB.lock.Lock()
	defer threadSafeDB.lock.Unlock()
	return threadSafeDB.innerDB.BatchSave(dbEntries)
}

func (threadSafeDB *ThreadSafeDB) FindOne(key string) (one []byte, err error) {
	threadSafeDB.lock.RLock()
	defer threadSafeDB.lock.RUnlock()
	return threadSafeDB.innerDB.FindOne(key)
}

func (threadSafeDB *ThreadSafeDB) FindAll() (map[string][]byte, error) {
	threadSafeDB.lock.RLock()
	defer threadSafeDB.lock.RUnlock()
	return threadSafeDB.innerDB.FindAll()
}

func (threadSafeDB *ThreadSafeDB) Delete(key string) error {
	threadSafeDB.lock.Lock()
	defer threadSafeDB.lock.Unlock()
	return threadSafeDB.innerDB.Delete(key)
}

func (threadSafeDB *ThreadSafeDB) DeletePrefix(prefix string) error {
	threadSafeDB.lock.Lock()
	defer threadSafeDB.lock.Unlock()
	return threadSafeDB.innerDB.DeletePrefix(prefix)
}

func (threadSafeDB *ThreadSafeDB) Close() error {
	threadSafeDB.lock.Lock()
	defer threadSafeDB.lock.Unlock()
	return threadSafeDB.innerDB.Close()
}
