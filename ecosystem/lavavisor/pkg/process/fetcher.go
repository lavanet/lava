package processmanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	lvutil "github.com/lavanet/lava/v2/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/v2/utils"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
)

type ProtocolBinaryFetcher struct {
	lavavisorPath         string
	CurrentRunningVersion string
	AutoDownload          bool
}

func (pbf *ProtocolBinaryFetcher) SetCurrentRunningVersion(currentVersion string) {
	pbf.CurrentRunningVersion = currentVersion
}

func (pbf *ProtocolBinaryFetcher) SetupLavavisorDir(dir string) error {
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

func (pbf *ProtocolBinaryFetcher) ValidateLavavisorDir(dir string) (lavavisorPath string, err error) {
	lavavisorPath, err = pbf.buildLavavisorPath(dir)
	if err != nil {
		return "", err
	}
	pbf.lavavisorPath = lavavisorPath

	// Validate the existence of ./lavavisor directory
	if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
		return "", utils.LavaFormatError("[Lavavisor] lavavisor directory is not found", err)
	}
	return lavavisorPath, nil
}

func (pbf *ProtocolBinaryFetcher) FetchProtocolBinary(protocolConsensusVersion *protocoltypes.Version) (selectedBinaryPath string, err error) {
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
			if !pbf.AutoDownload {
				return "", utils.LavaFormatError("[Lavavisor] Failed upgrading flow, waiting for directory upgrade directory to be placed in lavavisor config", nil)
			} else {
				return "", utils.LavaFormatError("[Lavavisor] Failed upgrading flow, couldn't fetch the new binary.", nil)
			}
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

func (pbf *ProtocolBinaryFetcher) buildLavavisorPath(dir string) (string, error) {
	dir, err := lvutil.ExpandTilde(dir)
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] unable to expand directory path", err)
	}
	// Build path to ./lavavisor
	return filepath.Join(dir, ".lavavisor"), nil
}

func (pbf *ProtocolBinaryFetcher) setUpLavavisorDirectory() error {
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

func (pbf *ProtocolBinaryFetcher) checkAndHandleVersionDir(versionDir string, protocolConsensusVersion *protocoltypes.Version, currentVersion *lvutil.SemanticVer) (selectedBinaryPath string, err error) {
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

func (pbf *ProtocolBinaryFetcher) dirExists(versionDir string) bool {
	_, err := os.Stat(versionDir)
	return !os.IsNotExist(err)
}

func (pbf *ProtocolBinaryFetcher) handleMissingDir(versionDir string, currentVersion *lvutil.SemanticVer) (binaryPath string, err error) {
	if !pbf.AutoDownload {
		return "", utils.LavaFormatError("[Lavavisor] Sub-directory for version not found and auto-download is disabled.", nil, utils.Attribute{Key: "Version", Value: currentVersion})
	}
	utils.LavaFormatInfo("[Lavavisor] Version directory does not exist, but auto-download is enabled. Attempting to download binary from GitHub...")
	utils.LavaFormatInfo("[Lavavisor] creating directory: " + versionDir)
	errMkdir := os.MkdirAll(versionDir, os.ModePerm)
	if errMkdir != nil {
		return "", utils.LavaFormatError("[Lavavisor] Failed to create binary directory", err)
	}
	utils.LavaFormatInfo("[Lavavisor] created " + versionDir + " successfully")

	utils.LavaFormatInfo("[Lavavisor] Trying to download:", utils.Attribute{Key: "Version", Value: currentVersion})
	if err := pbf.downloadAndBuildFromGithub(lvutil.FormatFromSemanticVersion(currentVersion), versionDir); err == nil {
		binaryPath = filepath.Join(versionDir, "lavap")
		return binaryPath, nil
	}

	// upon failed operation, remove versionDir
	utils.LavaFormatInfo("[Lavavisor] Failed downloading, deleting directory, retrying next block", utils.Attribute{Key: "Version", Value: currentVersion})
	err = os.RemoveAll(versionDir)
	if err != nil {
		return "", err
	}

	return "", utils.LavaFormatError("[Lavavisor] Failed to auto-download binary from GitHub\n ", nil)
}

func (pbf *ProtocolBinaryFetcher) handleExistingDir(versionDir string, protocolConsensusVersion *protocoltypes.Version, currentVersion *lvutil.SemanticVer) (binaryPath string, err error) {
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

func (pbf *ProtocolBinaryFetcher) downloadAndBuildFromGithub(version, versionDir string) error {
	// Clean up the binary directory if it exists
	err := os.RemoveAll(versionDir)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] failed to clean up binary directory", err)
	}
	// URL might need to be updated based on the actual GitHub repository
	url := fmt.Sprintf("https://github.com/lavanet/lava/v2/archive/refs/tags/v%s.zip", version)
	utils.LavaFormatInfo("[Lavavisor] Fetching the source from: ", utils.Attribute{Key: "URL", Value: url})

	// Send the request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return utils.LavaFormatError("[Lavavisor] bad HTTP status", nil, utils.Attribute{Key: "status", Value: resp.Status})
	}

	// Prepare the path for downloaded zip
	zipPath := filepath.Join(versionDir, version+".zip")
	// Make sure the directory exists
	err = pbf.createDirIfNotExist(filepath.Dir(zipPath))
	if err != nil {
		return err
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
	utils.LavaFormatInfo("[Lavavisor] Unzipping...")

	// Verify Go installation
	goCommand, err := pbf.VerifyGoInstallation()
	if err != nil {
		return err
	}

	// Build the binary
	srcPath := versionDir + "/lava-" + version
	protocolPath := srcPath + "/cmd/lavap"
	utils.LavaFormatInfo("[Lavavisor] building protocol", utils.Attribute{Key: "protocol-path", Value: protocolPath})

	cmd := exec.Command(goCommand, "build", "-o", "lavap")
	cmd.Dir = protocolPath
	err = cmd.Run()
	if err != nil {
		// try with "cmd/lavad" path again - this is for older versions than v0.22.0
		protocolPath = srcPath + "/cmd/lavad"
		utils.LavaFormatInfo("[Lavavisor] attempting to building protocol again", utils.Attribute{Key: "protocol-path", Value: protocolPath})
		cmd := exec.Command(goCommand, "build", "-o", "lavap")
		cmd.Dir = protocolPath
		err = cmd.Run()
		if err != nil {
			return utils.LavaFormatError("[Lavavisor] Unable to build lavap binary", err)
		}
	}

	// Move the binary to binaryPath
	err = os.Rename(filepath.Join(protocolPath, "lavap"), filepath.Join(versionDir, "lavap"))
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] failed to move compiled binary", err)
	}

	// Verify the compiled binary
	versionDir += "/lavap"
	binaryInfo, err := os.Stat(versionDir)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] failed to verify compiled binary", err)
	}
	binaryMode := binaryInfo.Mode()
	if binaryMode.Perm()&0o111 == 0 {
		return utils.LavaFormatError("[Lavavisor] compiled binary is not executable", nil)
	}
	utils.LavaFormatInfo("[Lavavisor] lavap binary is successfully verified!")

	// Remove the source files and zip file
	err = os.RemoveAll(srcPath)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] failed to remove source files", err)
	}

	err = os.Remove(zipPath)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] failed to remove zip file", err)
	}
	utils.LavaFormatInfo("[Lavavisor] Source and zip files removed from directory.")
	utils.LavaFormatInfo("[Lavavisor] Auto-download successful.")

	return nil
}

