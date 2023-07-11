package upgrade

import (
	"encoding/json"
	"sync"

	"github.com/lavanet/lava/utils"
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

func (um *UpgradeManager) SetProtocolVersion(newVersion *ProtocolVersion) {
	LavaProtocolVersion.ProviderMin = newVersion.ProviderMin
	LavaProtocolVersion.ProviderTarget = newVersion.ProviderTarget
	LavaProtocolVersion.ConsumerTarget = newVersion.ConsumerTarget
	LavaProtocolVersion.ConsumerMin = newVersion.ConsumerMin
}

// @audit return boolean here
func BuildVersionFromParamChangeEvent(event terderminttypes.Event) (*ProtocolVersion, error) {
	attributes := map[string]string{}
	// slices contains
	for _, attribute := range event.Attributes {
		attributes[string(attribute.Key)] = string(attribute.Value)
	}
	paramValue, ok := attributes["param"]
	if !ok || paramValue != "Version" {
		return nil, utils.LavaFormatError("failed building BuildVersionFromParamChangeEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}

	versionValue, ok := attributes["value"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVersionFromParamChangeEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	var version *ProtocolVersion
	err := json.Unmarshal([]byte(versionValue), &version)
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVersionFromParamChangeEvent", err, utils.Attribute{Key: "attributes", Value: attributes})
	}

	return version, nil
}
