package lavasession

import "time"

const (
	MaxConsecutiveConnectionAttemts                   = 3
	TimeoutForEstablishingAConnectionInMS             = 300 * time.Millisecond
	MaxSessionsAllowedPerProvider                     = 10 // Max number of sessions allowed per provider
	MaximumNumberOfFailiuresAllowedPerConsumerSession = 2
)