func (pbf *ProtocolBinaryFetcher) createDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o755)
		if err != nil {
			return utils.LavaFormatError("[Lavavisor] Failed creating directory", err, utils.Attribute{Key: "dir", Value: dir})
		}
	}
	return nil
}

func (pbf *ProtocolBinaryFetcher) getInstalledGoVersion(goPath string) (string, error) {
	goVersionRun := exec.Command(goPath, "version")
	goVersion, err := goVersionRun.Output()
	if err != nil {
		return "", utils.LavaFormatInfo("[Lavavisor] Error running go version", utils.Attribute{Key: "err", Value: err}, utils.Attribute{Key: "command", Value: goVersionRun})
	}

	stringGoVersion := string(goVersion)
	splitGoVersion := strings.Split(stringGoVersion, " ") // go version go1.20.5 linux/amd64
	if len(splitGoVersion) < 3 {
		return "", utils.LavaFormatError("[Lavavisor] Unable to parse go version", nil, utils.Attribute{Key: "version", Value: stringGoVersion})
	}

	versionBeforeCut := splitGoVersion[2]
	if !strings.HasPrefix(versionBeforeCut, "go") {
		utils.LavaFormatError("[Lavavisor] Unable to parse go version", nil, utils.Attribute{Key: "version", Value: stringGoVersion})
	}
	version := versionBeforeCut[2:]
	utils.LavaFormatInfo("[Lavavisor] Verified that go is on the right version", utils.Attribute{Key: "version", Value: version})
	return version, nil
}

func (pbf *ProtocolBinaryFetcher) downloadGo(downloadPath string, version string) (string, error) {
	if runtime.GOARCH == "" {
		return "", utils.LavaFormatError("[Lavavisor] Could not determine the machine architecture (runtime.GOARCH is empty). Aborting", nil)
	}

	if runtime.GOOS == "" {
		return "", utils.LavaFormatError("[Lavavisor] Could not determine the machine OS (runtime.GOOS is empty). Aborting", nil)
	}

	url := fmt.Sprintf("https://go.dev/dl/go%s.%s-%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	utils.LavaFormatInfo(fmt.Sprintf("Downloading Go from %s", url))

	resp, err := http.Get(url)
	if err != nil {
		return "", utils.LavaFormatError(fmt.Sprintf("Unable to download Go version %s", version), err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return "", utils.LavaFormatError(fmt.Sprintf("Unable to download Go version %s. Got status code: %d", version, resp.StatusCode), err,
			utils.Attribute{Key: "response", Value: resp})
	}

	// Write the body to file
	goInstallationFilePath := fmt.Sprintf(filepath.Join(downloadPath, "go%s.tar.gz"), version)
	goInstallationFileHandler, err := os.Create(goInstallationFilePath)
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] Unable to create Go installation file", err, utils.Attribute{Key: "filePath", Value: goInstallationFilePath})
	}
	defer goInstallationFileHandler.Close()

	_, err = io.Copy(goInstallationFileHandler, resp.Body)
	if err != nil {
		os.Remove(goInstallationFilePath)
		return "", utils.LavaFormatError("[Lavavisor] Error copying downloaded file data. Deleting file.", err, utils.Attribute{Key: "filePath", Value: goInstallationFilePath})
	}

	utils.LavaFormatInfo("[Lavavisor] Finished downloading go", utils.Attribute{Key: "goInstallationPath", Value: goInstallationFilePath})
	return goInstallationFilePath, nil
}

