package processmanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	lvutil "github.com/lavanet/lava/v2/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/v2/utils"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
)

type ProtocolBinaryFetcherWithoutBuild struct {
	lavavisorPath         string
	CurrentRunningVersion string
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) SetCurrentRunningVersion(currentVersion string) {
	pbf.CurrentRunningVersion = currentVersion
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) SetupLavavisorDir(dir string) error {
	lavavisorPath, err := pbf.buildLavavisorPath(dir)
	if err != nil {
		return err
	}
	pbf.lavavisorPath = lavavisorPath
	// Check if ./lavavisor directory exists
	if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
		// If not, create the directory
		err = pbf.setUpLavavisorDirectory()
		if err != nil {
			return utils.LavaFormatError("[Lavavisor] unable to create .lavavisor/ directory", err)
		}
		utils.LavaFormatInfo("[Lavavisor] .lavavisor/ folder successfully created", utils.Attribute{Key: "path:", Value: lavavisorPath})
	}
	return nil
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) ValidateLavavisorDir(dir string) (lavavisorPath string, err error) {
	lavavisorPath, err = pbf.buildLavavisorPath(dir)
	if err != nil {
		return "", err
	}
	pbf.lavavisorPath = lavavisorPath

	// Validate the existence of ./lavavisor directory and creating it if it doesn't exist
	pbf.createDirIfNotExist(lavavisorPath)
	return lavavisorPath, nil
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) FetchProtocolBinary(protocolConsensusVersion *protocoltypes.Version) (selectedBinaryPath string, err error) {
	if pbf.lavavisorPath == "" {
		return "", utils.LavaFormatError("[Lavavisor] The lavavisor path is not initialized. Should not get here!", nil)
	}

	var currentRunningVersion *lvutil.SemanticVer
	if pbf.CurrentRunningVersion != "" {
		currentRunningVersion = lvutil.ParseToSemanticVersion(pbf.CurrentRunningVersion)
	}
	currentVersion := lvutil.ParseToSemanticVersion(protocolConsensusVersion.ProviderTarget)
	minVersion := lvutil.ParseToSemanticVersion(protocolConsensusVersion.ProviderMin) // min(currentRunning, minVersionInParams)

	for ; !lvutil.IsVersionLessThan(currentVersion, minVersion); lvutil.DecrementVersion(currentVersion) {
		if currentRunningVersion != nil && (lvutil.IsVersionEqual(currentRunningVersion, currentVersion) || lvutil.IsVersionGreaterThan(currentRunningVersion, currentVersion)) {
			return "", utils.LavaFormatError("[Lavavisor] Failed upgrading flow, couldn't fetch the new binary.", nil)
		}
		utils.LavaFormatInfo("[Lavavisor] Trying to fetch", utils.Attribute{Key: "version", Value: lvutil.FormatFromSemanticVersion(currentVersion)})
		versionDir := filepath.Join(pbf.lavavisorPath, "upgrades", "v"+lvutil.FormatFromSemanticVersion(currentVersion))
		utils.LavaFormatInfo("[Lavavisor] Version Directory", utils.Attribute{Key: "versionDir", Value: versionDir})
		selectedBinaryPath, err = pbf.checkAndHandleVersionDir(versionDir, protocolConsensusVersion, currentVersion)
		if err == nil {
			return selectedBinaryPath, nil
		}
	}
	return "", utils.LavaFormatError("[Lavavisor] Failed to fetch protocol binary for both target and min versions", nil)
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) buildLavavisorPath(dir string) (string, error) {
	dir, err := lvutil.ExpandTilde(dir)
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] unable to expand directory path", err)
	}
	// Build path to ./lavavisor
	return filepath.Join(dir, ".lavavisor"), nil
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) setUpLavavisorDirectory() error {
	err := os.MkdirAll(pbf.lavavisorPath, 0o755)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] unable to create .lavavisor/ directory", err, utils.Attribute{Key: "lavavisorPath", Value: pbf.lavavisorPath})
	}
	// Create config.yml file inside .lavavisor and write placeholder text
	configPath := filepath.Join(pbf.lavavisorPath, "config.yml")
	configFile, err := os.Create(configPath)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] unable to create or clean config.yml", err)
	}
	defer configFile.Close() // Close the file

	// Create 'upgrades' directory inside .lavavisor
	upgradesPath := filepath.Join(pbf.lavavisorPath, "upgrades")
	if _, err := os.Stat(upgradesPath); os.IsNotExist(err) {
		err = os.MkdirAll(upgradesPath, 0o755)
		if err != nil {
			return utils.LavaFormatError("[Lavavisor] unable to create 'upgrades' directory", err)
		}
	}
	return nil
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) checkAndHandleVersionDir(versionDir string, protocolConsensusVersion *protocoltypes.Version, currentVersion *lvutil.SemanticVer) (selectedBinaryPath string, err error) {
	var binaryPath string
	if pbf.dirExists(versionDir) {
		utils.LavaFormatDebug("[Lavavisor] Handling existing version dir", utils.Attribute{Key: "versionDir", Value: versionDir})
		binaryPath, err = pbf.handleExistingDir(versionDir, protocolConsensusVersion, currentVersion)
		if err != nil {
			return "", err
		}
	} else {
		utils.LavaFormatDebug("[Lavavisor] Handling missing version dir", utils.Attribute{Key: "versionDir", Value: versionDir})
		binaryPath, err = pbf.handleMissingDir(versionDir, currentVersion)
		if err != nil {
			return "", err
		}
	}

	utils.LavaFormatInfo("[Lavavisor] Protocol binary with target version has been successfully set! path: " + binaryPath)
	return binaryPath, nil
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) dirExists(versionDir string) bool {
	_, err := os.Stat(versionDir)
	return !os.IsNotExist(err)
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) handleMissingDir(versionDir string, currentVersion *lvutil.SemanticVer) (binaryPath string, err error) {
	utils.LavaFormatInfo("[Lavavisor] Version directory does not exist, but auto-download is enabled. Attempting to download binary from GitHub...")
	utils.LavaFormatInfo("[Lavavisor] creating directory: " + versionDir)
	errMkdir := os.MkdirAll(versionDir, os.ModePerm)
	if errMkdir != nil {
		return "", utils.LavaFormatError("[Lavavisor] Failed to create binary directory", err)
	}
	utils.LavaFormatInfo("[Lavavisor] created " + versionDir + " successfully")

	utils.LavaFormatInfo("[Lavavisor] Trying to download:", utils.Attribute{Key: "Version", Value: currentVersion})
	if err := pbf.downloadBinaryFromGithub(lvutil.FormatFromSemanticVersion(currentVersion), versionDir); err == nil {
		binaryPath = filepath.Join(versionDir, "lavap")
		return binaryPath, nil
	}

	// upon failed operation, remove versionDir
	utils.LavaFormatFatal("[Lavavisor] Failed downloading, deleting directory, retrying next block", err, utils.Attribute{Key: "Version", Value: currentVersion})
	err = os.RemoveAll(versionDir)
	if err != nil {
		return "", err
	}

	return "", utils.LavaFormatError("[Lavavisor] Failed to auto-download binary from GitHub", nil)
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) downloadBinaryFromGithub(version, versionDir string) error {
	// Clean up the binary directory if it exists
	err := os.RemoveAll(versionDir)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] failed to clean up binary directory", err)
	}
	// URL might need to be updated based on the actual GitHub repository
	url := fmt.Sprintf("https://github.com/lavanet/lava/v2/releases/download/v%s/lavap-v%s-linux-amd64", version, version)
	utils.LavaFormatInfo("[Lavavisor] Fetching the source from: ", utils.Attribute{Key: "URL", Value: url})
	// Send the request
	resp, err := http.Get(url)
	if err != nil {
		utils.LavaFormatError("[Lavavisor] Failed getting binary", err, utils.Attribute{Key: "URL", Value: url})
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return utils.LavaFormatError("[Lavavisor] bad HTTP status", nil, utils.Attribute{Key: "status", Value: resp.Status})
	}

	// Make sure the directory exists
	utils.LavaFormatInfo("[Lavavisor] Creating directory", utils.Attribute{Key: "path", Value: versionDir})
	err = os.MkdirAll(versionDir, 0o755)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] Failed creating directory", err, utils.Attribute{Key: "dir", Value: versionDir})
	}

	// Prepare the path for downloaded zip
	lavapPath := filepath.Join(versionDir, "lavap")
	utils.LavaFormatInfo("[Lavavisor] Creating file", utils.Attribute{Key: "path", Value: lavapPath})
	out, err := os.Create(lavapPath)
	if err != nil {
		utils.LavaFormatError("[Lavavisor] Failed os.Create", err, utils.Attribute{Key: "path", Value: lavapPath})
		return err
	}
	defer out.Close()

	// Write the body to file
	utils.LavaFormatInfo("[Lavavisor] Creating binary", utils.Attribute{Key: "path", Value: lavapPath})
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	utils.LavaFormatInfo("[Lavavisor] Validating binary", utils.Attribute{Key: "path", Value: lavapPath})
	binaryInfo, err := os.Stat(lavapPath)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] failed to verify compiled binary", err)
	}
	binaryMode := binaryInfo.Mode()
	if binaryMode.Perm()&0o111 == 0 {
		err = os.Chmod(lavapPath, binaryMode|os.FileMode(0o111))
		if err != nil {
			return utils.LavaFormatError("[Lavavisor] failed to make the binary executable", err)
		}
	}
	utils.LavaFormatInfo("[Lavavisor] lavap binary is successfully verified!")
	utils.LavaFormatInfo("[Lavavisor] download successful.")
	return nil
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) createDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o755)
		if err != nil {
			return utils.LavaFormatError("[Lavavisor] Failed creating directory", err, utils.Attribute{Key: "dir", Value: dir})
		}
	}
	return nil
}

func (pbf *ProtocolBinaryFetcherWithoutBuild) handleExistingDir(versionDir string, protocolConsensusVersion *protocoltypes.Version, currentVersion *lvutil.SemanticVer) (binaryPath string, err error) {
	binaryPath = filepath.Join(versionDir, "lavap")
	version, _ := GetBinaryVersion(binaryPath)
	if version != "" {
		utils.LavaFormatInfo("found requested version", utils.Attribute{Key: "version", Value: version})
		return binaryPath, nil // found version.
	}
	binaryPath, err = pbf.handleMissingDir(versionDir, currentVersion)
	if err != nil {
		return "", err
	}
	return binaryPath, nil
}
