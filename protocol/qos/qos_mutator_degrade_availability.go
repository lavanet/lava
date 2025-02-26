package qos

import "github.com/lavanet/lava/v5/utils"

// Mutator for relay failure
type QoSMutatorDegradeAvailability struct {
	*QoSMutatorBase
}

func (qoSMutatorDegradeAvailability *QoSMutatorDegradeAvailability) Mutate(report *QoSReport) {
	if report.answeredRelays > 0 {
		report.answeredRelays--
	} else {
		utils.LavaFormatError("Tried to degrade availability more than answered relays", nil, utils.LogAttr("report", report))
	}
}
