package processmanager

import (
	"os"
	"os/exec"
	"strings"

	"github.com/lavanet/lava/v2/utils"
)

// TODOs:
// validate the binary that was created? if our lavap points to old its still bad.
// on bootstrap just download the right binary.
// try with which lavap, if it works dont use go path.

type ProtocolBinaryLinker struct {
	Fetcher *ProtocolBinaryFetcher
}

func (pbl *ProtocolBinaryLinker) CreateLink(binaryPath string) error {
	dest, err := pbl.FindLavaProtocolPath(binaryPath)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] Error in findLavaProtocolPath", err)
	}

	pbl.createAndVerifySymlink(binaryPath, dest)
	utils.LavaFormatInfo("[Lavavisor] Symbolic link created successfully.", utils.Attribute{Key: "Linked binary path", Value: dest})
	return nil
}

func (pbl *ProtocolBinaryLinker) FindLavaProtocolPath(binaryPath string) (string, error) {
	out, err := exec.LookPath("lavap")
	if err == nil {
		return strings.TrimSpace(out), nil
	}
	// if failed searching for lavap using "which lavap", try searching it in the goBin directory
	return pbl.searchLavapInGoPath(binaryPath)
}

// sometimes lavap is not in this context's path. we are looking for it where we assume go will be located in.
func (pbl *ProtocolBinaryLinker) searchLavapInGoPath(binaryPath string) (string, error) {
	goPath, err := pbl.Fetcher.VerifyGoInstallation()
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] Couldn't get go binary path", err)
	}
	goBin, err := exec.Command(goPath, "env", "GOPATH").Output()
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] Couldn't determine Go binary path", err)
	}
	goBinPath := strings.TrimSpace(string(goBin)) + "/bin/"
	err = pbl.validateBinaryExecutable(binaryPath)
	if err != nil {
		// failed to validate binary path is exeutable we need to remove it and re download next block.
		if osErr := os.Remove(binaryPath); osErr != nil {
			return "", utils.LavaFormatError("[Lavavisor] Couldn't remove existing link", osErr)
		}
		return "", utils.LavaFormatError("[Lavavisor] lavavisor removed the binary path and will attempt to download the lavap binary again", nil)
	}
	return goBinPath + "lavap", nil
}

func (pbl *ProtocolBinaryLinker) validateBinaryExecutable(path string) error {
	version, err := exec.Command(path, "version").Output()
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] Binary is not a valid executable: ", err)
	}
	utils.LavaFormatInfo("[Lavavisor] Executable binary validated.", utils.Attribute{Key: "version", Value: strings.TrimSpace(string(version))})
	return nil
}

func (pbl *ProtocolBinaryLinker) removeExistingLink(linkPath string) {
	if _, err := os.Lstat(linkPath); err == nil {
		utils.LavaFormatInfo("[Lavavisor] Discovered an existing link. Attempting to refresh.")
		if err := os.Remove(linkPath); err != nil {
			utils.LavaFormatError("[Lavavisor] Couldn't remove existing link", err)
			return
		}
	} else if !os.IsNotExist(err) {
		utils.LavaFormatError("[Lavavisor] Unexpected error when checking for existing link", err)
		return
	}
	utils.LavaFormatInfo("[Lavavisor] Removed Link Successfully")
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
