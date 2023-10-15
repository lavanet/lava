package processmanager

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type VersionMonitor struct {
	BinaryPath            string
	LavavisorPath         string
	updateTriggered       chan *protocoltypes.Version
	lastKnownVersion      *protocoltypes.Version
	processes             []string
	autoDownload          bool
	protocolBinaryFetcher *ProtocolBinaryFetcher
	protocolBinaryLinker  *ProtocolBinaryLinker
	lock                  sync.Mutex
}

func NewVersionMonitor(initVersion string, lavavisorPath string, processes []string, autoDownload bool) *VersionMonitor {
	versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+initVersion)
	binaryPath := filepath.Join(versionDir, "lavap")

	return &VersionMonitor{
		BinaryPath:      binaryPath,
		LavavisorPath:   lavavisorPath,
		updateTriggered: make(chan *protocoltypes.Version),
		processes:       processes,
		autoDownload:    autoDownload,
		protocolBinaryFetcher: &ProtocolBinaryFetcher{
			lavavisorPath: lavavisorPath,
		},
		protocolBinaryLinker: &ProtocolBinaryLinker{},
		lock:                 sync.Mutex{},
	}
}

func (vm *VersionMonitor) handleUpdateTrigger(incoming *protocoltypes.Version) {
	// set latest known version to incoming.
	utils.LavaFormatInfo("Update detected. Lavavisor starting the auto-upgrade...")
	vm.lastKnownVersion = incoming

	// 1. check lavavisor directory first and attempt to fetch new binary from there
	versionDir := filepath.Join(vm.LavavisorPath, "upgrades", "v"+vm.lastKnownVersion.ProviderTarget)
	binaryPath := filepath.Join(versionDir, "lavap")
	vm.BinaryPath = binaryPath // updating new binary path for validating new binary

	// fetcher
	_, err := vm.protocolBinaryFetcher.FetchProtocolBinary(vm.autoDownload, vm.lastKnownVersion)
	if err != nil {
		utils.LavaFormatWarning("Lavavisor was not able to fetch updated version. Skipping.", err, utils.Attribute{Key: "Version", Value: vm.lastKnownVersion.ProviderTarget})
		return
	}

	// linker
	err = vm.protocolBinaryLinker.CreateLink(binaryPath)
	if err != nil {
		utils.LavaFormatWarning("Lavavisor was not able to create link to the binaries. Skipping.", err, utils.Attribute{Key: "Version", Value: vm.lastKnownVersion.ProviderTarget})
		return
	}

	lavavisorServicesDir := vm.LavavisorPath + "/services/"
	if _, err := os.Stat(lavavisorServicesDir); os.IsNotExist(err) {
		utils.LavaFormatError("Directory does not exist. Skipping.", nil, utils.Attribute{Key: "lavavisorServicesDir", Value: lavavisorServicesDir})
		return
	}
	for _, process := range vm.processes {
		utils.LavaFormatInfo("Restarting process", utils.Attribute{Key: "Process", Value: process})
		err := StartProcess(process)
		if err != nil {
			utils.LavaFormatError("Failed starting process", err, utils.Attribute{Key: "Process", Value: process})
		}
	}
	utils.LavaFormatInfo("Lavavisor successfully updated protocol version!", utils.Attribute{Key: "Upgraded version:", Value: vm.lastKnownVersion.ProviderTarget})
}

func (vm *VersionMonitor) ValidateProtocolVersion(incoming *protocoltypes.Version) error {
	if !vm.lock.TryLock() { // if an upgrade is currently ongoing we don't need to check versions. just wait for the flow to end.
		utils.LavaFormatDebug("ValidateProtocolVersion is locked, assuming upgrade is ongoing")
		return nil
	}
	defer vm.lock.Unlock()
	currentBinaryVersion, err := GetBinaryVersion(vm.BinaryPath)
	if err != nil || currentBinaryVersion == "" {
		return utils.LavaFormatError("failed to get binary version", err)
	}

	if ValidateMismatch(incoming, currentBinaryVersion) {
		utils.LavaFormatInfo("New version detected", utils.Attribute{Key: "incoming", Value: incoming})
		utils.LavaFormatInfo("Started Version Upgrade flow")
		vm.handleUpdateTrigger(incoming)
		return nil
	}

	// version is ok.
	utils.LavaFormatInfo("Validated protocol version", utils.Attribute{Key: "current binary", Value: currentBinaryVersion})
	return nil
}
