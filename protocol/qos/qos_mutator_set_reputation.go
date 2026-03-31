package qos

import (
	pairingtypes "github.com/lavanet/lava/v5/types/relay"
)

// Mutator to set usage for a session
type QoSMutatorSetReputation struct {
	*QoSMutatorBase
	report *pairingtypes.QualityOfServiceReport
}

func (qoSMutatorSetReputation *QoSMutatorSetReputation) Mutate(report *QoSReport) {
	report.lastReputationQoSReport = qoSMutatorSetReputation.report
}
