package chainlib

import (
	"errors"
	"strconv"
	"strings"

	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

type UserData struct {
	ConsumerIp string
	DappId     string
}

type BaseProtocolMessage struct {
	ChainMessage
	directiveHeaders map[string]string
	relayRequestData *pairingtypes.RelayPrivateData
	userData         common.UserData
}

func (bpm *BaseProtocolMessage) IsDefaultApi() bool {
	api := bpm.GetApi()
	return strings.HasPrefix(api.Name, DefaultApiName) && api.BlockParsing.ParserFunc == spectypes.PARSER_FUNC_DEFAULT
}

func (bpm *BaseProtocolMessage) GetUserData() common.UserData {
	return bpm.userData
}

func (bpm *BaseProtocolMessage) GetDirectiveHeaders() map[string]string {
	return bpm.directiveHeaders
}

func (bpm *BaseProtocolMessage) RelayPrivateData() *pairingtypes.RelayPrivateData {
	return bpm.relayRequestData
}

func (bpm *BaseProtocolMessage) HashCacheRequest(chainId string) ([]byte, func([]byte) []byte, error) {
	return HashCacheRequest(bpm.relayRequestData, chainId)
}

// addMissingExtensions adds any extensions from updatedProtocolExtensions that are not in currentPrivateDataExtensions
func (bpm *BaseProtocolMessage) addMissingExtensions(updatedProtocolExtensions []*spectypes.Extension, currentPrivateDataExtensions []string) []string {
	// Create a map for O(1) lookups
	existingExtensions := make(map[string]struct{}, len(currentPrivateDataExtensions))
	for _, ext := range currentPrivateDataExtensions {
		existingExtensions[ext] = struct{}{}
	}

	// Add missing extensions
	for _, ext := range updatedProtocolExtensions {
		if _, exists := existingExtensions[ext.Name]; !exists {
			currentPrivateDataExtensions = append(currentPrivateDataExtensions, ext.Name)
			if len(updatedProtocolExtensions) == len(currentPrivateDataExtensions) {
				break
			}
		}
	}
	return currentPrivateDataExtensions
}

func (bpm *BaseProtocolMessage) UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64) bool {
	if earliestBlockHashRequested >= 0 {
		success := bpm.UpdateEarliestInMessage(earliestBlockHashRequested)
		// check if we successfully updated the earliest block in the message
		if success {
			// parse the extensions for the new updated earliest block
			extensionParser.ExtensionParsing(addon, bpm, uint64(seenBlock))
			updatedProtocolExtensions := bpm.GetExtensions()
			currentPrivateDataExtensions := bpm.RelayPrivateData().Extensions
			utils.LavaFormatTrace("[Archive Debug] Trying to add extensions", utils.LogAttr("currentProtocolExtensions", updatedProtocolExtensions), utils.LogAttr("currentPrivateDataExtensions", currentPrivateDataExtensions))
			if len(updatedProtocolExtensions) > len(currentPrivateDataExtensions) {
				// we need to add the missing extension to the private data.
				currentPrivateDataExtensions = bpm.addMissingExtensions(updatedProtocolExtensions, currentPrivateDataExtensions)
				bpm.RelayPrivateData().Extensions = currentPrivateDataExtensions
				utils.LavaFormatTrace("[Archive Debug] After Swap", utils.LogAttr("bpm.RelayPrivateData().Extensions", bpm.RelayPrivateData().Extensions))
				return true
			}
		}
	}
	return false
}

func (bpm *BaseProtocolMessage) GetBlockedProviders() []string {
	if bpm.directiveHeaders == nil {
		return nil
	}
	blockedProviders, ok := bpm.directiveHeaders[common.BLOCK_PROVIDERS_ADDRESSES_HEADER_NAME]
	if ok {
		blockProviders := strings.Split(blockedProviders, ",")
		if len(blockProviders) <= 2 {
			return blockProviders
		}
	}
	return nil
}

func NewProtocolMessage(chainMessage ChainMessage, directiveHeaders map[string]string, relayRequestData *pairingtypes.RelayPrivateData, dappId, consumerIp string) ProtocolMessage {
	return &BaseProtocolMessage{
		ChainMessage:     chainMessage,
		directiveHeaders: directiveHeaders,
		relayRequestData: relayRequestData,
		userData:         common.UserData{DappId: dappId, ConsumerIp: consumerIp},
	}
}

// GetCrossValidationParameters parses cross-validation headers from the request.
// Returns (params, headersPresent, error).
// - headersPresent is true if any cross-validation header was found (used by state machine to set Selection)
// - error is returned if headers are present but invalid
func (bpm *BaseProtocolMessage) GetCrossValidationParameters() (common.CrossValidationParams, bool, error) {
	var maxParticipants, agreementThreshold int
	var err error

	// Check if max-participants header is present
	maxParticipantsStr, maxPresent := bpm.directiveHeaders[common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS]
	// Check if agreement-threshold header is present
	agreementThresholdStr, thresholdPresent := bpm.directiveHeaders[common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD]

	// If no cross-validation headers are present, return defaults with headersPresent=false
	if !maxPresent && !thresholdPresent {
		return common.DefaultCrossValidationParams, false, nil
	}

	// At least one header is present - parse and validate both
	if maxPresent {
		maxParticipants, err = strconv.Atoi(maxParticipantsStr)
		if err != nil || maxParticipants < 1 {
			return common.CrossValidationParams{}, true, errors.New("invalid cross-validation max-participants: must be a positive integer")
		}
	} else {
		return common.CrossValidationParams{}, true, errors.New("cross-validation max-participants header is required when using cross-validation")
	}

	if thresholdPresent {
		agreementThreshold, err = strconv.Atoi(agreementThresholdStr)
		if err != nil || agreementThreshold < 1 {
			return common.CrossValidationParams{}, true, errors.New("invalid cross-validation agreement-threshold: must be a positive integer")
		}
	} else {
		return common.CrossValidationParams{}, true, errors.New("cross-validation agreement-threshold header is required when using cross-validation")
	}

	// Validate that agreementThreshold <= maxParticipants
	if agreementThreshold > maxParticipants {
		return common.CrossValidationParams{}, true, errors.New("cross-validation agreement-threshold cannot be greater than max-participants")
	}

	utils.LavaFormatInfo("CrossValidation parameters parsed",
		utils.LogAttr("maxParticipants", maxParticipants),
		utils.LogAttr("agreementThreshold", agreementThreshold))

	return common.CrossValidationParams{
		MaxParticipants:    maxParticipants,
		AgreementThreshold: agreementThreshold,
	}, true, nil
}

type ProtocolMessage interface {
	ChainMessage
	GetDirectiveHeaders() map[string]string
	RelayPrivateData() *pairingtypes.RelayPrivateData
	HashCacheRequest(chainId string) ([]byte, func([]byte) []byte, error)
	GetBlockedProviders() []string
	GetUserData() common.UserData
	IsDefaultApi() bool
	UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64) bool
	// GetCrossValidationParameters returns (params, headersPresent, error)
	// headersPresent indicates if cross-validation headers were found (used to set Selection type)
	GetCrossValidationParameters() (common.CrossValidationParams, bool, error)
}
