package qos

// Mutator for relay failure
type QoSMutatorDegradeAvailability struct {
	QoSMutatorBase
}

func (qoSMutatorDegradeAvailability *QoSMutatorDegradeAvailability) Mutate(report *QoSReport) {
	defer func() {
		qoSMutatorDegradeAvailability.doneChan <- struct{}{}
	}()
	report.answeredRelays--
}
