package version_montior

import (
	"os/exec"
	"strings"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
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
		return lvutil.MinVersionMismatchError
	}
	// check target version
	if incoming.ConsumerTarget != binaryVersion || incoming.ProviderTarget != binaryVersion {
		return lvutil.TargetVersionMismatchError
	}
	// version is ok.
	return nil
}
