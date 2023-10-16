package processmanager

import (
	"os"
	"os/exec"
	"strings"

	"github.com/lavanet/lava/utils"
)

// TODOs:
// validate the binary that was created? if our lavap points to old its still bad.
// on bootstrap just download the right binary.
// try with which lavap, if it works dont use go path.

type ProtocolBinaryLinker struct {
	Fetcher *ProtocolBinaryFetcher
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
	goPath, err := pbl.Fetcher.VerifyGoInstallation()
	if err != nil {
		return "", utils.LavaFormatError("Couldn't get go binary path", err)
	}
	goBin, err := exec.Command(goPath, "env", "GOPATH").Output()
	if err != nil {
		return "", utils.LavaFormatError("Couldn't determine Go binary path", err)
	}

	goBinPath := strings.TrimSpace(string(goBin)) + "/bin/"
	pbl.validateBinaryExecutable(binaryPath)
	return goBinPath + "lavap", nil
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
	utils.LavaFormatInfo("Removed Link Successfully")
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
