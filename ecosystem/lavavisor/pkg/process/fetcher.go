package processmanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

func FetchProtocolBinary(lavavisorPath string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version) (selectedBinaryPath string, err error) {
	currentVersion := lvutil.ParseToSemanticVersion(protocolConsensusVersion.ProviderTarget)
	minVersion := lvutil.ParseToSemanticVersion(protocolConsensusVersion.ProviderMin)

	for ; !lvutil.IsVersionLessThan(currentVersion, minVersion); lvutil.DecrementVersion(currentVersion) {
		utils.LavaFormatInfo("Trying to fetch", utils.Attribute{Key: "version", Value: lvutil.FormatFromSemanticVersion(currentVersion)})
		versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+lvutil.FormatFromSemanticVersion(currentVersion))
		selectedBinaryPath, err = checkAndHandleVersionDir(versionDir, autoDownload, protocolConsensusVersion, currentVersion)
		if err == nil {
			return selectedBinaryPath, nil
		}
	}

	return "", utils.LavaFormatError("Failed to fetch protocol binary for both target and min versions", nil)
}

func SetupLavavisorDir(dir string) (lavavisorPath string, err error) {
	lavavisorPath, err = buildLavavisorPath(dir)
	if err != nil {
		return "", err
	}
	// Check if ./lava-visor directory exists
	if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
		// If not, create the directory
		err = setUpLavavisorDirectory(lavavisorPath)
		if err != nil {
			return "", utils.LavaFormatError("unable to create .lava-visor/ directory", err)
		}
		utils.LavaFormatInfo(".lava-visor/ folder successfully created", utils.Attribute{Key: "path:", Value: lavavisorPath})
	}
	return lavavisorPath, nil
}

func ValidateLavavisorDir(dir string) (lavavisorPath string, err error) {
	lavavisorPath, err = buildLavavisorPath(dir)
	if err != nil {
		return "", err
	}
	// Validate the existence of ./lavavisor directory
	if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
		return "", utils.LavaFormatError("lavavisor directory is not found", err)
	}
	return lavavisorPath, nil
}

func buildLavavisorPath(dir string) (string, error) {
	dir, err := lvutil.ExpandTilde(dir)
	if err != nil {
		return "", utils.LavaFormatError("unable to expand directory path", err)
	}
	// Build path to ./lavavisor
	return filepath.Join(dir, ".lava-visor"), nil
}

func setUpLavavisorDirectory(lavavisorPath string) error {
	err := os.MkdirAll(lavavisorPath, 0o755)
	if err != nil {
		return utils.LavaFormatError("unable to create .lava-visor/ directory", err)
	}
	// Create config.yml file inside .lava-visor and write placeholder text
	configPath := filepath.Join(lavavisorPath, "config.yml")
	configFile, err := os.Create(configPath)
	if err != nil {
		return utils.LavaFormatError("unable to create or clean config.yml", err)
	}
	defer configFile.Close() // Close the file

	// Create 'upgrades' directory inside .lava-visor
	upgradesPath := filepath.Join(lavavisorPath, "upgrades")
	if _, err := os.Stat(upgradesPath); os.IsNotExist(err) {
		err = os.MkdirAll(upgradesPath, 0o755)
		if err != nil {
			return utils.LavaFormatError("unable to create 'upgrades' directory", err)
		}
	}
	return nil
}

func checkAndHandleVersionDir(versionDir string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version, currentVersion *lvutil.SemanticVer) (selectedBinaryPath string, err error) {
	var binaryPath string
	if dirExists(versionDir) {
		binaryPath, err = handleExistingDir(versionDir, autoDownload, protocolConsensusVersion, currentVersion)
		if err != nil {
			return "", err
		}
	} else {
		binaryPath, err = handleMissingDir(versionDir, autoDownload, currentVersion)
		if err != nil {
			return "", err
		}
	}
	// validate binary version after it has been set
	vm := VersionMonitor{
		BinaryPath: binaryPath,
	}
	if err := vm.ValidateProtocolVersion(protocolConsensusVersion); err != nil {
		return "", err
	}

	utils.LavaFormatInfo("Protocol binary with target version has been successfully set!")
	return binaryPath, nil
}

func dirExists(versionDir string) bool {
	_, err := os.Stat(versionDir)
	return !os.IsNotExist(err)
}

