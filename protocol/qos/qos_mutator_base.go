package qos

// Base interface for all mutators
type Mutator interface {
	Mutate(report *QoSReport)
	GetEpochAndSessionId() (epoch uint64, sessionId int64)
}

type QoSMutatorBase struct {
	epoch     uint64
	sessionId int64
	doneChan  chan<- struct{}
}

func (qoSMutatorBase *QoSMutatorBase) GetEpochAndSessionId() (epoch uint64, sessionId int64) {
	return qoSMutatorBase.epoch, qoSMutatorBase.sessionId
}
