package upgrade

import (
	"sync"

	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
)

type ProtocolVersion struct {
	ProviderTarget string
	ProviderMin    string
	ConsumerTarget string
	ConsumerMin    string
}

var LavaProtocolVersion = ProtocolVersion{
	ProviderTarget: "0.16.0",
	ProviderMin:    "0.16.0",
	ConsumerTarget: "0.16.0",
	ConsumerMin:    "0.16.0",
}

type UpgradeManager struct {
	lock sync.RWMutex
}

// Returning a new provider session manager
func NewUpdateManager() *UpgradeManager {
	return &UpgradeManager{}
}

func (um *UpgradeManager) SetProtocolVersion(newVersion *protocoltypes.Version) {
	um.lock.Lock()
	defer um.lock.Unlock()
	LavaProtocolVersion.ProviderMin = newVersion.ProviderMin
	LavaProtocolVersion.ProviderTarget = newVersion.ProviderTarget
	LavaProtocolVersion.ConsumerTarget = newVersion.ConsumerTarget
	LavaProtocolVersion.ConsumerMin = newVersion.ConsumerMin
}

func BuildVersionFromParamChangeEvent(event terderminttypes.Event) bool {
	expectedVersionKeys := map[string]bool{
		"Version.ProviderTarget": false,
		"Version.ProviderMin":    false,
		"Version.ConsumerTarget": false,
		"Version.ConsumerMin":    false,
	}
	for _, attribute := range event.Attributes {
		key := string(attribute.Key)
		if _, ok := expectedVersionKeys[key]; ok {
			expectedVersionKeys[key] = true
		}
	}
	for _, found := range expectedVersionKeys {
		if !found {
			return false
		}
	}
	return true
}
