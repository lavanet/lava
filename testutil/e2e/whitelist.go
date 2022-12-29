package e2e

// This is a map of error strings with the description why they are whitelisted
// Detect extra text to be sure it is the whitelisted error
var whitelist = map[string]string{
	"getSupportedApi":        "This error is whitelisted because the tendermint uri tests have a test that checks if the error is catched.",
	"No pairings available.": "This error is caused when the network is just booted up and pairings are not yet done. If after a few seconds the pairings are still not available the e2e would fail because the initial check if the provider is responsive would time out.",
}
