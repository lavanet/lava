package e2e

// This is a map of error strings with the description why they are allowed
// Detect extra text to be sure it is the allowed error
var allowedErrors = map[string]string{
	"getSupportedApi":        "This error is allowed because the Tendermint URI tests have a test that checks if the error is caught.",
	"No pairings available.": "This error is allowed because when the network is just booted up and pairings are not yet done this would happen. If after a few seconds the pairings are still not available the e2e would fail because the initial check if the provider is responsive would time out.",
	`error connecting to provider error="context deadline exceeded"`:                   "This error is allowed because it is caused by the initial bootup, continuous failure would be caught by the e2e so we can allowed this error.",
	"purging provider after all endpoints are disabled provider":                       "This error is allowed because it is caused by the initial bootup, continuous failure would be caught by the e2e so we can allowed this error.",
	"Provider Side Failed Sending Message, Reason: Unavailable":                        "This error is allowed because it is caused by the lavad restart to turn on emergency mode",
	"Maximum cu exceeded PrepareSessionForUsage":                                       "This error is allowed because it is caused by switching between providers, continuous failure would be caught by the e2e so we can allowed this error.",
	"Failed To Connect to cache at address":                                            "This error is allowed because it is caused by cache being connected only during the test and not during the bootup",
	"Not Implemented":                                                                  "This error is allowed because the /lavanet/lava/pairing/clients API endpoint returns 501 Not Implemented from providers during tests",
	"unsupported method":                                                               "This error is allowed because providers now return unsupported method errors for unimplemented APIs instead of retrying",
	"unsupported method 'Default-/lavanet/lava/pairing/clients/LAV1': Not Implemented": "This error is allowed because the /lavanet/lava/pairing/clients API endpoint is not implemented in test providers",
	"unsupported method 'Default-/lavanet/lava/conflict/params': test":                 "This error is allowed because the /lavanet/lava/conflict/params API endpoint is not implemented in test providers",
	"failed processing responses from providers":                                       "This error is allowed because it can occur when providers return unsupported method errors",
	"failed relay, insufficient results":                                               "This error is allowed because it can occur when providers return unsupported method errors",
	"Error_GUID":                                                                       "This error is allowed because it's part of error responses that contain unsupported method errors",
	"{\"Error_GUID\":":                                                                 "This error is allowed because it's a JSON error response containing unsupported method errors",
	"endpoint:LAV1rest":                                                                "This error is allowed because it's part of LAV1 REST endpoint error responses",
	"{\"error\":":                                                                      "This error is allowed because it's a JSON error response format",
	"\\\"Error_GUID\\\"":                                                               "This error is allowed because it's an escaped JSON error response containing unsupported method errors",
	"tx already exists in cache":                                                       "This error is allowed because it can occur when transactions are retried during the test",
	"failed to create canonical form":                                                  "This error is allowed because it is caused by the relay processor not being able to create a canonical form",
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
	"Requested Epoch Is Too Old":                    "This error is allowed because it can occur during payment E2E test when epoch transitions happen during test execution",
	"Failed to get a provider session":              "This error is allowed because it can occur during payment E2E test when epoch validation fails temporarily",
	"Tried to Report to an older epoch":             "This error is allowed because it can occur during payment E2E test when there are temporary synchronization issues between providers and consumers during epoch transitions",
	"provider lava block":                           "This error is allowed because it's part of epoch synchronization messages during payment E2E test",
	"consumer lava block":                           "This error is allowed because it's part of epoch synchronization messages during payment E2E test",
}
