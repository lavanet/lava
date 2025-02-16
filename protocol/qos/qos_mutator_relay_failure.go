package qos

// Mutator for relay failure
type QoSMutatorRelayFailure struct {
	*QoSMutatorBase
}

func (qoSMutatorRelayFailure *QoSMutatorRelayFailure) Mutate(report *QoSReport) {
	report.totalRelays++
}
