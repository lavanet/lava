package rpcconsumer

import (
	"github.com/lavanet/lava/v2/protocol/chainlib"
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

	if lavaslices.ContainsPredicate(chainMessage.GetExtensions(), relayExtensionManager.matchManagedExtension) {
		relayExtensionManager.extensionWasActiveOriginally = true
	}

	return relayExtensionManager
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
		return
	}
	rem.relayRequestData.Extensions = append(rem.relayRequestData.Extensions, rem.managedExtension.Name)
}

func (rem *RelayExtensionManager) IsExtensionActiveByDefault() bool {
	return rem.extensionWasActiveOriginally
}

func (rem *RelayExtensionManager) RemoveManagedExtension() {
	if !rem.extensionWasActiveOriginally {
		var success bool
		rem.relayRequestData.Extensions, success = lavaslices.Remove(rem.relayRequestData.Extensions, rem.managedExtension.Name)
		if !success {
			utils.LavaFormatError("Asked to remove missing extension from relay request data", nil,
				utils.LogAttr("rem.relayRequestData.Extensions", rem.relayRequestData.Extensions),
				utils.LogAttr("rem.managedExtension.Name", rem.managedExtension.Name),
			)
		}
		rem.chainMessage.RemoveExtension(rem.managedExtension.Name)
	}
}
