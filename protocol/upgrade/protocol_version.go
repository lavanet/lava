package upgrade

import (
	"strconv"
	"strings"

	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type ProtocolVersion struct {
	ConsumerVersion string
	ProviderVersion string
}

var lavaProtocolVersion = ProtocolVersion{
	ConsumerVersion: "0.22.3",
	ProviderVersion: "0.22.3",
}

func GetCurrentVersion() ProtocolVersion {
	return lavaProtocolVersion
}

func (pv *ProtocolVersion) ValidateProtocolVersion(incoming *protocoltypes.Version) error {
	// check min version
	if HasVersionMismatch(incoming.ConsumerMin, lavaProtocolVersion.ConsumerVersion) || HasVersionMismatch(incoming.ProviderMin, lavaProtocolVersion.ProviderVersion) {
		utils.LavaFormatFatal("minimum protocol version mismatch!, you must update your protocol version to at least the minimum required protocol version",
			nil,
			utils.Attribute{Key: "required (on-chain) consumer minimum version:", Value: incoming.ConsumerMin},
			utils.Attribute{Key: "required (on-chain) provider minimum version", Value: incoming.ProviderMin},
			utils.Attribute{Key: "binary consumer version: ", Value: lavaProtocolVersion.ConsumerVersion},
			utils.Attribute{Key: "binary provider version: ", Value: lavaProtocolVersion.ProviderVersion},
		)
	}

	// check target version
	if HasVersionMismatch(incoming.ConsumerTarget, lavaProtocolVersion.ConsumerVersion) || HasVersionMismatch(incoming.ProviderTarget, lavaProtocolVersion.ProviderVersion) {
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

// if versions are different or incoming > current return true, otherwise return false
func HasVersionMismatch(incoming string, current string) bool {
	if incoming == current {
		return false
	}
	incomingParts := strings.Split(incoming, ".")
	currentParts := strings.Split(current, ".")
	incomingLen := len(incomingParts)
	currentLen := len(currentParts)

	for index := range incomingParts {
		if index >= incomingLen || index >= currentLen {
			// i.e incoming = 0.22.1.1, current = 0.22.1 == return true meaning current is not valid
			return incomingLen > currentLen // if we didn't return before getting to the end of the array, we can return which version has another element
		}
		if incomingParts[index] != currentParts[index] {
			// parse the part.
			incomingParsed, err := strconv.Atoi(incomingParts[index])
			if err != nil {
				utils.LavaFormatError("Failed parsing incomingParts[index] to Atoi", err)
				return false
			}
			currentParsed, err := strconv.Atoi(currentParts[index])
			if err != nil {
				utils.LavaFormatError("Failed parsing currentParts[index] to Atoi", err)
				return false
			}
			return incomingParsed > currentParsed
		}
	}
	return false
}
