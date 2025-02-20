package lavasession

import (
	"sync"

	"github.com/lavanet/lava/v5/utils"
)

type StickySession struct {
	Provider string
	Epoch    uint64
}

type StickySessionStore struct {
	lock     sync.RWMutex
	sessions map[string]*StickySession
}

func NewStickySessionStore() *StickySessionStore {
	return &StickySessionStore{
		sessions: make(map[string]*StickySession),
	}
}

func (s *StickySessionStore) Get(id string) (*StickySession, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	session, exists := s.sessions[id]
	return session, exists
}

func (s *StickySessionStore) Set(id string, session *StickySession) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.sessions[id] = session
}

func (s *StickySessionStore) Delete(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.sessions, id)
}

func (s *StickySessionStore) DeleteOldSessions(epoch uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for id, session := range s.sessions {
		if session.Epoch < epoch {
			utils.LavaFormatTrace("deleting sticky session", utils.LogAttr("id", id))
			delete(s.sessions, id)
		}
	}
}
