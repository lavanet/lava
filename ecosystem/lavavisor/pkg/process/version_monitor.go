package processmanager

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v2/utils"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
)

type ProtocolBinaryFetcherInf interface {
	SetCurrentRunningVersion(currentVersion string)
	FetchProtocolBinary(protocolConsensusVersion *protocoltypes.Version) (selectedBinaryPath string, err error)
}

type KeyRingPassword struct {
	Password   bool   // whether the password is required
	Passphrase string // the password
}

type VersionMonitor struct {
	BinaryPath            string
	LavavisorPath         string
	lastKnownVersion      *protocoltypes.Version
	processes             []string
	autoDownload          bool
	protocolBinaryFetcher ProtocolBinaryFetcherInf
	protocolBinaryLinker  *ProtocolBinaryLinker
	allowNilLinker        bool
	lock                  sync.Mutex
	restart               chan struct{}
	isWrapProcess         bool
	LaunchedServices      bool // indicates whether version was matching or not so we can decide wether to launch services
	onGoingCmd            *exec.Cmd
	command               []string
}

func NewVersionMonitor(initVersion string, lavavisorPath string, processes []string, autoDownload bool) *VersionMonitor {
	var binaryPath string
	if initVersion != "" { // handle case if not found valid lavap at all
		versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+initVersion)
		binaryPath = filepath.Join(versionDir, "lavap")
	}
	fetcher := &ProtocolBinaryFetcher{
		lavavisorPath: lavavisorPath,
		AutoDownload:  autoDownload,
	}
	return &VersionMonitor{
		BinaryPath:            binaryPath,
		LavavisorPath:         lavavisorPath,
		processes:             processes,
		autoDownload:          autoDownload,
		protocolBinaryFetcher: fetcher,
		protocolBinaryLinker:  &ProtocolBinaryLinker{Fetcher: fetcher},
		lock:                  sync.Mutex{},
	}
}

func (vm *VersionMonitor) handleUpdateTrigger(currentBinaryVersion string) error {
	// set latest known version to incoming.
	utils.LavaFormatInfo("[Lavavisor] Update detected. Lavavisor starting the auto-upgrade...")
	vm.protocolBinaryFetcher.SetCurrentRunningVersion(currentBinaryVersion)
	// 1. check lavavisor directory first and attempt to fetch new binary from there

	if vm.protocolBinaryFetcher == nil {
		utils.LavaFormatError("[Lavavisor] for some reason the vm.protocolBinaryFetcher is nil", nil)
	}
	// fetcher
	_, err := vm.protocolBinaryFetcher.FetchProtocolBinary(vm.lastKnownVersion)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] Lavavisor was not able to fetch updated version. Skipping.", err, utils.Attribute{Key: "Version", Value: vm.lastKnownVersion.ProviderTarget})
	}
	versionDir := filepath.Join(vm.LavavisorPath, "upgrades", "v"+vm.lastKnownVersion.ProviderTarget)
	binaryPath := filepath.Join(versionDir, "lavap")
	vm.BinaryPath = binaryPath // updating new binary path for validating new binary

	err = vm.createLink()
	if err != nil {
		return err
	}
	return vm.TriggerRestartProcess()
}

// create link to the golang go env path of "lavap"
func (vm *VersionMonitor) createLink() error {
	if vm.protocolBinaryLinker == nil { // this happens in the pods flow where we don't link and don't validate golang installations
		if vm.allowNilLinker {
			return nil
		} else {
			utils.LavaFormatFatal("vm.protocolBinaryLinker is nil and its not allowed", nil)
		}
	}
	err := vm.protocolBinaryLinker.CreateLink(vm.BinaryPath)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] Lavavisor was not able to create link to the binaries. Skipping.", err, utils.Attribute{Key: "Version", Value: vm.lastKnownVersion.ProviderTarget})
	}
	return nil
}

// create a link for lavap from the binary path and restart the services
func (vm *VersionMonitor) TriggerRestartProcess() error {
	if vm.isWrapProcess {
		if vm.onGoingCmd != nil {
			utils.LavaFormatInfo("[Lavavisor] triggering vm.restart")
			vm.restart <- struct{}{}
			utils.LavaFormatInfo("[Lavavisor] done vm.restart")
			return nil
		} else {
			return nil
		}
	}
	lavavisorServicesDir := vm.LavavisorPath + "/services/"
	if _, err := os.Stat(lavavisorServicesDir); os.IsNotExist(err) {
		return utils.LavaFormatError("[Lavavisor] Directory does not exist. Skipping.", nil, utils.Attribute{Key: "lavavisorServicesDir", Value: lavavisorServicesDir})
	}

	// First reload the daemon.
	err := ReloadDaemon()
	if err != nil {
		utils.LavaFormatError("[Lavavisor] Failed reloading daemon", err)
	}

	// now start all services
	var wg sync.WaitGroup
	for _, process := range vm.processes {
		wg.Add(1)
		go func(process string) {
			defer wg.Done() // Decrement the WaitGroup when done
			utils.LavaFormatInfo("[Lavavisor] Restarting process", utils.Attribute{Key: "Process", Value: process})
			err := StartProcess(process)
			if err != nil {
				utils.LavaFormatError("[Lavavisor] Failed starting process", err, utils.Attribute{Key: "Process", Value: process})
			}
			utils.LavaFormatInfo("[Lavavisor] Finished restarting process successfully", utils.Attribute{Key: "Process", Value: process})
		}(process)
	}
	// Wait for all Goroutines to finish
	wg.Wait()
	vm.LaunchedServices = true
	utils.LavaFormatInfo("[Lavavisor] Lavavisor successfully updated protocol version!", utils.Attribute{Key: "Upgraded version:", Value: vm.lastKnownVersion.ProviderTarget})
	return nil
}

