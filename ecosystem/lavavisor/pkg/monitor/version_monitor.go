package version_montior

import (
	"os/exec"
	"strings"

	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

func getBinaryVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", utils.LavaFormatError("failed to execute command", err)
	}

	// output format is "lava-protocol version x.x.x"
	version := strings.Split(string(output), " ")[2]
	return version, nil
}

func ValidateProtocolBinaryVersion(incoming *protocoltypes.Version, binaryPath string) error {
	binaryVersion, err := getBinaryVersion(binaryPath)
	if err != nil {
		return utils.LavaFormatError("failed to get binary version", err)
	}

	// check min version
	if incoming.ConsumerMin != binaryVersion || incoming.ProviderMin != binaryVersion {
		utils.LavaFormatError("minimum protocol version mismatch!",
			nil,
			utils.Attribute{Key: "required (on-chain) consumer minimum version:", Value: incoming.ConsumerMin},
			utils.Attribute{Key: "required (on-chain) provider minimum version", Value: incoming.ProviderMin},
			utils.Attribute{Key: "binary consumer version: ", Value: binaryVersion},
			utils.Attribute{Key: "binary provider version: ", Value: binaryVersion},
		)
	}
	// check target version
	if incoming.ConsumerTarget != binaryVersion || incoming.ProviderTarget != binaryVersion {
		return utils.LavaFormatError("target protocol version mismatch!",
			nil,
			utils.Attribute{Key: "required (on-chain) consumer target version:", Value: incoming.ConsumerTarget},
			utils.Attribute{Key: "required (on-chain) provider target version", Value: incoming.ProviderTarget},
			utils.Attribute{Key: "binary consumer version: ", Value: binaryVersion},
			utils.Attribute{Key: "binary provider version: ", Value: binaryVersion},
		)
	}
	// version is ok.
	return nil
}
