package e2e

// This is a map of error strings with the description why they are allowed
// Detect extra text to be sure it is the allowed error
var allowedErrors = map[string]string{
	"getSupportedApi":        "This error is allowed because the Tendermint URI tests have a test that checks if the error is caught.",
	"No pairings available.": "This error is allowed because when the network is just booted up and pairings are not yet done this would happen. If after a few seconds the pairings are still not available the e2e would fail because the initial check if the provider is responsive would time out.",
	`error connecting to provider error="context deadline exceeded"`: "This error is allowed because it is caused by the initial bootup, continuous failure would be caught by the e2e so we can allowed this error.",
	"purging provider after all endpoints are disabled provider":     "This error is allowed because it is caused by the initial bootup, continuous failure would be caught by the e2e so we can allowed this error.",
	"Provider Side Failed Sending Message, Reason: Unavailable":      "This error is allowed because it is caused by the lavad restart to turn on emergency mode",
	"Maximum cu exceeded PrepareSessionForUsage":                     "This error is allowed because it is caused by switching between providers, continuous failure would be caught by the e2e so we can allowed this error.",
	"Failed To Connect to cache at address":                          "This error is allowed because it is caused by cache being connected only during the test and not during the bootup",
}

var allowedErrorsDuringEmergencyMode = map[string]string{
	"connection refused":           "Connection to tendermint port sometimes can happen as we shut down the node and we try to fetch info during emergency mode",
	"Connection refused":           "Connection to tendermint port sometimes can happen as we shut down the node and we try to fetch info during emergency mode",
	"connection reset by peer":     "Connection to tendermint port sometimes can happen as we shut down the node and we try to fetch info during emergency mode",
	"Failed Querying EpochDetails": "Connection to tendermint port sometimes can happen as we shut down the node and we try to fetch info during emergency mode",
	"http://[IP_ADDRESS]:26657":    "This error is allowed because it can happen when EOF error happens when we shut down the node in emergency mode",
}

var allowedErrorsPaymentE2E = map[string]string{
	"conflict with is already open for this client": "This error is allowed because it's unrelated to payment E2E",
	"could not get pairing":                         "This error is allowed because the test passes and then there's a random instance of this error. It's ok to allow it because if there was no pairing in a critical step of the test, the test would fail since it checks payments",
	"block is too new":                              "This error is allowed because it's unrelated to payment E2E",
}