func handleMissingDir(versionDir string, autoDownload bool, currentVersion *lvutil.SemanticVer) (binaryPath string, err error) {
	if !autoDownload {
		return "", utils.LavaFormatError("Sub-directory for version not found and auto-download is disabled.", nil, utils.Attribute{Key: "Version", Value: currentVersion})
	}
	utils.LavaFormatInfo("Version directory does not exist, but auto-download is enabled. Attempting to download binary from GitHub...")
	utils.LavaFormatInfo("Trying to download:", utils.Attribute{Key: "Version", Value: currentVersion})
	os.MkdirAll(versionDir, os.ModePerm)
	if err := downloadAndBuildFromGithub(lvutil.FormatFromSemanticVersion(currentVersion), versionDir); err == nil {
		binaryPath = filepath.Join(versionDir, "lavap")
		return binaryPath, nil
	}
	// upon failed operation, remove versionDir
	err = os.RemoveAll(versionDir)
	if err != nil {
		return "", err
	}

	return "", utils.LavaFormatError("Failed to auto-download binary from GitHub\n ", nil)
}

func handleExistingDir(versionDir string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version, currentVersion *lvutil.SemanticVer) (binaryPath string, err error) {
	binaryPath = filepath.Join(versionDir, "lavap")
	vm := VersionMonitor{
		BinaryPath: binaryPath,
	}
	if err := vm.ValidateProtocolVersion(protocolConsensusVersion); err != nil {
		if !autoDownload {
			return "", utils.LavaFormatError("Protocol version mismatch or binary not found in lavavisor directory\n ", err)
		}
		binaryPath, err = handleMissingDir(versionDir, autoDownload, currentVersion)
		if err != nil {
			return "", err
		}
	}
	return binaryPath, nil
}

func downloadAndBuildFromGithub(version, versionDir string) error {
	// Clean up the binary directory if it exists
	err := os.RemoveAll(versionDir)
	if err != nil {
		return utils.LavaFormatError("failed to clean up binary directory", err)
	}
	// URL might need to be updated based on the actual GitHub repository
	url := fmt.Sprintf("https://github.com/lavanet/lava/archive/refs/tags/v%s.zip", version)
	utils.LavaFormatInfo("Fetching the source from: ", utils.Attribute{Key: "URL", Value: url})

	// Send the request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return utils.LavaFormatError("bad HTTP status", nil, utils.Attribute{Key: "status", Value: resp.Status})
	}

	// Prepare the path for downloaded zip
	zipPath := filepath.Join(versionDir, version+".zip")
	// Make sure the directory exists
	dir := filepath.Dir(zipPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o755)
		if err != nil {
			return err
		}
	}

	// Write the body to file
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	// Unzip the source
	_, err = lvutil.Unzip(zipPath, versionDir)
	if err != nil {
		return err
	}
	utils.LavaFormatInfo("Unzipping...")

	// Build the binary
	srcPath := versionDir + "/lava-" + version
	protocolPath := srcPath + "/cmd/lavap"
	utils.LavaFormatInfo("building protocol", utils.Attribute{Key: "protocol-path", Value: protocolPath})

	cmd := exec.Command("go", "build", "-o", "lavap")
	cmd.Dir = protocolPath
	err = cmd.Run()
	if err != nil {
		// try with "cmd/lavad" path again - this is for older versions than v0.22.0
		protocolPath = srcPath + "/cmd/lavad"
		utils.LavaFormatInfo("attempting to building protocol again", utils.Attribute{Key: "protocol-path", Value: protocolPath})
		cmd := exec.Command("go", "build", "-o", "lavap")
		cmd.Dir = protocolPath
		err = cmd.Run()
		if err != nil {
			return utils.LavaFormatError("Unable to build lavap binary", err)
		}
	}

	// Move the binary to binaryPath
	err = os.Rename(filepath.Join(protocolPath, "lavap"), filepath.Join(versionDir, "lavap"))
	if err != nil {
		return utils.LavaFormatError("failed to move compiled binary", err)
	}

	// Verify the compiled binary
	versionDir += "/lavap"
	binaryInfo, err := os.Stat(versionDir)
	if err != nil {
		return utils.LavaFormatError("failed to verify compiled binary", err)
	}
	binaryMode := binaryInfo.Mode()
	if binaryMode.Perm()&0o111 == 0 {
		return utils.LavaFormatError("compiled binary is not executable", nil)
	}
	utils.LavaFormatInfo("lavap binary is successfully verified!")

	// Remove the source files and zip file
	err = os.RemoveAll(srcPath)
	if err != nil {
		return utils.LavaFormatError("failed to remove source files", err)
	}

	err = os.Remove(zipPath)
	if err != nil {
		return utils.LavaFormatError("failed to remove zip file", err)
	}
	utils.LavaFormatInfo("Source and zip files removed from directory.")
	utils.LavaFormatInfo("Auto-download successful.")

	return nil
}
