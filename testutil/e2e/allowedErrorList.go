package e2e

// This is a map of error strings with the description why they are allowed
// Detect extra text to be sure it is the allowed error
var allowedErrors = map[string]string{
	"getSupportedApi":        "This error is allowed because the Tendermint URI tests have a test that checks if the error is caught.",
	"No pairings available.": "This error is allowed because when the network is just booted up and pairings are not yet done this would happen. If after a few seconds the pairings are still not available the e2e would fail because the initial check if the provider is responsive would time out.",
	`error connecting to provider error="context deadline exceeded"`:                        "This error is allowed because it is caused by the initial bootup, continuous failure would be caught by the e2e so we can allowed this error.",
	"purging provider after all endpoints are disabled provider":                            "This error is allowed because it is caused by the initial bootup, continuous failure would be caught by the e2e so we can allowed this error.",
	"query hash mismatch on data reliability message":                                       "This error is allowed temporarily because of data reliability",
	"invalid pairing with consumer":                                                         "This error is allowed temporarily because of data reliability",
	"Could not get reply to reliability relay":                                              "This error is allowed temporarily because of data reliability",
	"VerifyReliabilityAddressSigning invalid":                                               "This error is allowed temporarily because of data reliability",
	"invalid self pairing with consumer consumer":                                           "This error is allowed temporarily because of data reliability",
	"Simulation: conflict with is already open for this client and providers in this epoch": "This error is allowed temporarily because of data reliability",
}
