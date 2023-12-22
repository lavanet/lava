package rpcconsumer

import (
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/maps"
)

type RelayErrors struct {
	relayErrors []RelayError
}

func (r *RelayErrors) GetBestErrorMessageForUser() utils.Attribute {
	bestIndex := -1
	bestResult := github_com_cosmos_cosmos_sdk_types.ZeroDec()
	errorMap := make(map[string]int)
	for idx, relayError := range r.relayErrors {
		errorMessage := relayError.err.Error()
		errorMap[errorMessage]++
		if relayError.ProviderInfo.ProviderQoSExcellenceSummery.IsNil() || relayError.ProviderInfo.ProviderStake.Amount.IsNil() {
			continue
		}
		currentResult := relayError.ProviderInfo.ProviderQoSExcellenceSummery.MulInt(relayError.ProviderInfo.ProviderStake.Amount)
		if currentResult.GTE(bestResult) { // 0 or 1 here are valid replacements, so even 0 scores will return the error value
			bestResult.Set(currentResult)
			bestIndex = idx
		}
	}

	errorKey, errorCount := maps.FindLargestIntValueInMap(errorMap)
	if errorCount >= (len(r.relayErrors) / 2) {
		// we have majority of errors we can return this error.
		return utils.LogAttr("error", errorKey)
	}

	if bestIndex != -1 {
		// Return the chosen error.
		// Print info for the consumer to know which errors happened
		utils.LavaFormatInfo("Failed all relays", utils.LogAttr("error_map", errorMap))
		return utils.LogAttr("error", r.relayErrors[bestIndex].err.Error())
	}
	// if we didn't manage to find any index return all.
	utils.LavaFormatError("Failed finding the best error index in GetErrorMessageForUser", nil, utils.LogAttr("relayErrors", r.relayErrors))
	return utils.LogAttr("errors", r.getAllErrors())
}

func (r *RelayErrors) getAllErrors() []error {
	allErrors := make([]error, len(r.relayErrors))
	for idx, relayError := range r.relayErrors {
		allErrors[idx] = relayError.err
	}
	return allErrors
}

type RelayError struct {
	err          error
	ProviderInfo common.ProviderInfo
}
