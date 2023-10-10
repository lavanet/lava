package processmanager

import (
	"os"
	"os/exec"
	"strings"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils"
)

type ProtocolBinaryLinker struct {
}

func (pbl *ProtocolBinaryLinker) CreateLink(binaryPath string) error {
	dest, err := pbl.findLavaProtocolPath(binaryPath)
	if err != nil {
		return utils.LavaFormatError("Error in findLavaProtocolPath", err)
	}

	pbl.createAndVerifySymlink(binaryPath, dest)
	utils.LavaFormatInfo("Symbolic link created successfully.", utils.Attribute{Key: "Linked binary path", Value: dest})
	return nil
}

func (pbl *ProtocolBinaryLinker) findLavaProtocolPath(binaryPath string) (string, error) {
	out, err := exec.LookPath("lavap")
	if err == nil {
		return strings.TrimSpace(out), nil
	}
	return pbl.copyBinaryToSystemPath(binaryPath)
}

func (pbl *ProtocolBinaryLinker) copyBinaryToSystemPath(binaryPath string) (string, error) {
	gobin, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return "", utils.LavaFormatError("Couldn't determine Go binary path", err)
	}

	goBinPath := strings.TrimSpace(string(gobin)) + "/bin/"
	pbl.validateBinaryExecutable(binaryPath)
	pbl.removeExistingLink(goBinPath + "lavap")

	err = lvutil.Copy(binaryPath, goBinPath+"lavap")
	if err != nil {
		return "", utils.LavaFormatError("Couldn't copy binary to system path", err)
	}

	out, err := exec.LookPath("lavap")
	if err != nil {
		return "", utils.LavaFormatError("Couldn't find the binary in the system path", err)
	}
	return strings.TrimSpace(out), nil
}

func (pbl *ProtocolBinaryLinker) validateBinaryExecutable(path string) {
	version, err := exec.Command(path, "version").Output()
	if err != nil {
		utils.LavaFormatFatal("Binary is not a valid executable: ", err)
	}
	utils.LavaFormatInfo("Executable binary validated.", utils.Attribute{Key: "version", Value: strings.TrimSpace(string(version))})
}

func (pbl *ProtocolBinaryLinker) removeExistingLink(linkPath string) {
	if _, err := os.Lstat(linkPath); err == nil {
		utils.LavaFormatInfo("Discovered an existing link. Attempting to refresh.")
		if err := os.Remove(linkPath); err != nil {
			utils.LavaFormatFatal("Couldn't remove existing link", err)
		}
	} else if !os.IsNotExist(err) {
		utils.LavaFormatFatal("Unexpected error when checking for existing link", err)
	}
}

func (pbl *ProtocolBinaryLinker) createAndVerifySymlink(binaryPath, dest string) {
	if _, err := os.Lstat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			utils.LavaFormatFatal("Couldn't remove existing link", err)
		}
	}

	if err := os.Symlink(binaryPath, dest); err != nil {
		utils.LavaFormatFatal("Couldn't create symbolic link", err)
	}

	link, err := os.Readlink(dest)
	if err != nil || link != binaryPath {
		utils.LavaFormatFatal("Failed to verify symbolic link", err)
	}
}
