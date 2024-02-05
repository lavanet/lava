package rpcconsumer

import (
	"fmt"
	"strconv"

	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
)

type RelayErrors struct {
	relayErrors       []RelayError
	onFailureMergeAll bool
}

type relayErrorAppearances struct {
	indexes     []int // which indexes contain the errors
	appearances int   // number of appearances
}

func (r *RelayErrors) findMaxAppearances(input map[string]*relayErrorAppearances) (int, []int) {
	var maxVal int
	var maxValIndexArray []int // one of the indexes
	firstIteration := true
	for _, val := range input {
		if firstIteration || val.appearances > maxVal {
			maxVal = val.appearances
			maxValIndexArray = val.indexes
			firstIteration = false
		}
	}
	return maxVal, maxValIndexArray
}

func (r *RelayErrors) GetBestErrorMessageForUser() RelayError {
	bestIndex := -1
	bestResult := github_com_cosmos_cosmos_sdk_types.ZeroDec()
	errorMap := make(map[string]*relayErrorAppearances)
	for idx, relayError := range r.relayErrors {
		errorMessage := relayError.err.Error()
		value, ok := errorMap[errorMessage]
		if ok {
			value.appearances++
			value.indexes = append(value.indexes, idx)
		} else {
			errorMap[errorMessage] = &relayErrorAppearances{indexes: []int{idx}, appearances: 1}
		}
		if relayError.ProviderInfo.ProviderQoSExcellenceSummery.IsNil() || relayError.ProviderInfo.ProviderStake.Amount.IsNil() {
			continue
		}
		currentResult := relayError.ProviderInfo.ProviderQoSExcellenceSummery.MulInt(relayError.ProviderInfo.ProviderStake.Amount)
		if currentResult.GTE(bestResult) { // 0 or 1 here are valid replacements, so even 0 scores will return the error value
			bestResult.Set(currentResult)
			bestIndex = idx
		}
	}

	errorCount, indexes := r.findMaxAppearances(errorMap)
	if len(indexes) > 0 && errorCount >= (len(r.relayErrors)/2) {
		// we have majority of errors we can return this error.
		return r.relayErrors[indexes[0]]
	}

	if bestIndex != -1 {
		// Return the chosen error.
		// Print info for the consumer to know which errors happened
		utils.LavaFormatInfo("Failed all relays", utils.LogAttr("error_map", errorMap))
		return r.relayErrors[bestIndex]
	}
	// if we didn't manage to find any index return all.
	utils.LavaFormatError("Failed finding the best error index in GetErrorMessageForUser", nil, utils.LogAttr("relayErrors", r.relayErrors))
	if r.onFailureMergeAll {
		return RelayError{err: r.mergeAllErrors()}
	}
	// otherwise return the first element of the RelayErrors
	return r.relayErrors[0]
}

func (r *RelayErrors) getAllErrors() []error {
	allErrors := make([]error, len(r.relayErrors))
	for idx, relayError := range r.relayErrors {
		allErrors[idx] = relayError.err
	}
	return allErrors
}

func (r *RelayErrors) mergeAllErrors() error {
	mergedMessage := ""
	allErrors := r.getAllErrors()
	allErrorsLenght := len(allErrors)
	for idx, message := range allErrors {
		mergedMessage += strconv.Itoa(idx) + ". " + message.Error()
		if idx < allErrorsLenght {
			mergedMessage += ", "
		}
	}
	return fmt.Errorf(mergedMessage)
}

type RelayError struct {
	err          error
	ProviderInfo common.ProviderInfo
	response     *relayResponse
}
