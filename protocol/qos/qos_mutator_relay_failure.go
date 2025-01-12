package qos

// Mutator for relay failure
type QoSMutatorRelayFailure struct {
	QoSMutatorBase
}

func (qoSMutatorRelayFailure *QoSMutatorRelayFailure) Mutate(report *QoSReport) {
	report.lock.Lock()
	defer func() {
		report.lock.Unlock()
		qoSMutatorRelayFailure.doneChan <- struct{}{}
	}()
	report.totalRelays++
}
