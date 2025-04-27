package upgrade

import (
	"strconv"
	"strings"

	"github.com/lavanet/lava/v5/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v5/utils"
	protocoltypes "github.com/lavanet/lava/v5/x/protocol/types"
)

type ProtocolVersion struct {
	ConsumerVersion string
	ProviderVersion string
}

var lavaProtocolVersion = ProtocolVersion{
	ConsumerVersion: protocoltypes.DefaultVersion.ConsumerTarget,
	ProviderVersion: protocoltypes.DefaultVersion.ProviderTarget,
}

func GetCurrentVersion() ProtocolVersion {
	return lavaProtocolVersion
}

func (pv *ProtocolVersion) ValidateProtocolVersion(incoming *updaters.ProtocolVersionResponse) error {
	// version is ok.
	return nil
}

// if incoming >(Bigger then and not equal) current return true, otherwise return false
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
