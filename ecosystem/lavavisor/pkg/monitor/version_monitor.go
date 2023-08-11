package processmanager

import (
	"archive/zip"
	"context"
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

func getBinaryVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", utils.LavaFormatError("failed to execute command", err)
	}

	// output format is "lava-protocol version x.x.x"
	version := strings.Split(string(output), " ")[2]
	return strings.TrimSpace(version), nil
}

// ToDo: we will implement trigger logic here!
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

func FetchAndBuildFromGithub(version, versionDir string) error {
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
	_, err = unzip(zipPath, versionDir)
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

func unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, utils.LavaFormatError("illegal file path", nil)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
