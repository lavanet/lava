package qos

import (
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

// Mutator to set usage for a session
type QoSMutatorSetReputation struct {
	*QoSMutatorBase
	report *pairingtypes.QualityOfServiceReport
}

func (qoSMutatorSetReputation *QoSMutatorSetReputation) Mutate(report *QoSReport) {
	report.lastReputationQoSReport = qoSMutatorSetReputation.report
}
