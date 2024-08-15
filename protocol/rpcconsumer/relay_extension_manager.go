package rpcconsumer

import (
	"github.com/lavanet/lava/v2/protocol/chainlib"
	common "github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type RelayExtensionManager struct {
	chainMessage                 chainlib.ChainMessage
	relayRequestData             *pairingtypes.RelayPrivateData
	managedExtension             *spectypes.Extension
	extensionWasActiveOriginally bool
}

func NewRelayExtensionManager(chainMessage chainlib.ChainMessage, relayRequestData *pairingtypes.RelayPrivateData, managedExtension *spectypes.Extension) *RelayExtensionManager {
	relayExtensionManager := &RelayExtensionManager{
		chainMessage:     chainMessage,
		relayRequestData: relayRequestData,
		managedExtension: managedExtension,
	}
	relayExtensionManager.SetExtensionWasActiveOriginally()
	return relayExtensionManager
}

func (rem *RelayExtensionManager) SetExtensionWasActiveOriginally() {
	if lavaslices.ContainsPredicate(rem.chainMessage.GetExtensions(), rem.matchManagedExtension) {
		rem.extensionWasActiveOriginally = true
	}
}

func (rem *RelayExtensionManager) matchManagedExtension(extension *spectypes.Extension) bool {
	return extension.Name == rem.managedExtension.Name
}

func (rem *RelayExtensionManager) GetManagedExtensionName() string {
	return rem.managedExtension.Name
}

func (rem *RelayExtensionManager) SetManagedExtension() {
	if rem.extensionWasActiveOriginally {
		return // no need to set the extension as it was originally enabled
	}

	// set extensions on both relayRequestData and chainMessage.
	rem.chainMessage.SetExtension(rem.managedExtension)
	if lavaslices.Contains(rem.relayRequestData.Extensions, rem.managedExtension.Name) {
		utils.LavaFormatError("relayRequestData already contains extension", nil,
			utils.LogAttr("rem.relayRequestData.Extensions", rem.relayRequestData.Extensions),
			utils.LogAttr("rem.managedExtension.Name", rem.managedExtension.Name),
		)
	}
	// reset extension names to currently supported extensions
	rem.SetRelayDataExtensionsToChainMessageValues()
}

func (rem *RelayExtensionManager) SetRelayDataExtensionsToChainMessageValues() {
	rem.relayRequestData.Extensions = common.GetExtensionNames(rem.chainMessage.GetExtensions())
}

func (rem *RelayExtensionManager) IsExtensionActiveByDefault() bool {
	return rem.extensionWasActiveOriginally
}

func (rem *RelayExtensionManager) RemoveManagedExtension() {
	if !rem.extensionWasActiveOriginally {
		rem.chainMessage.RemoveExtension(rem.managedExtension.Name)
		if !lavaslices.Contains(rem.relayRequestData.Extensions, rem.managedExtension.Name) {
			utils.LavaFormatError("Asked to remove missing extension from relay request data", nil,
				utils.LogAttr("rem.relayRequestData.Extensions", rem.relayRequestData.Extensions),
				utils.LogAttr("rem.managedExtension.Name", rem.managedExtension.Name),
			)
		}
		// reset extension names to currently supported extensions
		rem.SetRelayDataExtensionsToChainMessageValues()
	}
}
