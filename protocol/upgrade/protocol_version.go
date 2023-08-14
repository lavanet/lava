package upgrade

import (
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type ProtocolVersion struct {
	ConsumerVersion string
	ProviderVersion string
}

var lavaProtocolVersion = ProtocolVersion{
	ConsumerVersion: "0.21.0",
	ProviderVersion: "0.21.0",
}

func GetCurrentVersion() ProtocolVersion {
	return lavaProtocolVersion
}

func ValidateProtocolVersion(incoming *protocoltypes.Version) error {
	// check min version
	if incoming.ConsumerMin != lavaProtocolVersion.ConsumerVersion || incoming.ProviderMin != lavaProtocolVersion.ProviderVersion {
		utils.LavaFormatPanic("minimum protocol version mismatch!, you must update your protocol version to at least the minimum required protocol version",
			nil,
			utils.Attribute{Key: "required (on-chain) consumer minimum version:", Value: incoming.ConsumerMin},
			utils.Attribute{Key: "required (on-chain) provider minimum version", Value: incoming.ProviderMin},
			utils.Attribute{Key: "binary consumer version: ", Value: lavaProtocolVersion.ConsumerVersion},
			utils.Attribute{Key: "binary provider version: ", Value: lavaProtocolVersion.ProviderVersion},
		)
	}
	// check target version
	if incoming.ConsumerTarget != lavaProtocolVersion.ConsumerVersion || incoming.ProviderTarget != lavaProtocolVersion.ProviderVersion {
		return utils.LavaFormatError("target protocol version mismatch, there is a newer version available. We highly recommend to upgrade.",
			nil,
			utils.Attribute{Key: "required (on-chain) consumer target version:", Value: incoming.ConsumerTarget},
			utils.Attribute{Key: "required (on-chain) provider target version", Value: incoming.ProviderTarget},
			utils.Attribute{Key: "binary consumer version: ", Value: lavaProtocolVersion.ConsumerVersion},
			utils.Attribute{Key: "binary provider version: ", Value: lavaProtocolVersion.ProviderVersion},
		)
	}
	// version is ok.
	return nil
}
