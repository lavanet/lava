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

const (
	DEFAULT_QUORUM_RATE = 0.66
	DEFAULT_QUORUM_MAX  = 5
	DEFAULT_QUORUM_MIN  = 2
)

func (bpm *BaseProtocolMessage) GetQuorumParameters() (common.QuorumParams, error) {
	var err error
	enabled := false
	var quorumRate float64
	var quorumMax int
	var quorumMin int

	quorumRateString, ok := bpm.directiveHeaders[common.QUORUM_HEADER_RATE]
	enabled = enabled || ok
	if !ok {
		quorumRate = DEFAULT_QUORUM_RATE
	} else {
		quorumRate, err = strconv.ParseFloat(quorumRateString, 64)
		if err != nil || quorumRate < 0 || quorumRate > 1 {
			return common.QuorumParams{}, errors.New("invalid quorum rate")
		}
	}

	quorumMaxRateString, ok := bpm.directiveHeaders[common.QUORUM_HEADER_MAX]
	enabled = enabled || ok
	if !ok {
		quorumMax = DEFAULT_QUORUM_MAX
	} else {
		quorumMax, err = strconv.Atoi(quorumMaxRateString)
		if err != nil || quorumMax < 0 {
			return common.QuorumParams{}, errors.New("invalid quorum max")
		}
	}

	quorumMinRateString, ok := bpm.directiveHeaders[common.QUORUM_HEADER_MIN]
	enabled = enabled || ok
	if !ok {
		quorumMin = DEFAULT_QUORUM_MIN
	} else {
		quorumMin, err = strconv.Atoi(quorumMinRateString)
		if err != nil || quorumMin < 0 {
			return common.QuorumParams{}, errors.New("invalid quorum min")
		}
	}

	if quorumMin > quorumMax {
		return common.QuorumParams{}, errors.New("quorum min is greater than quorum max")
	}

	if enabled {
		utils.LavaFormatInfo("Quorum parameters", utils.LogAttr("quorumRate", quorumRate), utils.LogAttr("quorumMax", quorumMax), utils.LogAttr("quorumMin", quorumMin))
		return common.QuorumParams{Rate: quorumRate, Max: quorumMax, Min: quorumMin}, nil
	} else {
		utils.LavaFormatInfo("Quorum parameters not enabled")
		return common.QuorumParams{Rate: 1, Max: 1, Min: 1}, nil
	}
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
	GetQuorumParameters() (common.QuorumParams, error)
}
