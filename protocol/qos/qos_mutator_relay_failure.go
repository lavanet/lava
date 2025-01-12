package qos

// Mutator for relay failure
type QoSMutatorRelayFailure struct {
	QoSMutatorBase
}

func (qoSMutatorRelayFailure *QoSMutatorRelayFailure) Mutate(report *QoSReport) {
	defer func() {
		qoSMutatorRelayFailure.doneChan <- struct{}{}
	}()
	report.totalRelays++
}