func (vm *VersionMonitor) validateLinkPointsToTheRightTarget() error {
	if vm.protocolBinaryLinker == nil { // this happens in the pods flow where we don't link and don't validate golang installations
		if vm.allowNilLinker {
			return nil
		} else {
			utils.LavaFormatFatal("vm.protocolBinaryLinker is nil and its not allowed", nil)
		}
	}
	lavapPath, err := vm.protocolBinaryLinker.FindLavaProtocolPath(vm.BinaryPath)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] Failed searching for lavap path, failed to validate link exists for lavap latest version", err)
	}
	var createLink bool
	_, err = os.Stat(lavapPath)
	if err != nil {
		// failed to validate lavap path.
		utils.LavaFormatDebug("[Lavavisor] lavap link path was not found attempting to create link", utils.Attribute{Key: "lavap_path", Value: lavapPath})
		createLink = true
	} else {
		utils.LavaFormatDebug("[Lavavisor] lavap link path found, validating linked binary is the right one", utils.Attribute{Key: "lavap_path", Value: lavapPath})
		// read link
		targetPath, err := os.Readlink(lavapPath)
		if err != nil {
			utils.LavaFormatInfo("[Lavavisor] failed reading link from lavap path", utils.Attribute{Key: "error", Value: err})
			createLink = true
		} else if targetPath != vm.BinaryPath {
			// utils.LavaFormatDebug("[Lavavisor] target validation", utils.Attribute{Key: "targetPath", Value: targetPath})
			utils.LavaFormatInfo("[Lavavisor] lavap link was pointing to the wrong binary. removing and creating a new link")
			createLink = true
		}
	}
	if createLink {
		utils.LavaFormatInfo("[Lavavisor] Attempting new link creation for lavap path", utils.Attribute{Key: "lavap", Value: lavapPath}, utils.Attribute{Key: "binary path", Value: vm.BinaryPath})
		err = vm.createLink()
		if err != nil {
			return err
		}
		err = vm.TriggerRestartProcess()
	}
	return err
}

func (vm *VersionMonitor) ValidateProtocolVersion(incoming *updaters.ProtocolVersionResponse) error {
	if !vm.lock.TryLock() { // if an upgrade is currently ongoing we don't need to check versions. just wait for the flow to end.
		utils.LavaFormatDebug("[Lavavisor] ValidateProtocolVersion is locked, assuming upgrade is ongoing")
		return nil
	}
	defer vm.lock.Unlock()
	currentBinaryVersion, _ := GetBinaryVersion(vm.BinaryPath)
	vm.lastKnownVersion = incoming.Version

	if currentBinaryVersion == "" || ValidateMismatch(incoming.Version, currentBinaryVersion) {
		utils.LavaFormatInfo("[Lavavisor] New version detected", utils.Attribute{Key: "incoming", Value: incoming})
		utils.LavaFormatInfo("[Lavavisor] Current Running Version", utils.Attribute{Key: "currentBinaryVersion", Value: currentBinaryVersion})
		utils.LavaFormatInfo("[Lavavisor] Started Version Upgrade flow")
		err := vm.handleUpdateTrigger(currentBinaryVersion)
		if err != nil {
			utils.LavaFormatInfo("[Lavavisor] protocol update failed, lavavisor will continue trying to upgrade version every block until it succeeds")
		}
		return err
	} else if currentBinaryVersion != "" { // in case we have the latest version already installed we need to validate a few things.
		err := vm.validateLinkPointsToTheRightTarget()
		if err != nil {
			return utils.LavaFormatError("[Lavavisor] Failed to validateLinkPointsToTheRightTarget", err)
		}
	}

	// version is ok.
	utils.LavaFormatInfo("[Lavavisor] Validated protocol version",
		utils.Attribute{Key: "current_binary", Value: currentBinaryVersion},
		utils.Attribute{Key: "version_min", Value: incoming.Version.ProviderMin},
		utils.Attribute{Key: "version_target", Value: incoming.Version.ProviderTarget},
		utils.Attribute{Key: "lava_block_number", Value: incoming.BlockNumber})
	return nil
}

func (vm *VersionMonitor) StartProcess(keyringPassword *KeyRingPassword) {
	// Create a channel to capture OS signals (e.g., Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start subprocess in a Goroutine
	go vm.startSubprocess(keyringPassword)

	// Main loop for monitoring the subprocess
	for {
		select {
		case <-vm.restart:
			utils.LavaFormatInfo("[Lavavisor] Received restart signal, restarting subprocess...")
			vm.StopSubprocess()
			go vm.startSubprocess(keyringPassword)
		case sig := <-sigCh:
			utils.LavaFormatInfo("[Lavavisor] Received signal:", utils.Attribute{Key: "signal", Value: sig})
			vm.StopSubprocess()
			return
		}
	}
}

