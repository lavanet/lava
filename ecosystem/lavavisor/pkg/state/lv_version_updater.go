package lvstatetracker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	versionmontior "github.com/lavanet/lava/ecosystem/lavavisor/pkg/monitor"
	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

const (
	CallbackKeyForVersionUpdate = "version-update"
)

type VersionStateQuery interface {
	GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error)
}

type ProviderListener interface {
	GetProviders() []*lvutil.ProviderProcess
}

type VersionUpdater struct {
	lock              sync.RWMutex
	eventTracker      *EventTracker
	versionStateQuery VersionStateQuery
	lastKnownVersion  *protocoltypes.Version
	lavavisorPath     string
	currentBinary     string
	autoDownload      bool
	providers         ProviderListener
}

func NewVersionUpdater(versionStateQuery VersionStateQuery, eventTracker *EventTracker, version *protocoltypes.Version, lavavisorPath string, currentBinary string, autoDownload bool, providers ProviderListener) *VersionUpdater {
	return &VersionUpdater{versionStateQuery: versionStateQuery, eventTracker: eventTracker, lastKnownVersion: version, lavavisorPath: lavavisorPath, currentBinary: currentBinary, autoDownload: autoDownload, providers: providers}
}

func (vu *VersionUpdater) UpdaterKey() string {
	return CallbackKeyForVersionUpdate
}

func (vu *VersionUpdater) RegisterVersionUpdatable() {
	vu.lock.RLock()
	defer vu.lock.RUnlock()
	err := versionmontior.ValidateProtocolBinaryVersion(vu.lastKnownVersion, vu.currentBinary)
	if err != nil {
		utils.LavaFormatError("Protocol Version Error", err)
	}
}

func (vu *VersionUpdater) Update(latestBlock int64) {
	vu.lock.Lock()
	defer vu.lock.Unlock()
	versionUpdated := vu.eventTracker.getLatestVersionEvents()
	if versionUpdated {
		// fetch updated version from consensus
		version, err := vu.versionStateQuery.GetProtocolVersion(context.Background())
		if err != nil {
			utils.LavaFormatError("could not get version when updated, did not update protocol version and needed to", err)
			return
		}
		utils.LavaFormatInfo("Protocol version has been fetched successfully!",
			utils.Attribute{Key: "old_version", Value: vu.lastKnownVersion},
			utils.Attribute{Key: "new_version", Value: version})
		// if no error, set the last known version.
		vu.lastKnownVersion = version
	}
	// check version on each new block
	// if there is no version upgrades, we expect this check to pass
	// if mismatch detected, lavavisor will start upgrade
	var mismatchType error
	err := versionmontior.ValidateProtocolBinaryVersion(vu.lastKnownVersion, vu.currentBinary)
	if err != nil {
		// 1. detect min or target version mismatch
		var versionToFetch string
		mismatchType = err // set mismatch type (min or target)
		switch err {
		case lvutil.MinVersionMismatchError:
			versionToFetch = vu.lastKnownVersion.ProviderMin
		case lvutil.TargetVersionMismatchError:
			versionToFetch = vu.lastKnownVersion.ProviderTarget
		default:
			utils.LavaFormatError("Unexpected error during version validation", err)
		}

		utils.LavaFormatInfo("Lavavisor detected a version upgrade. Initiating the fetching process...")

		// set the new binary path that lavavisor
		versionDir := filepath.Join(vu.lavavisorPath, "upgrades", "v"+vu.lastKnownVersion.ProviderMin)
		vu.currentBinary = filepath.Join(versionDir, "lava-protocol")
		// check if version directory exists
		if _, err := os.Stat(versionDir); os.IsNotExist(err) {
			if vu.autoDownload {
				utils.LavaFormatInfo("Version directory does not exist, but auto-download is enabled. Attempting to download binary from GitHub...")
				os.MkdirAll(versionDir, os.ModePerm) // before downloading, ensure version directory exists
				err = versionmontior.FetchAndBuildFromGithub(versionToFetch, versionDir)
				if err != nil {
					utils.LavaFormatError("Failed to auto-download binary from GitHub\n ", err)
					os.Exit(1)
				}
			} else {
				utils.LavaFormatError("Sub-directory for version not found in lavavisor", nil, utils.Attribute{Key: "version", Value: versionToFetch})
				os.Exit(1)
			}
			// ToDo: add checkLavaProtocolVersion after version flag is added to release
			//
		} else {
			err = versionmontior.ValidateProtocolBinaryVersion(vu.lastKnownVersion, vu.currentBinary) // validate with newly set binary
			if err != nil {
				if vu.autoDownload {
					utils.LavaFormatInfo("Version mismatch or binary not found, but auto-download is enabled. Attempting to download binary from GitHub...")
					err = versionmontior.FetchAndBuildFromGithub(versionToFetch, versionDir)
					if err != nil {
						utils.LavaFormatError("Failed to auto-download binary from GitHub\n ", err)
						os.Exit(1)
					}
				} else {
					utils.LavaFormatError("Protocol version mismatch or binary not found in lavavisor directory\n ", err)
					os.Exit(1)
				}
				// ToDo: add checkLavaProtocolVersion after version flag is added to release
				//
			}
		}
	}
	utils.LavaFormatInfo("Protocol binary with target version has been successfully set!")

	// re-create link, reboot the protocol processes
	if versionUpdated {
		if mismatchType == lvutil.MinVersionMismatchError {
			vu.stopProviders() // make sure all provider process is stopped
		}
		vu.updateBinaryLink()
		providerList := vu.providers.GetProviders()
		for _, provider := range providerList {
			fmt.Printf("Reboting provider: %s\n", provider.Name)
			lvutil.StartProvider(&providerList, provider.Name)
		}
	}
}

