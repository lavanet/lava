package relaycore

import (
	"fmt"
	"regexp"
	"strconv"

	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
)

type RelayErrors struct {
	RelayErrors       []RelayError
	OnFailureMergeAll bool
}

// checking the errors that appeared the most and returning the number of errors that were the same and the index of one of them
func (r *RelayErrors) findMaxAppearances(input map[string][]int) (maxVal int, indexToReturn int) {
	var maxValIndexArray []int // one of the indexes
	for _, val := range input {
		if len(val) > maxVal {
			maxVal = len(val)
			maxValIndexArray = val
		}
	}
	if len(maxValIndexArray) > 0 {
		indexToReturn = maxValIndexArray[0]
	} else {
		indexToReturn = -1
	}
	return maxVal, indexToReturn
}

func replacePattern(input, pattern, replacement string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(input, replacement)
}

func (r *RelayErrors) sanitizeError(err error) string {
	errMsg := err.Error()
	// Replace SessionId:(any digit here) with SessionId:*
	errMsg = replacePattern(errMsg, `SessionId:\d+`, "SessionId:*")

	// Replace GUID:(any digit here) with GUID:*
	errMsg = replacePattern(errMsg, `GUID:\d+`, "GUID:*")

	return errMsg
}

// AddError adds a new error to the RelayErrors collection
func (r *RelayErrors) AddError(err RelayError) {
	r.RelayErrors = append(r.RelayErrors, err)
}

func (r *RelayErrors) GetBestErrorMessageForUser() RelayError {
	bestIndex := -1
	bestResult := github_com_cosmos_cosmos_sdk_types.ZeroDec()
	errorMap := make(map[string][]int)
	for idx, relayError := range r.RelayErrors {
		errorMessage := r.sanitizeError(relayError.Err)
		errorMap[errorMessage] = append(errorMap[errorMessage], idx)
		if relayError.ProviderInfo.ProviderReputationSummary.IsNil() || relayError.ProviderInfo.ProviderStake.Amount.IsNil() {
			continue
		}
		currentResult := relayError.ProviderInfo.ProviderReputationSummary.MulInt(relayError.ProviderInfo.ProviderStake.Amount)
		if currentResult.GTE(bestResult) { // 0 or 1 here are valid replacements, so even 0 scores will return the error value
			bestResult.Set(currentResult)
			bestIndex = idx
		}
	}

	errorCount, index := r.findMaxAppearances(errorMap)
	if index >= 0 && errorCount >= (len(r.RelayErrors)/2) {
		// we have majority of errors we can return this error.
		if r.RelayErrors[index].Response != nil {
			r.RelayErrors[index].Response.RelayResult.Quorum = errorCount
		}
		return r.RelayErrors[index]
	}

	if bestIndex != -1 {
		// Return the chosen error.
		// Print info for the consumer to know which errors happened
		utils.LavaFormatDebug("Failed all relays", utils.LogAttr("error_map", errorMap))
		return r.RelayErrors[bestIndex]
	}
	// if we didn't manage to find any index return all.
	utils.LavaFormatError("Failed finding the best error index in GetErrorMessageForUser", nil, utils.LogAttr("relayErrors", r.RelayErrors))
	if r.OnFailureMergeAll {
		return RelayError{Err: r.mergeAllErrors()}
	}
	// otherwise return the first element of the RelayErrors
	return r.RelayErrors[0]
}

func (r *RelayErrors) getAllUniqueErrors() []error {
	allErrors := []error{}
	repeatingErrors := make(map[string]struct{})
	for _, relayError := range r.RelayErrors {
		errString := r.sanitizeError(relayError.Err) // using strings to filter repeating errors
		_, ok := repeatingErrors[errString]
		if ok {
			continue
		}
		repeatingErrors[errString] = struct{}{}
		allErrors = append(allErrors, relayError.Err)
	}
	return allErrors
}

func (r *RelayErrors) mergeAllErrors() error {
	mergedMessage := ""
	allErrors := r.getAllUniqueErrors()
	allErrorsLength := len(allErrors)
	for idx, message := range allErrors {
		mergedMessage += strconv.Itoa(idx) + ". " + message.Error()
		if idx < allErrorsLength {
			mergedMessage += ", "
		}
	}
	return fmt.Errorf("%s", mergedMessage)
}

// RelayResponse represents a response from a relay operation
type RelayResponse struct {
	RelayResult common.RelayResult
	Err         error
}

// TODO: there's no need to save error twice and provider info twice, this can just be a RelayResponse
type RelayError struct {
	Err          error
	ProviderInfo common.ProviderInfo
	Response     *RelayResponse
}

func (re RelayError) String() string {
	return fmt.Sprintf("err: %s, ProviderInfo: %v, response: %v", re.Err, re.ProviderInfo, re.Response)
}

// GetError returns the underlying error
func (re RelayError) GetError() error {
	return re.Err
}