func (vm *VersionMonitor) StopSubprocess() {
	if vm.onGoingCmd != nil && vm.onGoingCmd.Process != nil {
		utils.LavaFormatInfo("[Lavavisor] Stopping old subprocess...")
		if err := vm.onGoingCmd.Process.Kill(); err != nil {
			utils.LavaFormatError("[Lavavisor] Error stopping subprocess", err)
		}
	}
}

func (vm *VersionMonitor) startSubprocess(keyringPassword *KeyRingPassword) {
	// make sure the subprocess wont continue running if we run into a panic.
	defer func() {
		if r := recover(); r != nil {
			utils.LavaFormatError("[Lavavisor] Panic occurred: ", nil)
			vm.StopSubprocess()
		}
	}()

	utils.LavaFormatInfo("[Lavavisor] Starting subprocess...")
	vm.onGoingCmd = exec.Command(vm.BinaryPath, vm.command...)

	// Set up output redirection so you can see the subprocess's output
	vm.onGoingCmd.Stdout = os.Stdout
	// vm.onGoingCmd.Stderr = os.Stderr

	foundPasswordTrigger := make(chan struct{})
	processStart := common.ProcessStartLogText
	stderrPipe, err := vm.onGoingCmd.StderrPipe()
	if err != nil {
		utils.LavaFormatError("[Lavavisor] Error obtaining stderr pipe:", err)
		return
	}

	// Interaction with the command's Stdin
	stdin, err := vm.onGoingCmd.StdinPipe()
	if err != nil {
		fmt.Println("Error obtaining stdin:", err)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if strings.Contains(line, processStart) {
				foundPasswordTrigger <- struct{}{}
			}
		}
	}()

	if err := vm.onGoingCmd.Start(); err != nil {
		utils.LavaFormatError("[Lavavisor] Error starting subprocess:", err)
	}

	<-foundPasswordTrigger
	// wait to make sure process is waiting for password
	time.Sleep(time.Second * 3)

	if keyringPassword != nil && keyringPassword.Password {
		// Send input to the command
		_, err = stdin.Write([]byte(keyringPassword.Passphrase + "\n"))
		if err != nil {
			fmt.Println("Error writing to stdin:", err)
			return
		}
		utils.LavaFormatInfo("[Lavavisor] entered keyring-os password.")
	}
	stdin.Close() // Flush the input stream (this sends the input to the process)

	if err := vm.onGoingCmd.Wait(); err != nil {
		if strings.Contains(err.Error(), "signal: killed") {
			utils.LavaFormatInfo("[Lavavisor] Subprocess stopped due to sig killed.")
		} else {
			utils.LavaFormatError("[Lavavisor] Subprocess exited with error", err)
		}
	} else {
		utils.LavaFormatInfo("[Lavavisor] Subprocess exited without error.")
	}
}

func NewVersionMonitorProcessWrapFlow(initVersion string, lavavisorPath string, autoDownload bool, command string) *VersionMonitor {
	var binaryPath string
	if initVersion != "" { // handle case if not found valid lavap at all
		versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+initVersion)
		binaryPath = filepath.Join(versionDir, "lavap")
	}
	fetcher := &ProtocolBinaryFetcher{
		lavavisorPath: lavavisorPath,
		AutoDownload:  autoDownload,
	}

	// Check if the string starts with "lavap"
	command = strings.TrimPrefix(command, "lavap ")
	return &VersionMonitor{
		BinaryPath:            binaryPath,
		LavavisorPath:         lavavisorPath,
		autoDownload:          autoDownload,
		protocolBinaryFetcher: fetcher,
		protocolBinaryLinker:  &ProtocolBinaryLinker{Fetcher: fetcher},
		lock:                  sync.Mutex{},
		isWrapProcess:         true,
		restart:               make(chan struct{}),
		command:               strings.Fields(command),
	}
}

func NewVersionMonitorProcessPodFlow(initVersion string, lavavisorPath string, command string) *VersionMonitor {
	var binaryPath string
	if initVersion != "" { // handle case if not found valid lavap at all
		versionDir := filepath.Join(lavavisorPath, "upgrades", "v"+initVersion)
		binaryPath = filepath.Join(versionDir, "lavap")
	}
	fetcher := &ProtocolBinaryFetcherWithoutBuild{
		lavavisorPath: lavavisorPath,
	}

	// Check if the string starts with "lavap"
	command = strings.TrimPrefix(command, "lavap ")
	return &VersionMonitor{
		BinaryPath:            binaryPath,
		LavavisorPath:         lavavisorPath,
		protocolBinaryFetcher: fetcher,
		allowNilLinker:        true,
		lock:                  sync.Mutex{},
		isWrapProcess:         true,
		restart:               make(chan struct{}),
		command:               strings.Fields(command),
	}
}
