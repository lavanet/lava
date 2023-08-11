package processmanager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type VersionMonitor struct {
	BinaryPath       string
	LavavisorPath    string
	updateTriggered  chan bool
	mismatchType     lvutil.MismatchType
	lastknownversion *protocoltypes.Version
	providers        []*ProviderProcess
}

func NewVersionMonitor(initVersion string, lavavisorPath string, providers []*ProviderProcess) *VersionMonitor {

	versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+initVersion)
	binaryPath := filepath.Join(versionDir, "lava-protocol")

	return &VersionMonitor{
		BinaryPath:      binaryPath,
		LavavisorPath:   lavavisorPath,
		updateTriggered: make(chan bool),
		providers:       providers,
	}
}

func (vm *VersionMonitor) MonitorVersionUpdates(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-vm.updateTriggered:
				// fetch new version from consensus
				fmt.Println("UPDATE DETECTED. LAVAVISOR STARTING AUTO-UPGRADE...")

				// 1. check lavavisor directory first and attempt to fetch new binary from there
				var versionToUpgrade string
				if vm.mismatchType == 1 {
					versionToUpgrade = vm.lastknownversion.ProviderMin
				} else if vm.mismatchType == 2 {
					versionToUpgrade = vm.lastknownversion.ProviderTarget
				} else {
					utils.LavaFormatFatal("Unknown mismatch type detected in Version Monitor!", nil)
				}
				versionDir := filepath.Join(vm.LavavisorPath, "upgrades", "v"+versionToUpgrade)
				binaryPath := filepath.Join(versionDir, "lava-protocol")

				if _, err := os.Stat(versionDir); os.IsNotExist(err) {
					utils.LavaFormatError("Sub-directory for version not found in lavavisor", nil, utils.Attribute{Key: "version", Value: vm.lastknownversion})
					os.Exit(1)
				}
				vm.BinaryPath = binaryPath // updating new binary path for validating new binary
				err := vm.ValidateProtocolVersion(vm.lastknownversion)
				if err != nil {
					utils.LavaFormatError("Protocol version mismatch or binary not found in lavavisor directory\n ", err)
					os.Exit(1)
				}
				utils.LavaFormatInfo("Updated protocol binary with target version has been successfully set!")

				// 2. check auto-download flag and attempt to download

				// 3. re-create symlink
				// 3- if found: create a link from that binary to $(which lava-protocol)
				out, err := exec.LookPath("lava-protocol")
				if err != nil {
					// if "which" command fails, copy binary to system path
					gobin, err := exec.Command("go", "env", "GOPATH").Output()
					if err != nil {
						utils.LavaFormatFatal("couldn't determine Go binary path", err)
					}

					goBinPath := strings.TrimSpace(string(gobin)) + "/bin/"

					// Check if the fetched binary is executable
					// ToDo: change flag to "--version" once relased binaries support the flag
					_, err = exec.Command(binaryPath, "--help").Output()
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

					err = lvutil.Copy(binaryPath, goBinPath+"lava-protocol")
					if err != nil {
						utils.LavaFormatFatal("couldn't copy binary to system path", err)
					}

					// try "which" command again
					out, err = exec.LookPath("lava-protocol")
					if err != nil {
						utils.LavaFormatFatal("couldn't find the binary in the system path", err)
					} else {
						utils.LavaFormatInfo("Found binary at:", utils.Attribute{Key: "Path", Value: out})
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

				err = os.Symlink(binaryPath, dest)
				if err != nil {
					utils.LavaFormatFatal("couldn't create symbolic link", err)
				}

				// check that the link has been established
				link, err := os.Readlink(dest)
				if err != nil || link != binaryPath {
					utils.LavaFormatFatal("failed to verify symbolic link", err)
				}

				utils.LavaFormatInfo("Symbolic link created successfully.", utils.Attribute{Key: "Linked binary path", Value: dest})

				// 4. restart provider processes

				for _, provider := range vm.providers {
					fmt.Printf("Restarting provider: %s\n", provider.Name)
					vm.providers = StartProvider(vm.providers, provider.Name)
				}

				utils.LavaFormatInfo("Lavavisor successfully updated protocol version!", utils.Attribute{Key: "Upgraded version:", Value: versionToUpgrade})
			}
		}
	}()
}

func (vm *VersionMonitor) ValidateProtocolVersion(incoming *protocoltypes.Version) error {
	binaryVersion, err := getBinaryVersion(vm.BinaryPath)
	if err != nil {
		return utils.LavaFormatError("failed to get binary version", err)
	}
	utils.LavaFormatInfo("Validated protocol version", utils.Attribute{Key: "current binary", Value: binaryVersion})
	// check min version
	if incoming.ConsumerMin != binaryVersion || incoming.ProviderMin != binaryVersion {
		vm.updateTriggered <- true
		vm.mismatchType = lvutil.MinVersionMismatch
		vm.lastknownversion = incoming
		return lvutil.MinVersionMismatchError
	}
	// check target version
	if incoming.ConsumerTarget != binaryVersion || incoming.ProviderTarget != binaryVersion {
		vm.updateTriggered <- true
		vm.mismatchType = lvutil.TargetVersionMismatch
		vm.lastknownversion = incoming
		return lvutil.TargetVersionMismatchError
	}
	// version is ok.
	return nil
}