func (pbf *ProtocolBinaryFetcher) installGo(installPath string, goFilePath string) (string, error) {
	utils.LavaFormatDebug("[Lavavisor] Extracting go files...", utils.Attribute{Key: "installPath", Value: installPath}, utils.Attribute{Key: "goFilePath", Value: goFilePath})
	goInstallCommand := exec.Command("tar", "-C", installPath, "-xzf", goFilePath)
	output, err := goInstallCommand.Output()
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] Unable to install Go", err, utils.Attribute{Key: "command", Value: goInstallCommand}, utils.Attribute{Key: "output", Value: output})
	}
	utils.LavaFormatDebug("[Lavavisor] Finished extracting go")
	return filepath.Join(installPath, "/go/bin/go"), nil
}

func (pbf *ProtocolBinaryFetcher) downloadInstallAndVerifyGo(installPath string, goVersion string) (string, error) {
	goFilePath, err := pbf.downloadGo(installPath, goVersion)
	if err != nil {
		return "", err
	}

	goBinary, err := pbf.installGo(installPath, goFilePath)
	if err != nil {
		return "", err
	}

	installedGoVersion, err := pbf.getInstalledGoVersion(goBinary)
	if err != nil {
		return "", err
	}

	if installedGoVersion != goVersion {
		return "", utils.LavaFormatError("[Lavavisor] Installed Go version does not match expected version", nil,
			utils.Attribute{Key: "expectedVersion", Value: installedGoVersion},
			utils.Attribute{Key: "installedVersion", Value: goVersion})
	}

	return goBinary, nil
}

func (pbf *ProtocolBinaryFetcher) VerifyGoInstallation() (string, error) {
	goCommand := "go"
	emptyGoCommand := ""
	expectedGeVersion := "1.20.5"
	homePath, err := GetHomePath()
	if err != nil {
		return emptyGoCommand, err
	}
	goPath := filepath.Join(homePath, "go") // In case go is not installed

	installedGoVersion, err := pbf.getInstalledGoVersion(goCommand)
	if err != nil {
		utils.LavaFormatInfo("[Lavavisor] Go was not found in PATH")
		potentialGoCommands := []string{
			filepath.Join(homePath, "/go/bin"), // ~/go/bin/go
			"/usr/local/go/bin",
		}

		found := false
		for _, potentialGoCommand := range potentialGoCommands {
			goBin := filepath.Join(potentialGoCommand, "/go")
			utils.LavaFormatInfo(fmt.Sprintf("Attempting %s", goBin))

			installedGoVersion, err = pbf.getInstalledGoVersion(goBin)
			if err == nil {
				utils.LavaFormatInfo(fmt.Sprintf("Found go %s with version %s", goBin, installedGoVersion))
				found = true
				goCommand = goBin
				err := AddGoPathToDollarPath(potentialGoCommand)
				if err == nil {
					return goCommand, nil
				}
				break
			}
		}

		if !found {
			utils.LavaFormatInfo("[Lavavisor] Could not find Go. Installing Go...")

			err = pbf.createDirIfNotExist(goPath)
			if err != nil {
				return "", err
			}

			goCommand, err = pbf.downloadInstallAndVerifyGo(goPath, expectedGeVersion)
			if err != nil {
				return emptyGoCommand, utils.LavaFormatError("[Lavavisor] Unable to download and install Go", err)
			}
			return goCommand, nil
		}
	}

	if installedGoVersion != expectedGeVersion {
		utils.LavaFormatInfo(fmt.Sprintf("Version %s of Go mismatch the desired version. Installing %s", installedGoVersion, expectedGeVersion))
		if _, err := os.Stat(goPath); err != nil {
			err := os.RemoveAll(goPath)
			if err != nil {
				return emptyGoCommand, utils.LavaFormatError("[Lavavisor] Unable to remove existing Go installation", err)
			}
		}

		goCommand, err = pbf.downloadInstallAndVerifyGo(goPath, expectedGeVersion)
		if err != nil {
			return emptyGoCommand, utils.LavaFormatError("[Lavavisor] Unable to download and install Go", err)
		}
	}

	return goCommand, nil
}
