package chainlib

import (
	"math"
	"time"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type updatableRPCInput interface {
	rpcInterfaceMessages.GenericMessage
	UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool)
	AppendHeader(metadata []pairingtypes.Metadata)
}

type baseChainMessageContainer struct {
	api                    *spectypes.Api
	latestRequestedBlock   int64
	earliestRequestedBlock int64
	msg                    updatableRPCInput
	apiCollection          *spectypes.ApiCollection
	extensions             []*spectypes.Extension
	timeoutOverride        time.Duration
	forceCacheRefresh      bool
	// resultErrorParsingMethod passed by each api interface message to parse the result of the message
	// and validate it doesn't contain a node error
	resultErrorParsingMethod func(data []byte, httpStatusCode int) (hasError bool, errorMessage string)
}

// not necessary for base chain message.
func (pm *baseChainMessageContainer) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	if pm.resultErrorParsingMethod == nil {
		utils.LavaFormatError("tried calling resultErrorParsingMethod when it is not set", nil)
		return false, ""
	}
	return pm.resultErrorParsingMethod(data, httpStatusCode)
}

func (pm *baseChainMessageContainer) TimeoutOverride(override ...time.Duration) time.Duration {
	if len(override) > 0 {
		pm.timeoutOverride = override[0]
	}
	return pm.timeoutOverride
}

func (pm *baseChainMessageContainer) SetForceCacheRefresh(force bool) bool {
	pm.forceCacheRefresh = force
	return pm.forceCacheRefresh
}

func (pm *baseChainMessageContainer) GetForceCacheRefresh() bool {
	return pm.forceCacheRefresh
}

func (pm *baseChainMessageContainer) DisableErrorHandling() {
	pm.msg.DisableErrorHandling()
}

func (pm baseChainMessageContainer) AppendHeader(metadata []pairingtypes.Metadata) {
	pm.msg.AppendHeader(metadata)
}

func (pm baseChainMessageContainer) GetApi() *spectypes.Api {
	return pm.api
}

func (pm baseChainMessageContainer) GetApiCollection() *spectypes.ApiCollection {
	return pm.apiCollection
}

func (pm baseChainMessageContainer) RequestedBlock() (latest int64, earliest int64) {
	if pm.earliestRequestedBlock == 0 {
		// earliest is optional and not set here
		return pm.latestRequestedBlock, pm.latestRequestedBlock
	}
	return pm.latestRequestedBlock, pm.earliestRequestedBlock
}

func (pm baseChainMessageContainer) GetRPCMessage() rpcInterfaceMessages.GenericMessage {
	return pm.msg
}

func (pm *baseChainMessageContainer) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modifiedOnLatestReq bool) {
	requestedBlock, _ := pm.RequestedBlock()
	if latestBlock <= spectypes.NOT_APPLICABLE || requestedBlock != spectypes.LATEST_BLOCK {
		return false
	}
	success := pm.msg.UpdateLatestBlockInMessage(uint64(latestBlock), modifyContent)
	if success {
		pm.latestRequestedBlock = latestBlock
		return true
	}
	return false
}

func (pm *baseChainMessageContainer) GetExtensions() []*spectypes.Extension {
	return pm.extensions
}

// adds the following extensions
func (pm *baseChainMessageContainer) OverrideExtensions(extensionNames []string, extensionParser *extensionslib.ExtensionParser) {
	existingExtensions := map[string]struct{}{}
	for _, extension := range pm.extensions {
		existingExtensions[extension.Name] = struct{}{}
	}
	for _, extensionName := range extensionNames {
		if _, ok := existingExtensions[extensionName]; !ok {
			existingExtensions[extensionName] = struct{}{}
			extensionKey := extensionslib.ExtensionKey{
				Extension:      extensionName,
				ConnectionType: pm.apiCollection.CollectionData.Type,
				InternalPath:   pm.apiCollection.CollectionData.InternalPath,
				Addon:          pm.apiCollection.CollectionData.AddOn,
			}
			extension := extensionParser.GetExtension(extensionKey)
			if extension != nil {
				pm.extensions = append(pm.extensions, extension)
				pm.updateCUForApi(extension)
			}
		}
	}
}

func (pm *baseChainMessageContainer) SetExtension(extension *spectypes.Extension) {
	if len(pm.extensions) > 0 {
		for _, ext := range pm.extensions {
			if ext.Name == extension.Name {
				// already existing, no need to add
				return
			}
		}
		pm.extensions = append(pm.extensions, extension)
	} else {
		pm.extensions = []*spectypes.Extension{extension}
	}
	pm.updateCUForApi(extension)
}

func (pm *baseChainMessageContainer) updateCUForApi(extension *spectypes.Extension) {
	copyApi := *pm.api // we can't modify this because it points to an object inside the chainParser
	copyApi.ComputeUnits = uint64(math.Floor(float64(extension.GetCuMultiplier()) * float64(copyApi.ComputeUnits)))
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
