package chainlib

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type updatableRPCInput interface {
	rpcInterfaceMessages.GenericMessage
	UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool)
	AppendHeader(metadata []pairingtypes.Metadata)
	SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string
	GetRawRequestHash() ([]byte, error)
}

type baseChainMessageContainer struct {
	api                    *spectypes.Api
	latestRequestedBlock   int64
	requestedBlockHashes   []string
	earliestRequestedBlock int64
	msg                    updatableRPCInput
	apiCollection          *spectypes.ApiCollection
	extensions             []*spectypes.Extension
	timeoutOverride        time.Duration
	forceCacheRefresh      bool
	parseDirective         *spectypes.ParseDirective // setting the parse directive related to the api, can be nil

	inputHashCache []byte
	// resultErrorParsingMethod passed by each api interface message to parse the result of the message
	// and validate it doesn't contain a node error
	resultErrorParsingMethod func(data []byte, httpStatusCode int) (hasError bool, errorMessage string)
}

// Used to create the key for used providers so all extensions are
// always in the same order. e.g. "archive;ws"
func (bcmc *baseChainMessageContainer) sortExtensions() {
	if len(bcmc.extensions) <= 1 { // nothing to sort
		return
	}

	sort.SliceStable(bcmc.extensions, func(i, j int) bool {
		return bcmc.extensions[i].Name < bcmc.extensions[j].Name
	})
}

func (bcmc *baseChainMessageContainer) GetRequestedBlocksHashes() []string {
	return bcmc.requestedBlockHashes
}

func (bcmc *baseChainMessageContainer) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return bcmc.msg.SubscriptionIdExtractor(reply)
}

// returning parse directive for the api. can be nil.
func (bcmc *baseChainMessageContainer) GetParseDirective() *spectypes.ParseDirective {
	return bcmc.parseDirective
}

func (bcmc *baseChainMessageContainer) GetRawRequestHash() ([]byte, error) {
	if bcmc.inputHashCache != nil && len(bcmc.inputHashCache) > 0 {
		// Get the cached value
		return bcmc.inputHashCache, nil
	}
	hash, err := bcmc.msg.GetRawRequestHash()
	if err == nil {
		// Now we have the hash cached so we call it only once.
		bcmc.inputHashCache = hash
	}
	return hash, err
}

// not necessary for base chain message.
func (bcmc *baseChainMessageContainer) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	if bcmc.resultErrorParsingMethod == nil {
		utils.LavaFormatError("tried calling resultErrorParsingMethod when it is not set", nil)
		return false, ""
	}
	return bcmc.resultErrorParsingMethod(data, httpStatusCode)
}

func (bcmc *baseChainMessageContainer) TimeoutOverride(override ...time.Duration) time.Duration {
	if len(override) > 0 {
		bcmc.timeoutOverride = override[0]
	}
	return bcmc.timeoutOverride
}

func (bcmc *baseChainMessageContainer) SetForceCacheRefresh(force bool) bool {
	bcmc.forceCacheRefresh = force
	return bcmc.forceCacheRefresh
}

func (bcmc *baseChainMessageContainer) GetForceCacheRefresh() bool {
	return bcmc.forceCacheRefresh
}

func (bcmc *baseChainMessageContainer) DisableErrorHandling() {
	bcmc.msg.DisableErrorHandling()
}

func (bcmc baseChainMessageContainer) AppendHeader(metadata []pairingtypes.Metadata) {
	bcmc.msg.AppendHeader(metadata)
}

func (bcmc baseChainMessageContainer) GetApi() *spectypes.Api {
	return bcmc.api
}

func (bcmc baseChainMessageContainer) GetApiCollection() *spectypes.ApiCollection {
	return bcmc.apiCollection
}

func (bcmc *baseChainMessageContainer) CompareAndSwapEarliestRequestedBlockIfApplicable(incomingEarliest int64) bool {
	swapped := false
	if bcmc.earliestRequestedBlock != spectypes.EARLIEST_BLOCK {
		if bcmc.earliestRequestedBlock > incomingEarliest {
			bcmc.earliestRequestedBlock = incomingEarliest
			swapped = true
		}
	}
	return swapped
}

