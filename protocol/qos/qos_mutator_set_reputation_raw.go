package qos

import (
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

// Mutator to set usage for a session
type QoSMutatorSetReputationRaw struct {
	QoSMutatorBase
	report *pairingtypes.QualityOfServiceReport
}

func (qoSMutatorSetReputationRaw *QoSMutatorSetReputationRaw) Mutate(report *QoSReport) {
	defer func() {
		qoSMutatorSetReputationRaw.doneChan <- struct{}{}
	}()
	report.lastReputationQoSReportRaw = qoSMutatorSetReputationRaw.report
}
