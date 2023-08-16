package processmanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

func FetchProtocolBinary(lavavisorPath string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version) (selectedBinaryPath string, err error) {
	versions := []string{protocolConsensusVersion.ProviderTarget, protocolConsensusVersion.ProviderMin}

	for _, version := range versions {
		utils.LavaFormatInfo("Trying to fetch", utils.Attribute{Key: "version", Value: version})
		versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+version)
		selectedBinaryPath, err = checkAndHandleVersionDir(versionDir, autoDownload, protocolConsensusVersion)
		if err == nil {
			return selectedBinaryPath, nil
		}
	}

	return "", utils.LavaFormatError("Failed to fetch protocol binary for both target and min versions", nil)
}

func GetLavavisorPath(dir string) (lavavisorPath string, err error) {
	dir, err = lvutil.ExpandTilde(dir)
	if err != nil {
		return "", utils.LavaFormatError("unable to expand directory path", err)
	}
	// Build path to ./lavavisor
	lavavisorPath = filepath.Join(dir, "./.lavavisor")

	// Check if ./lavavisor directory exists
	if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
		// If not, create the directory
		err = setUpLavavisorDirectory(lavavisorPath)
		if err != nil {
			return "", utils.LavaFormatError("unable to create .lavavisor/ directory", err)
		}
		utils.LavaFormatInfo(".lavavisor/ folder successfully created", utils.Attribute{Key: "path:", Value: lavavisorPath})
	}
	return lavavisorPath, nil
}

func setUpLavavisorDirectory(lavavisorPath string) error {
	err := os.MkdirAll(lavavisorPath, 0755)
	if err != nil {
		return utils.LavaFormatError("unable to create .lavavisor/ directory", err)
	}
	// Create config.yml file inside .lavavisor and write placeholder text
	configPath := filepath.Join(lavavisorPath, "config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		sampleServices := []string{
			"consumer-ETH1",
			"provider1-ETH1",
			"provider1-LAV1",
		}
		placeholderText := "services:\n  - " + strings.Join(sampleServices, "\n  - ")
		err = os.WriteFile(configPath, []byte(placeholderText), 0644)
		if err != nil {
			return utils.LavaFormatError("unable to write to config.yml file", err)
		}
	}
	// Create 'upgrades' directory inside .lavavisor
	upgradesPath := filepath.Join(lavavisorPath, "upgrades")
	if _, err := os.Stat(upgradesPath); os.IsNotExist(err) {
		err = os.MkdirAll(upgradesPath, 0755)
		if err != nil {
			return utils.LavaFormatError("unable to create 'upgrades' directory", err)
		}
	}
	return nil
}

func checkAndHandleVersionDir(versionDir string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version) (selectedBinaryPath string, err error) {
	binaryPath := filepath.Join(versionDir, "lava-protocol")

	if dirExists(versionDir) {
		err := handleExistingDir(versionDir, autoDownload, protocolConsensusVersion, binaryPath)
		if err != nil {
			return "", err
		}
	} else {
		err := handleMissingDir(versionDir, autoDownload, protocolConsensusVersion)
		if err != nil {
			return "", err
		}
	}
	utils.LavaFormatInfo("Protocol binary with target version has been successfully set!")
	return binaryPath, nil
}

func dirExists(versionDir string) bool {
	_, err := os.Stat(versionDir)
	return !os.IsNotExist(err)
}

func handleMissingDir(versionDir string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version) error {
	if !autoDownload {
		return utils.LavaFormatError("Sub-directory for version not found and auto-download is disabled.", nil, utils.Attribute{Key: "Version", Value: protocolConsensusVersion.ProviderMin})
	}
	utils.LavaFormatInfo("Version directory does not exist, but auto-download is enabled. Attempting to download binary from GitHub...")
	os.MkdirAll(versionDir, os.ModePerm)
	if err := downloadAndBuildFromGithub(protocolConsensusVersion.ProviderMin, versionDir); err != nil {
		return utils.LavaFormatError("Failed to auto-download binary from GitHub\n ", err)
	}
	// ToDo: add ValidateProtocolVersion check here after 'version' command available in protocol binary release in github
	return nil
}

func handleExistingDir(versionDir string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version, binaryPath string) error {
	vm := VersionMonitor{
		BinaryPath: binaryPath,
	}
	if err := vm.ValidateProtocolVersion(protocolConsensusVersion); err != nil {
		err = handleErrorOnValidation(err, versionDir, autoDownload, protocolConsensusVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleErrorOnValidation(err error, versionDir string, autoDownload bool, protocolConsensusVersion *protocoltypes.Version) error {
	if !autoDownload {
		return utils.LavaFormatError("Protocol version mismatch or binary not found in lavavisor directory\n ", err)
	}
	utils.LavaFormatInfo("Version mismatch or binary not found, but auto-download is enabled. Attempting to download binary from GitHub...")
	if err := downloadAndBuildFromGithub(protocolConsensusVersion.ProviderMin, versionDir); err != nil {
		return utils.LavaFormatError("Failed to auto-download binary from GitHub\n ", err)
	}
	return nil
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
	protocolPath := srcPath + "/protocol"
	utils.LavaFormatInfo("building protocol", utils.Attribute{Key: "protocol-path", Value: protocolPath})

	cmd := exec.Command("go", "build", "-o", "lava-protocol")
	cmd.Dir = protocolPath
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Move the binary to binaryPath
	err = os.Rename(filepath.Join(protocolPath, "lava-protocol"), filepath.Join(versionDir, "lava-protocol"))
	if err != nil {
		return utils.LavaFormatError("failed to move compiled binary", err)
	}

	// Verify the compiled binary
	versionDir += "/lava-protocol"
	binaryInfo, err := os.Stat(versionDir)
	if err != nil {
		return utils.LavaFormatError("failed to verify compiled binary", err)
	}
	binaryMode := binaryInfo.Mode()
	if binaryMode.Perm()&0o111 == 0 {
		return utils.LavaFormatError("compiled binary is not executable", nil)
	}
	utils.LavaFormatInfo("lava-protocol binary is successfully verified!")

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