func (bcmc *baseChainMessageContainer) RequestedBlock() (latest int64, earliest int64) {
	if bcmc.earliestRequestedBlock == 0 {
		// earliest is optional and not set here
		return bcmc.latestRequestedBlock, bcmc.latestRequestedBlock
	}
	return bcmc.latestRequestedBlock, bcmc.earliestRequestedBlock
}

func (bcmc baseChainMessageContainer) GetRPCMessage() rpcInterfaceMessages.GenericMessage {
	return bcmc.msg
}

func (bcmc *baseChainMessageContainer) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modifiedOnLatestReq bool) {
	requestedBlock, _ := bcmc.RequestedBlock()
	if latestBlock <= spectypes.NOT_APPLICABLE || requestedBlock != spectypes.LATEST_BLOCK {
		return false
	}
	success := bcmc.msg.UpdateLatestBlockInMessage(uint64(latestBlock), modifyContent)
	if success {
		bcmc.latestRequestedBlock = latestBlock
		return true
	}
	return false
}

func (bcmc *baseChainMessageContainer) GetExtensions() []*spectypes.Extension {
	return bcmc.extensions
}

func (bcmc *baseChainMessageContainer) GetConcatenatedExtensions() string {
	extensionsNames := []string{}
	for _, extension := range bcmc.extensions {
		extensionsNames = append(extensionsNames, extension.Name)
	}
	return strings.Join(extensionsNames, ";")
}

// adds the following extensions
func (bcmc *baseChainMessageContainer) OverrideExtensions(extensionNames []string, extensionParser *extensionslib.ExtensionParser) {
	existingExtensions := map[string]struct{}{}
	for _, extension := range bcmc.extensions {
		existingExtensions[extension.Name] = struct{}{}
	}
	for _, extensionName := range extensionNames {
		if _, ok := existingExtensions[extensionName]; !ok {
			existingExtensions[extensionName] = struct{}{}
			extensionKey := extensionslib.ExtensionKey{
				Extension:      extensionName,
				ConnectionType: bcmc.apiCollection.CollectionData.Type,
				InternalPath:   bcmc.apiCollection.CollectionData.InternalPath,
				Addon:          bcmc.apiCollection.CollectionData.AddOn,
			}
			extension := extensionParser.GetExtension(extensionKey)
			if extension != nil {
				bcmc.extensions = append(bcmc.extensions, extension)
				bcmc.addExtensionCu(extension)
				bcmc.sortExtensions()
			}
		}
	}
}

func (bcmc *baseChainMessageContainer) SetExtension(extension *spectypes.Extension) {
	if len(bcmc.extensions) > 0 {
		for _, ext := range bcmc.extensions {
			if ext.Name == extension.Name {
				// already existing, no need to add
				return
			}
		}
		bcmc.extensions = append(bcmc.extensions, extension)
		bcmc.sortExtensions()
	} else {
		bcmc.extensions = []*spectypes.Extension{extension}
	}
	bcmc.addExtensionCu(extension)
}

func (bcmc *baseChainMessageContainer) RemoveExtension(extensionName string) {
	for _, ext := range bcmc.extensions {
		if ext.Name == extensionName {
			bcmc.extensions, _ = lavaslices.Remove(bcmc.extensions, ext)
			bcmc.removeExtensionCu(ext)
			break
		}
	}
	bcmc.sortExtensions()
}

func (bcmc *baseChainMessageContainer) addExtensionCu(extension *spectypes.Extension) {
	copyApi := *bcmc.api // we can't modify this because it points to an object inside the chainParser
	copyApi.ComputeUnits = uint64(math.Floor(float64(extension.GetCuMultiplier()) * float64(copyApi.ComputeUnits)))
	bcmc.api = &copyApi
}

func (bcmc *baseChainMessageContainer) removeExtensionCu(extension *spectypes.Extension) {
	copyApi := *bcmc.api // we can't modify this because it points to an object inside the chainParser
	copyApi.ComputeUnits = uint64(math.Floor(float64(copyApi.ComputeUnits) / float64(extension.GetCuMultiplier())))
	bcmc.api = &copyApi
}

type CraftData struct {
	Path           string
	Data           []byte
	ConnectionType string
}

func CraftChainMessage(parsing *spectypes.ParseDirective, connectionType string, chainParser ChainParser, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	return chainParser.CraftMessage(parsing, connectionType, craftData, metadata)
}
