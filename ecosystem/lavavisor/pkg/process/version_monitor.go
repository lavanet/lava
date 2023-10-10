package processmanager

import (
	"context"
	"os"
	"path/filepath"

	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	protocolVersion "github.com/lavanet/lava/protocol/upgrade"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type VersionMonitor struct {
	BinaryPath            string
	LavavisorPath         string
	updateTriggered       chan bool
	mismatchType          lvutil.MismatchType
	lastKnownVersion      *protocoltypes.Version
	processes             []*ServiceProcess
	autoDownload          bool
	protocolBinaryFetcher *ProtocolBinaryFetcher
	protocolBinaryLinker  *ProtocolBinaryLinker
}

func NewVersionMonitor(initVersion string, lavavisorPath string, processes []*ServiceProcess, autoDownload bool) *VersionMonitor {
	versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+initVersion)
	binaryPath := filepath.Join(versionDir, "lavap")

	return &VersionMonitor{
		BinaryPath:      binaryPath,
		LavavisorPath:   lavavisorPath,
		updateTriggered: make(chan bool),
		processes:       processes,
		autoDownload:    autoDownload,
		protocolBinaryFetcher: &ProtocolBinaryFetcher{
			lavavisorPath: lavavisorPath,
		},
		protocolBinaryLinker: &ProtocolBinaryLinker{},
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
				utils.LavaFormatInfo("Update detected. Lavavisor starting the auto-upgrade...")

				// 1. check lavavisor directory first and attempt to fetch new binary from there
				var versionToUpgrade string
				if vm.mismatchType == 1 {
					versionToUpgrade = vm.lastKnownVersion.ProviderMin
				} else if vm.mismatchType == 2 {
					versionToUpgrade = vm.lastKnownVersion.ProviderTarget
				} else {
					utils.LavaFormatWarning("Unknown mismatch type detected in Version Monitor. Skipping.", nil, utils.Attribute{Key: "mismatchType", Value: vm.mismatchType})
					continue
				}
				versionDir := filepath.Join(vm.LavavisorPath, "upgrades", "v"+versionToUpgrade)
				binaryPath := filepath.Join(versionDir, "lavap")
				vm.BinaryPath = binaryPath // updating new binary path for validating new binary

				// fetcher
				_, err := vm.protocolBinaryFetcher.FetchProtocolBinary(vm.autoDownload, vm.lastKnownVersion)
				if err != nil {
					utils.LavaFormatWarning("Lavavisor was not able to fetch updated version. Skipping.", err, utils.Attribute{Key: "Version", Value: versionToUpgrade})
					continue
				}

				// linker
				err = vm.protocolBinaryLinker.CreateLink(binaryPath)
				if err != nil {
					utils.LavaFormatWarning("Lavavisor was not able to create link to the binaries. Skipping.", err, utils.Attribute{Key: "Version", Value: versionToUpgrade})
					continue
				}

				lavavisorServicesDir := vm.LavavisorPath + "/services/"
				if _, err := os.Stat(lavavisorServicesDir); os.IsNotExist(err) {
					utils.LavaFormatError("Directory does not exist. Skipping.", nil, utils.Attribute{Key: "lavavisorServicesDir", Value: lavavisorServicesDir})
					continue
				}
				for _, process := range vm.processes {
					utils.LavaFormatInfo("Restarting process", utils.Attribute{Key: "Process", Value: process.Name})
					serviceDir := lavavisorServicesDir + process.Name
					vm.processes = StartProcess(vm.processes, process.Name, serviceDir)
				}

				utils.LavaFormatInfo("Lavavisor successfully updated protocol version!", utils.Attribute{Key: "Upgraded version:", Value: versionToUpgrade})
			}
		}
	}()
}

func (vm *VersionMonitor) ValidateProtocolVersion(incoming *protocoltypes.Version) error {
	currentBinaryVersion, err := GetBinaryVersion(vm.BinaryPath)
	if err != nil || currentBinaryVersion == "" {
		return utils.LavaFormatError("failed to get binary version", err)
	}

	minVersionMismatch := (protocolVersion.HasVersionMismatch(incoming.ConsumerMin, currentBinaryVersion) || protocolVersion.HasVersionMismatch(incoming.ProviderMin, currentBinaryVersion))
	targetVersionMismatch := (protocolVersion.HasVersionMismatch(incoming.ConsumerTarget, currentBinaryVersion) || protocolVersion.HasVersionMismatch(incoming.ProviderTarget, currentBinaryVersion))

	if minVersionMismatch || targetVersionMismatch {
		if minVersionMismatch {
			vm.mismatchType = lvutil.MinVersionMismatch
		} else {
			vm.mismatchType = lvutil.TargetVersionMismatch
		}
		vm.lastKnownVersion = incoming
		vm.updateTriggered <- true // Trigger new version
		utils.LavaFormatInfo("New version detected", utils.Attribute{Key: "incoming", Value: incoming})
		return nil
	}

	// version is ok.
	utils.LavaFormatInfo("Validated protocol version", utils.Attribute{Key: "current binary", Value: currentBinaryVersion})

	return nil
}
