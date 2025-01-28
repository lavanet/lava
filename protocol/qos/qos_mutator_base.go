package qos

import "sync/atomic"

// Base interface for all mutators
type Mutator interface {
	Mutate(report *QoSReport)
	GetEpochAndSessionId() (epoch uint64, sessionId int64)
}

type QoSMutatorBase struct {
	epoch     uint64
	sessionId int64
}

func (qoSMutatorBase *QoSMutatorBase) GetEpochAndSessionId() (epoch uint64, sessionId int64) {
	return atomic.LoadUint64(&qoSMutatorBase.epoch), atomic.LoadInt64(&qoSMutatorBase.sessionId)
}
