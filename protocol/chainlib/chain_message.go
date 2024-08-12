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
	requestedBlocksHashes  []string
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

func (pm *baseChainMessageContainer) sortExtensions() {
	if len(pm.extensions) == 0 {
		return
	}

	sort.SliceStable(pm.extensions, func(i, j int) bool {
		return pm.extensions[i].Name < pm.extensions[j].Name
	})
}

func (pm *baseChainMessageContainer) GetRequestedBlocksHashes() []string {
	return pm.requestedBlocksHashes
}
func (bcnc *baseChainMessageContainer) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return bcnc.msg.SubscriptionIdExtractor(reply)
}

// returning parse directive for the api. can be nil.
func (bcnc *baseChainMessageContainer) GetParseDirective() *spectypes.ParseDirective {
	return bcnc.parseDirective
}

func (pm *baseChainMessageContainer) GetRawRequestHash() ([]byte, error) {
	if pm.inputHashCache != nil && len(pm.inputHashCache) > 0 {
		// Get the cached value
		return pm.inputHashCache, nil
	}
	hash, err := pm.msg.GetRawRequestHash()
	if err == nil {
		// Now we have the hash cached so we call it only once.
		pm.inputHashCache = hash
	}
	return hash, err
}

// not necessary for base chain message.
func (bcnc *baseChainMessageContainer) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	if bcnc.resultErrorParsingMethod == nil {
		utils.LavaFormatError("tried calling resultErrorParsingMethod when it is not set", nil)
		return false, ""
	}
	return bcnc.resultErrorParsingMethod(data, httpStatusCode)
}

func (bcnc *baseChainMessageContainer) TimeoutOverride(override ...time.Duration) time.Duration {
	if len(override) > 0 {
		bcnc.timeoutOverride = override[0]
	}
	return bcnc.timeoutOverride
}

func (bcnc *baseChainMessageContainer) SetForceCacheRefresh(force bool) bool {
	bcnc.forceCacheRefresh = force
	return bcnc.forceCacheRefresh
}

func (bcnc *baseChainMessageContainer) GetForceCacheRefresh() bool {
	return bcnc.forceCacheRefresh
}

func (bcnc *baseChainMessageContainer) DisableErrorHandling() {
	bcnc.msg.DisableErrorHandling()
}

func (bcnc baseChainMessageContainer) AppendHeader(metadata []pairingtypes.Metadata) {
	bcnc.msg.AppendHeader(metadata)
}

func (bcnc baseChainMessageContainer) GetApi() *spectypes.Api {
	return bcnc.api
}

func (bcnc baseChainMessageContainer) GetApiCollection() *spectypes.ApiCollection {
	return bcnc.apiCollection
}

func (bcnc baseChainMessageContainer) RequestedBlock() (latest int64, earliest int64) {
	if bcnc.earliestRequestedBlock == 0 {
		// earliest is optional and not set here
		return bcnc.latestRequestedBlock, bcnc.latestRequestedBlock
	}
	return bcnc.latestRequestedBlock, bcnc.earliestRequestedBlock
}

func (bcnc baseChainMessageContainer) GetRPCMessage() rpcInterfaceMessages.GenericMessage {
	return bcnc.msg
}

func (bcnc *baseChainMessageContainer) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modifiedOnLatestReq bool) {
	requestedBlock, _ := bcnc.RequestedBlock()
	if latestBlock <= spectypes.NOT_APPLICABLE || requestedBlock != spectypes.LATEST_BLOCK {
		return false
	}
	success := bcnc.msg.UpdateLatestBlockInMessage(uint64(latestBlock), modifyContent)
	if success {
		bcnc.latestRequestedBlock = latestBlock
		return true
	}
	return false
}

func (bcnc *baseChainMessageContainer) GetExtensions() []*spectypes.Extension {
	return bcnc.extensions
}

func (pm *baseChainMessageContainer) GetConcatenatedExtensions() string {
	extensionsNames := []string{}
	for _, extension := range pm.extensions {
		extensionsNames = append(extensionsNames, extension.Name)
	}
	return strings.Join(extensionsNames, ";")
}

// adds the following extensions
func (bcnc *baseChainMessageContainer) OverrideExtensions(extensionNames []string, extensionParser *extensionslib.ExtensionParser) {
	existingExtensions := map[string]struct{}{}
	for _, extension := range bcnc.extensions {
		existingExtensions[extension.Name] = struct{}{}
	}
	for _, extensionName := range extensionNames {
		if _, ok := existingExtensions[extensionName]; !ok {
			existingExtensions[extensionName] = struct{}{}
			extensionKey := extensionslib.ExtensionKey{
				Extension:      extensionName,
				ConnectionType: bcnc.apiCollection.CollectionData.Type,
				InternalPath:   bcnc.apiCollection.CollectionData.InternalPath,
				Addon:          bcnc.apiCollection.CollectionData.AddOn,
			}
			extension := extensionParser.GetExtension(extensionKey)
			if extension != nil {
				bcnc.extensions = append(bcnc.extensions, extension)
				bcnc.addExtensionCu(extension)
			}
		}
	}

	bcnc.sortExtensions()
}

func (bcnc *baseChainMessageContainer) SetExtension(extension *spectypes.Extension) {
	// TODO: Need locks?
	if len(bcnc.extensions) > 0 {
		for _, ext := range bcnc.extensions {
			if ext.Name == extension.Name {
				// already existing, no need to add
				return
			}
		}
		bcnc.extensions = append(bcnc.extensions, extension)
	} else {
		bcnc.extensions = []*spectypes.Extension{extension}
	}
	bcnc.addExtensionCu(extension)
	bcnc.sortExtensions()
}

func (pm *baseChainMessageContainer) RemoveExtension(extensionName string) {
	for _, ext := range pm.extensions {
		if ext.Name == extensionName {
			pm.extensions, _ = lavaslices.Remove(pm.extensions, ext)
			pm.removeExtensionCu(ext)
			break
		}
	}
	pm.sortExtensions()
}

func (bcnc *baseChainMessageContainer) addExtensionCu(extension *spectypes.Extension) {
	copyApi := *bcnc.api // we can't modify this because it points to an object inside the chainParser
	copyApi.ComputeUnits = uint64(math.Floor(float64(extension.GetCuMultiplier()) * float64(copyApi.ComputeUnits)))
	bcnc.api = &copyApi
}

func (pm *baseChainMessageContainer) removeExtensionCu(extension *spectypes.Extension) {
	copyApi := *pm.api // we can't modify this because it points to an object inside the chainParser
	copyApi.ComputeUnits = uint64(math.Floor(float64(copyApi.ComputeUnits) / float64(extension.GetCuMultiplier())))
	pm.api = &copyApi
}

type CraftData struct {
	Path           string
	Data           []byte
	ConnectionType string
}

func CraftChainMessage(parsing *spectypes.ParseDirective, connectionType string, chainParser ChainParser, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	return chainParser.CraftMessage(parsing, connectionType, craftData, metadata)
}