func (vu *VersionUpdater) stopProviders() {
	for _, provider := range vu.providers.GetProviders() {
		if provider.IsRunning {
			cmd := exec.Command("sudo", "systemctl", "stop", provider.Name+".service")
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Failed to stop provider: %s, Error: %s\n", cmd, err)
				fmt.Printf("Command Output: \n%s\n", output)
			} else {
				fmt.Printf("Successfully stopped provider: %s\n", cmd)
				fmt.Printf("Command Output: \n%s\n", output)
			}
		}
	}
}

func (vu *VersionUpdater) updateBinaryLink() {
	// 3- if found: create a link from that binary to $(which lava-protocol)
	out, err := exec.Command("which", "lava-protocol").Output()
	if err != nil {
		// if "which" command fails, copy binary to system path
		gobin, err := exec.Command("go", "env", "GOPATH").Output()
		if err != nil {
			utils.LavaFormatFatal("couldn't determine Go binary path", err)
		}

		goBinPath := strings.TrimSpace(string(gobin)) + "/bin/"

		// Check if the fetched binary is executable
		// ToDo: change flag to "--version" once relased binaries support the flag
		_, err = exec.Command(vu.currentBinary, "--help").Output()
		if err != nil {
			utils.LavaFormatFatal("binary is not a valid executable: ", err)
		}

		// Check if the link already exists and remove it
		lavaLinkPath := goBinPath + "lava-protocol"
		if _, err := os.Lstat(lavaLinkPath); err == nil {
			utils.LavaFormatInfo("Discovered an existing link. Attempting to refresh.")
			err = os.Remove(lavaLinkPath)
			if err != nil {
				utils.LavaFormatFatal("couldn't remove existing link", err)
			}
		} else if !os.IsNotExist(err) {
			// other error
			utils.LavaFormatFatal("unexpected error when checking for existing link", err)
		}
		utils.LavaFormatInfo("Old binary link successfully removed. Attempting to create the updated link.")

		err = lvutil.Copy(vu.currentBinary, goBinPath+"lava-protocol")
		if err != nil {
			utils.LavaFormatFatal("couldn't copy binary to system path", err)
		}

		// try "which" command again
		out, err = exec.Command("which", "lava-protocol").Output()
		if err != nil {
			utils.LavaFormatFatal("couldn't extract binary at the system path", err)
		}
	}
	dest := strings.TrimSpace(string(out))

	if _, err := os.Lstat(dest); err == nil {
		// if destination file exists, remove it
		err = os.Remove(dest)
		if err != nil {
			utils.LavaFormatFatal("couldn't remove existing link", err)
		}
	}

	err = os.Symlink(vu.currentBinary, dest)
	if err != nil {
		utils.LavaFormatFatal("couldn't create symbolic link", err)
	}

	// check that the link has been established
	link, err := os.Readlink(dest)
	if err != nil || link != vu.currentBinary {
		utils.LavaFormatFatal("failed to verify symbolic link", err)
	}

	utils.LavaFormatInfo("Symbolic link created successfully")
}
