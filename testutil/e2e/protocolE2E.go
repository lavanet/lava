package e2e

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	tmclient "github.com/cometbft/cometbft/rpc/client/http"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	commonconsts "github.com/lavanet/lava/v5/testutil/common/consts"
	"github.com/lavanet/lava/v5/testutil/e2e/sdk"
	"github.com/lavanet/lava/v5/utils"
	epochStorageTypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	pairingTypes "github.com/lavanet/lava/v5/x/pairing/types"
	planTypes "github.com/lavanet/lava/v5/x/plans/types"
	specTypes "github.com/lavanet/lava/v5/x/spec/types"
	subscriptionTypes "github.com/lavanet/lava/v5/x/subscription/types"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	protocolLogsFolder         = "./testutil/e2e/protocolLogs/"
	configFolder               = "./testutil/e2e/e2eConfigs/"
	providerConfigsFolder      = configFolder + "provider"
	consumerConfigsFolder      = configFolder + "consumer"
	policiesFolder             = configFolder + "policies"
	badgeserverConfigFolder    = configFolder + "badgeserver"
	EmergencyModeStartLine     = "+++++++++++ EMERGENCY MODE START ++++++++++"
	EmergencyModeEndLine       = "+++++++++++ EMERGENCY MODE END ++++++++++"
	NumberOfSpecsExpectedInE2E = 10
	startLavaDaemonLogName     = "00_StartLava_Daemon"
)

var (
	checkedPlansE2E         = []string{"DefaultPlan", "EmergencyModePlan"}
	checkedSubscriptions    = []string{"user1", "user2", "user3", "user5"}
	checkedSpecsE2E         = []string{"LAV1", "ETH1"}
	checkedSpecsE2ELOL      = []string{"SEP1"}
	checkedSubscriptionsLOL = []string{"user4"}
)

type lavaTest struct {
	// Thread-safety fields
	testFinishedProperly atomic.Bool
	savingLogs           atomic.Bool // Prevents recursive saveLogs() calls
	logsMu               sync.RWMutex
	commandsMu           sync.RWMutex
	expectedExitMu       sync.RWMutex
	providerTypeMu       sync.RWMutex

	// Original fields
	grpcConn             *grpc.ClientConn
	lavadPath            string
	protocolPath         string
	lavadArgs            string
	consumerArgs         string
	logs                 map[string]*sdk.SafeBuffer
	commands             map[string]*exec.Cmd
	expectedCommandExit  map[string]bool
	providerType         map[string][]epochStorageTypes.Endpoint
	wg                   sync.WaitGroup
	logPath              string
	tokenDenom           string
	consumerCacheAddress string
	providerCacheAddress string
}

func init() {
	_, filename, _, _ := runtime.Caller(0)
	// Move to parent directory since running "go test ./testutil/e2e/ -v" would move working directory
	// to testutil/e2e/ which breaks relative paths in scripts
	dir := path.Join(path.Dir(filename), "../..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	fmt.Println("Test Directory", dir)
}

// contextSleep performs a context-aware sleep
func contextSleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// waitForCondition waits for a condition to be true with context
func waitForCondition(ctx context.Context, condition func() bool, checkInterval time.Duration, description string) error {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s: %w", description, ctx.Err())
		case <-ticker.C:
			if condition() {
				return nil
			}
			utils.LavaFormatInfo("Waiting for condition", utils.LogAttr("description", description))
		}
	}
}

func (lt *lavaTest) execCommandWithRetry(ctx context.Context, funcName string, logName string, command string) {
	// Check if context is already canceled before starting
	if ctx.Err() != nil {
		utils.LavaFormatError("Context already canceled, skipping command", ctx.Err(),
			utils.LogAttr("funcName", funcName),
			utils.LogAttr("command", command))
		return
	}

	utils.LavaFormatDebug("Executing command " + command)
	lt.logsMu.Lock()
	lt.logs[logName] = &sdk.SafeBuffer{}
	lt.logsMu.Unlock()

	cmd := exec.CommandContext(ctx, "", "")
	cmd.Args = strings.Fields(command)
	cmd.Path = cmd.Args[0]
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	// Set process group ID so we can kill the entire process tree
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	lt.commandsMu.Lock()
	lt.commands[logName] = cmd
	lt.commandsMu.Unlock()
	retries := 0 // Counter for retries
	maxRetries := 2

	lt.wg.Add(1)
	go func() {
		defer lt.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				// Check if context is canceled before retrying
				if ctx.Err() != nil {
					utils.LavaFormatError("Context canceled, not retrying", ctx.Err(),
						utils.LogAttr("funcName", funcName),
						utils.LogAttr("logName", logName))
					return
				}

				utils.LavaFormatError("Panic occurred", fmt.Errorf("%v", r),
					utils.LogAttr("funcName", funcName),
					utils.LogAttr("logName", logName))
				if retries < maxRetries {
					retries++
					utils.LavaFormatInfo("Restarting goroutine",
						utils.LogAttr("funcName", funcName),
						utils.LogAttr("remainingRetries", maxRetries-retries))
					go lt.execCommandWithRetry(ctx, funcName, logName, command)
				} else {
					lt.saveLogs()
					panic(fmt.Errorf("maximum number of retries exceeded for %s", funcName))
				}
			}
		}()
		lt.listenCmdCommand(cmd, funcName+" process returned unexpectedly", funcName, logName)
	}()
}

func (lt *lavaTest) execCommand(ctx context.Context, funcName string, logName string, command string, wait bool) {
	defer func() {
		if r := recover(); r != nil {
			lt.saveLogs()
			panic(fmt.Sprintf("Panic happened with command: %s", command))
		}
	}()

	lt.logsMu.Lock()
	lt.logs[logName] = &sdk.SafeBuffer{}
	lt.logsMu.Unlock()

	cmd := exec.CommandContext(ctx, "", "")
	utils.LavaFormatInfo("Executing Command: " + command)
	cmd.Args = strings.Fields(command)
	cmd.Path = cmd.Args[0]
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	// Set process group ID so we can kill the entire process tree
	// This ensures child processes (like proxy.test spawned by go test) are also killed
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Ensure PATH includes the Go bin directory
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	currentPath := os.Getenv("PATH")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s/bin:%s", gopath, currentPath))

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	if wait {
		if err = cmd.Wait(); err != nil {
			panic(funcName + " failed " + err.Error())
		}
	} else {
		lt.commandsMu.Lock()
		lt.commands[logName] = cmd
		lt.commandsMu.Unlock()
		lt.wg.Add(1)
		go func() {
			defer lt.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					utils.LavaFormatError("Panic in command listener", fmt.Errorf("%v", r),
						utils.LogAttr("funcName", funcName),
						utils.LogAttr("logName", logName))
					lt.saveLogs()
				}
			}()
			lt.listenCmdCommand(cmd, funcName+" process returned unexpectedly", funcName, logName)
		}()
	}
}

func (lt *lavaTest) listenCmdCommand(cmd *exec.Cmd, panicReason string, functionName string, logName string) {
	err := cmd.Wait()
	exitExpected := lt.consumeCommandExitExpectation(logName)

	lt.commandsMu.Lock()
	delete(lt.commands, logName)
	lt.commandsMu.Unlock()

	if err != nil && !lt.testFinishedProperly.Load() && !exitExpected {
		utils.LavaFormatError(functionName+" cmd wait err", err,
			utils.LogAttr("logName", logName))
	}
	if exitExpected {
		utils.LavaFormatInfo(functionName+" exit expected; skipping panic",
			utils.LogAttr("logName", logName))
		return
	}
	if lt.testFinishedProperly.Load() {
		return
	}
	lt.saveLogs()
	panic(panicReason)
}

func (lt *lavaTest) expectCommandExit(logName string) {
	lt.expectedExitMu.Lock()
	defer lt.expectedExitMu.Unlock()
	if lt.expectedCommandExit == nil {
		lt.expectedCommandExit = make(map[string]bool)
	}
	lt.expectedCommandExit[logName] = true
	utils.LavaFormatInfo("Marked command exit expectation",
		utils.LogAttr("logName", logName))
}

func (lt *lavaTest) consumeCommandExitExpectation(logName string) bool {
	lt.expectedExitMu.Lock()
	defer lt.expectedExitMu.Unlock()
	if lt.expectedCommandExit == nil {
		return false
	}
	val := lt.expectedCommandExit[logName]
	delete(lt.expectedCommandExit, logName)
	utils.LavaFormatDebug("Consume command exit expectation",
		utils.LogAttr("logName", logName),
		utils.LogAttr("value", val))
	return val
}

const OKstr = " OK"

func (lt *lavaTest) startLava(ctx context.Context) {
	// First initialize the chain
	initCommand := "./scripts/start_env_dev.sh"
	logName := "00_StartLava"
	funcName := "startLava"

	lt.execCommand(ctx, funcName, logName, initCommand, true)

	// Now start the daemon in the background
	startCommand := lt.lavadPath + " start"
	logNameDaemon := startLavaDaemonLogName
	funcNameDaemon := "startLavaDaemon"

	lt.execCommand(ctx, funcNameDaemon, logNameDaemon, startCommand, false)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) checkLava(timeout time.Duration) {
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

	deadline := time.Now().Add(timeout)

	// First wait for the spec query to work
	for time.Now().Before(deadline) {
		// This loop would wait for the lavad server to be up before chain init
		_, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
		if err != nil && strings.Contains(err.Error(), "rpc error") {
			utils.LavaFormatInfo("Waiting for Lava")
			if err := contextSleep(context.Background(), time.Second*10); err != nil {
				return
			}
		} else if err == nil {
			break
		} else {
			panic(err)
		}
	}

	if time.Now().After(deadline) {
		panic("Lava Check Failed: timeout waiting for spec query")
	}

	// Additionally, wait for validators to be available (needed for init_e2e.sh operator_address call)
	utils.LavaFormatInfo("Waiting for validators to be ready...")
	stakingQueryClient := stakingtypes.NewQueryClient(lt.grpcConn)
	for time.Now().Before(deadline) {
		resp, err := stakingQueryClient.Validators(context.Background(), &stakingtypes.QueryValidatorsRequest{})
		if err == nil && resp != nil && len(resp.Validators) > 0 {
			utils.LavaFormatInfo("Validators are ready",
				utils.LogAttr("validatorCount", len(resp.Validators)),
				utils.LogAttr("firstValidator", resp.Validators[0].OperatorAddress))
			return
		}
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			// If it's not an RPC error, it's something more serious
			utils.LavaFormatWarning("Error querying validators", err)
		}
		utils.LavaFormatInfo("Validators not ready yet, waiting...")
		if err := contextSleep(context.Background(), time.Second*2); err != nil {
			return
		}
	}
	panic("Lava Check Failed: validators not available")
}

func (lt *lavaTest) waitForValidators(timeout time.Duration) {
	stakingQueryClient := stakingtypes.NewQueryClient(lt.grpcConn)

	for start := time.Now(); time.Since(start) < timeout; {
		// Try to get validators
		validatorsResp, err := stakingQueryClient.Validators(context.Background(), &stakingtypes.QueryValidatorsRequest{})
		if err == nil && validatorsResp != nil && len(validatorsResp.Validators) > 0 {
			utils.LavaFormatInfo("Validators registered")
			return
		}

		utils.LavaFormatInfo("Waiting for validators to be registered")
		if err := contextSleep(context.Background(), time.Second*5); err != nil {
			return
		}
	}
	panic("Validators not registered within timeout")
}

func (lt *lavaTest) stakeLava(ctx context.Context) {
	command := "./scripts/test/init_e2e.sh"
	logName := "01_stakeLava"
	funcName := "stakeLava"

	lt.execCommand(ctx, funcName, logName, command, true)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) checkStakeLava(
	planCount int,
	specCount int,
	subsCount int,
	providerCount int,
	checkedPlans []string,
	checkedSpecs []string,
	checkedSubscriptions []string,
	successMessage string,
) {
	planQueryClient := planTypes.NewQueryClient(lt.grpcConn)

	// query all plans
	planQueryRes, err := planQueryClient.List(context.Background(), &planTypes.QueryListRequest{})
	if err != nil {
		panic(err)
	}

	// check if plans added exist
	if len(planQueryRes.PlansInfo) != planCount {
		panic(fmt.Errorf("staking failed: plan count mismatch - expected %d plans, got %d plans\nPlans found: %+v",
			planCount, len(planQueryRes.PlansInfo), planQueryRes.PlansInfo))
	}

	for _, plan := range planQueryRes.PlansInfo {
		if !slices.Contains(checkedPlans, plan.Index) {
			panic(fmt.Errorf("staking failed: unexpected plan found - got '%s', expected one of %v",
				plan.Index, checkedPlans))
		}
	}

	subsQueryClient := subscriptionTypes.NewQueryClient(lt.grpcConn)

	// query all subscriptions
	subsQueryRes, err := subsQueryClient.List(context.Background(), &subscriptionTypes.QueryListRequest{})
	if err != nil {
		panic(err)
	}

	// check if subscriptions added exist
	if len(subsQueryRes.SubsInfo) != subsCount {
		panic(fmt.Errorf("staking failed: subscription count mismatch - expected %d subscriptions, got %d subscriptions\nSubscriptions found: %+v",
			subsCount, len(subsQueryRes.SubsInfo), subsQueryRes.SubsInfo))
	}

	for _, key := range checkedSubscriptions {
		subscriptionQueryClient := subscriptionTypes.NewQueryClient(lt.grpcConn)
		_, err = subscriptionQueryClient.Current(context.Background(), &subscriptionTypes.QueryCurrentRequest{Consumer: lt.getKeyAddress(key)})
		if err != nil {
			panic(fmt.Errorf("staking failed: could not get subscription for user '%s' (address: %s): %w",
				key, lt.getKeyAddress(key), err))
		}
	}

	// providerCount and clientCount refers to number and providers and client for each spec
	// number of providers and clients should be the same for all specs for simplicity's sake
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

	// query all specs
	specQueryRes, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
	if err != nil {
		panic(err)
	}

	pairingQueryClient := pairingTypes.NewQueryClient(lt.grpcConn)
	// check if all specs added exist
	if len(specQueryRes.Spec) != specCount {
		utils.LavaFormatError("Spec count mismatch", nil,
			utils.LogAttr("expected", specCount),
			utils.LogAttr("actual", len(specQueryRes.Spec)),
			utils.LogAttr("specs", specQueryRes.Spec))
		panic(fmt.Errorf("staking failed: spec count mismatch - expected %d specs, got %d specs",
			specCount, len(specQueryRes.Spec)))
	}
	for _, spec := range specQueryRes.Spec {
		if !slices.Contains(checkedSpecs, spec.Index) {
			continue
		}
		// Query providers

		fmt.Println(spec.GetIndex())
		providerQueryRes, err := pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
			panic(err)
		}
		if len(providerQueryRes.StakeEntry) != providerCount {
			utils.LavaFormatError("Provider count mismatch", nil,
				utils.LogAttr("chainID", spec.GetIndex()),
				utils.LogAttr("expected", providerCount),
				utils.LogAttr("actual", len(providerQueryRes.StakeEntry)),
				utils.LogAttr("providers", providerQueryRes.StakeEntry))
			panic(fmt.Errorf("staking failed: provider count mismatch for chain %s - expected %d providers, got %d providers",
				spec.GetIndex(), providerCount, len(providerQueryRes.StakeEntry)))
		}
		for _, providerStakeEntry := range providerQueryRes.StakeEntry {
			fmt.Println("provider", providerStakeEntry.Address, providerStakeEntry.Endpoints)
			lt.providerTypeMu.Lock()
			lt.providerType[providerStakeEntry.Address] = providerStakeEntry.Endpoints
			lt.providerTypeMu.Unlock()
		}
	}
	utils.LavaFormatInfo(successMessage)
}

func (lt *lavaTest) checkBadgeServerResponsive(ctx context.Context, badgeServerAddr string, timeout time.Duration) {
	// Use exponential backoff for more efficient waiting
	initialDelay := 500 * time.Millisecond
	maxDelay := 5 * time.Second
	currentDelay := initialDelay

	deadline := time.Now().Add(timeout)
	attemptCount := 0

	for time.Now().Before(deadline) {
		attemptCount++
		if attemptCount%5 == 0 { // Log every 5th attempt to reduce noise
			utils.LavaFormatInfo("Waiting for Badge Server "+badgeServerAddr,
				utils.LogAttr("attempt", attemptCount),
				utils.LogAttr("remaining", time.Until(deadline).Round(time.Second)))
		}

		nctx, cancel := context.WithTimeout(ctx, 2*time.Second) // Increased from 1s to 2s
		grpcClient, err := grpc.DialContext(nctx, badgeServerAddr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		cancel()

		if err == nil {
			grpcClient.Close()
			utils.LavaFormatInfo("Badge Server is responsive "+badgeServerAddr,
				utils.LogAttr("attempts", attemptCount))
			return
		}

		// Exponential backoff with cap
		time.Sleep(currentDelay)
		currentDelay = time.Duration(float64(currentDelay) * 1.5)
		if currentDelay > maxDelay {
			currentDelay = maxDelay
		}
	}

	panic(fmt.Sprintf("checkBadgeServerResponsive: Badge server didn't respond after %d attempts: %s", attemptCount, badgeServerAddr))
}

func (lt *lavaTest) startJSONRPCProxy(ctx context.Context) {
	goExecutablePath, err := exec.LookPath("go")
	if err != nil {
		panic("Could not find go executable path")
	}
	// force go's test timeout to 0, otherwise the default is 10m; our timeout
	// will be enforced by the given ctx.
	command := fmt.Sprintf("%s test ./testutil/e2e/proxy/. -v -timeout 0 -args -host eth", goExecutablePath)
	logName := "02_jsonProxy"
	funcName := "startJSONRPCProxy"

	lt.execCommand(ctx, funcName, logName, command, false)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) startJSONRPCProvider(ctx context.Context) {
	for idx := 1; idx <= 5; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/jsonrpcProvider%d.yml --cache-be %s --chain-id=lava --from servicer%d %s",
			lt.protocolPath, providerConfigsFolder, idx, lt.providerCacheAddress, idx, lt.lavadArgs,
		)
		logName := "03_EthProvider_" + fmt.Sprintf("%02d", idx)
		funcName := fmt.Sprintf("startJSONRPCProvider (provider %02d)", idx)
		lt.execCommandWithRetry(ctx, funcName, logName, command)
	}

	// validate all providers are up
	for idx := 1; idx < 5; idx++ {
		lt.checkProviderResponsive(ctx, fmt.Sprintf("127.0.0.1:222%d", idx), time.Minute)
	}

	utils.LavaFormatInfo("startJSONRPCProvider OK")
}

func (lt *lavaTest) startJSONRPCConsumer(ctx context.Context) {
	for idx, u := range []string{"user1"} {
		command := fmt.Sprintf(
			"%s rpcconsumer %s/ethConsumer%d.yml --cache-be %s --chain-id=lava --from %s %s",
			lt.protocolPath, consumerConfigsFolder, idx+1, lt.consumerCacheAddress, u, lt.lavadArgs+lt.consumerArgs,
		)
		logName := "04_jsonConsumer_" + fmt.Sprintf("%02d", idx+1)
		funcName := fmt.Sprintf("startJSONRPCConsumer (consumer %02d)", idx+1)
		lt.execCommand(ctx, funcName, logName, command, false)
	}
	utils.LavaFormatInfo("startJSONRPCConsumer OK")
}

// If after timeout and the check does not return it means it failed
func (lt *lavaTest) checkJSONRPCConsumer(rpcURL string, timeout time.Duration, message string) {
	// Use exponential backoff for more efficient waiting
	initialDelay := 500 * time.Millisecond
	maxDelay := 5 * time.Second
	currentDelay := initialDelay

	deadline := time.Now().Add(timeout)
	attemptCount := 0

	for time.Now().Before(deadline) {
		attemptCount++
		if attemptCount%5 == 0 { // Log every 5th attempt to reduce noise
			utils.LavaFormatInfo("Waiting JSONRPC Consumer",
				utils.LogAttr("url", rpcURL),
				utils.LogAttr("attempt", attemptCount),
				utils.LogAttr("remaining", time.Until(deadline).Round(time.Second)))
		}

		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			time.Sleep(currentDelay)
			currentDelay = time.Duration(float64(currentDelay) * 1.5)
			if currentDelay > maxDelay {
				currentDelay = maxDelay
			}
			continue
		}

		res, err := client.BlockNumber(context.Background())
		client.Close()

		if err == nil {
			utils.LavaFormatInfo(message)
			utils.LavaFormatInfo("Validated proxy is alive got response",
				utils.Attribute{Key: "res", Value: res},
				utils.LogAttr("attempts", attemptCount))
			return
		}

		// Exponential backoff with cap
		time.Sleep(currentDelay)
		currentDelay = time.Duration(float64(currentDelay) * 1.5)
		if currentDelay > maxDelay {
			currentDelay = maxDelay
		}
	}

	panic(fmt.Sprintf("checkJSONRPCConsumer: Consumer didn't respond after %d attempts: %s", attemptCount, rpcURL))
}

func (lt *lavaTest) checkProviderResponsive(ctx context.Context, rpcURL string, timeout time.Duration) {
	// Use exponential backoff for more efficient waiting
	initialDelay := 500 * time.Millisecond
	maxDelay := 5 * time.Second
	currentDelay := initialDelay

	deadline := time.Now().Add(timeout)
	attemptCount := 0

	for time.Now().Before(deadline) {
		attemptCount++
		if attemptCount%5 == 0 { // Log every 5th attempt to reduce noise
			utils.LavaFormatInfo("Waiting Provider "+rpcURL,
				utils.LogAttr("attempt", attemptCount),
				utils.LogAttr("remaining", time.Until(deadline).Round(time.Second)))
		}

		nctx, cancel := context.WithTimeout(ctx, 2*time.Second) // Increased from 1s to 2s
		var tlsConf tls.Config
		tlsConf.InsecureSkipVerify = true // skip CA validation
		credentials := credentials.NewTLS(&tlsConf)
		grpcClient, err := grpc.DialContext(nctx, rpcURL, grpc.WithBlock(), grpc.WithTransportCredentials(credentials))
		cancel()

		if err == nil {
			grpcClient.Close()
			utils.LavaFormatInfo("Provider is responsive "+rpcURL,
				utils.LogAttr("attempts", attemptCount))
			return
		}

		// Exponential backoff with cap
		time.Sleep(currentDelay)
		currentDelay = time.Duration(float64(currentDelay) * 1.5)
		if currentDelay > maxDelay {
			currentDelay = maxDelay
		}
	}

	panic(fmt.Sprintf("checkProviderResponsive: Provider didn't respond after %d attempts: %s", attemptCount, rpcURL))
}

func jsonrpcTests(rpcURL string, testDuration time.Duration) error {
	ctx := context.Background()
	utils.LavaFormatInfo("Starting JSONRPC Tests")
	errors := []string{}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return err
	}
	rawClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return err
	}
	for start := time.Now(); time.Since(start) < testDuration; {
		// eth_blockNumber
		latestBlockNumberUint, err := client.BlockNumber(ctx)
		if err != nil {
			errors = append(errors, "error eth_blockNumber")
		}

		// put in a loop for cases that a block have no tx because
		var latestBlock *ethtypes.Block
		var latestBlockNumber *big.Int
		var latestBlockTxs ethtypes.Transactions
		for {
			// eth_getBlockByNumber
			latestBlockNumber = big.NewInt(int64(latestBlockNumberUint))
			latestBlock, err = client.BlockByNumber(ctx, latestBlockNumber)
			if err != nil {
				errors = append(errors, "error eth_getBlockByNumber")
				continue
			}
			latestBlockTxs = latestBlock.Transactions()

			if len(latestBlockTxs) == 0 {
				latestBlockNumberUint -= 1
				continue
			}
			break
		}

		// eth_gasPrice
		_, err = client.SuggestGasPrice(ctx)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			errors = append(errors, "error eth_gasPrice")
		}

		targetTx := latestBlockTxs[0]

		// eth_getTransactionByHash
		targetTx, _, err = client.TransactionByHash(ctx, targetTx.Hash())
		if err != nil {
			errors = append(errors, "error eth_getTransactionByHash")
		}

		// eth_getTransactionReceipt
		_, err = client.TransactionReceipt(ctx, targetTx.Hash())
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			errors = append(errors, "error eth_getTransactionReceipt")
		}

		sender, err := ethtypes.Sender(ethtypes.LatestSignerForChainID(targetTx.ChainId()), targetTx)
		utils.LavaFormatInfo("sender", utils.Attribute{Key: "sender", Value: sender})
		if err != nil {
			errors = append(errors, "error eth_getTransactionReceipt")
		}

		// eth_getBalance
		balance, err := client.BalanceAt(ctx, sender, nil)
		utils.LavaFormatInfo("balance", utils.Attribute{Key: "balance", Value: balance})
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			errors = append(errors, "error eth_getBalance")
		}

		// eth_getStorageAt
		_, err = client.StorageAt(ctx, *targetTx.To(), common.HexToHash("00"), nil)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			errors = append(errors, "error eth_getStorageAt")
		}

		// eth_getCode
		_, err = client.CodeAt(ctx, *targetTx.To(), nil)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			errors = append(errors, "error eth_getCode")
		}

		// debug and extensions test:
		blockZero := big.NewInt(0)
		_, err = client.BlockByNumber(ctx, blockZero)
		if err != nil {
			errors = append(errors, "error eth_getBlockByNumber")
			continue
		}
		var result interface{}
		err = rawClient.CallContext(ctx, &result, "debug_getRawHeader", "latest")
		if err != nil {
			errors = append(errors, "error debug_getRawHeader")
			continue
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}

	return nil
}

func (lt *lavaTest) startLavaProviders(ctx context.Context) {
	for idx := 6; idx <= 10; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/lavaProvider%d --cache-be %s --chain-id=lava --from servicer%d %s",
			lt.protocolPath, providerConfigsFolder, idx, lt.providerCacheAddress, idx, lt.lavadArgs,
		)
		logName := "05_LavaProvider_" + fmt.Sprintf("%02d", idx-5)
		funcName := fmt.Sprintf("startLavaProviders (provider %02d)", idx-5)
		lt.execCommand(ctx, funcName, logName, command, false)
	}

	// validate all providers are up
	for idx := 6; idx <= 10; idx++ {
		lt.checkProviderResponsive(ctx, fmt.Sprintf("127.0.0.1:226%d", idx-5), time.Minute)
	}

	utils.LavaFormatInfo("startLavaProviders OK")
}

func (lt *lavaTest) startLavaConsumer(ctx context.Context) {
	for idx, u := range []string{"user3"} {
		command := fmt.Sprintf(
			"%s rpcconsumer %s/lavaConsumer%d.yml --cache-be %s --chain-id=lava --from %s %s",
			lt.protocolPath, consumerConfigsFolder, idx+1, lt.consumerCacheAddress, u, lt.lavadArgs+lt.consumerArgs,
		)
		logName := "06_RPCConsumer_" + fmt.Sprintf("%02d", idx+1)
		funcName := fmt.Sprintf("startRPCConsumer (consumer %02d)", idx+1)
		lt.execCommand(ctx, funcName, logName, command, false)
	}
	utils.LavaFormatInfo("startRPCConsumer OK")
}

func (lt *lavaTest) startConsumerCache(ctx context.Context) {
	command := fmt.Sprintf("%s cache %s --log_level debug", lt.protocolPath, lt.consumerCacheAddress)
	logName := "08_Consumer_Cache"
	funcName := "startConsumerCache"

	lt.execCommand(ctx, funcName, logName, command, false)
	lt.checkCacheIsUp(ctx, lt.consumerCacheAddress, time.Minute)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) startProviderCache(ctx context.Context) {
	command := fmt.Sprintf("%s cache %s --log_level debug", lt.protocolPath, lt.providerCacheAddress)
	logName := "09_Provider_Cache"
	funcName := "startProviderCache"

	lt.execCommand(ctx, funcName, logName, command, false)
	lt.checkCacheIsUp(ctx, lt.providerCacheAddress, time.Minute)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) checkCacheIsUp(ctx context.Context, cacheAddress string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting Cache " + cacheAddress)
		nctx, cancel := context.WithTimeout(ctx, time.Second)
		grpcClient, err := grpc.DialContext(nctx, cacheAddress, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			cancel()
			time.Sleep(time.Second)
			continue
		}
		cancel()
		grpcClient.Close()
		return
	}
	panic("checkCacheIsUp: Check Failed Cache didn't respond" + cacheAddress)
}

func (lt *lavaTest) startLavaEmergencyConsumer(ctx context.Context) {
	for idx, u := range []string{"user5"} {
		command := fmt.Sprintf(
			"%s rpcconsumer %s/lavaConsumerEmergency%d.yml --chain-id=lava --from %s %s",
			lt.protocolPath, consumerConfigsFolder, idx+1, u, lt.lavadArgs+lt.consumerArgs,
		)
		logName := "11_RPCEmergencyConsumer_" + fmt.Sprintf("%02d", idx+1)
		funcName := fmt.Sprintf("startRPCEmergencyConsumer (consumer %02d)", idx+1)
		lt.execCommand(ctx, funcName, logName, command, false)
	}
	utils.LavaFormatInfo("startRPCEmergencyConsumer OK")
}

func (lt *lavaTest) checkTendermintConsumer(rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting TENDERMINT Consumer")
		client, err := tmclient.New(rpcURL, "/websocket")
		if err != nil {
			continue
		}
		_, err = client.Status(context.Background())
		if err == nil {
			utils.LavaFormatInfo("checkTendermintConsumer OK")
			return
		}
		time.Sleep(time.Second)
	}
	panic("TENDERMINT Check Failed")
}

func tendermintTests(rpcURL string, testDuration time.Duration) error {
	ctx := context.Background()
	utils.LavaFormatInfo("Starting TENDERMINT Tests")
	errors := []string{}
	successCount := 0
	client, err := tmclient.New(rpcURL, "/websocket")
	if err != nil {
		errors = append(errors, "error client dial")
	}
	for start := time.Now(); time.Since(start) < testDuration; {
		_, err := client.Status(ctx)
		if err != nil {
			// Ignore epoch mismatch errors during initial sync period
			if !strings.Contains(err.Error(), "Tried to Report to an older epoch") &&
				!strings.Contains(err.Error(), "provider lava block") {
				errors = append(errors, err.Error())
			}
		} else {
			successCount++
		}
		_, err = client.Health(ctx)
		if err != nil {
			// Ignore epoch mismatch errors during initial sync period
			if !strings.Contains(err.Error(), "Tried to Report to an older epoch") &&
				!strings.Contains(err.Error(), "provider lava block") {
				errors = append(errors, err.Error())
			}
		} else {
			successCount++
		}
	}
	// Only fail if we got ZERO successes - some errors during sync are expected
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}
	return nil
}

func tendermintURITests(rpcURL string, testDuration time.Duration) error {
	utils.LavaFormatInfo("Starting TENDERMINTRPC URI Tests")
	errors := []string{}
	successCount := 0
	mostImportantApisToTest := map[string]bool{
		"%s/health":                              true,
		"%s/status":                              true,
		"%s/block?height=1":                      true,
		"%s/blockchain?minHeight=0&maxHeight=10": true,
		// "%s/dial_peers?persistent=true&unconditional=true&private=true": false, // this is a rpc affecting query and is not available on the spec so it should fail
	}
	for start := time.Now(); time.Since(start) < testDuration; {
		for api, noFail := range mostImportantApisToTest {
			reply, err := getRequest(fmt.Sprintf(api, rpcURL))
			if err != nil && noFail {
				// Ignore epoch mismatch errors during initial sync period
				errorStr := fmt.Sprintf("%s", err)
				if !strings.Contains(errorStr, "Tried to Report to an older epoch") &&
					!strings.Contains(errorStr, "provider lava block") {
					errors = append(errors, errorStr)
				}
			} else if strings.Contains(string(reply), "error") && noFail {
				// Ignore epoch mismatch errors in responses
				replyStr := string(reply)
				if !strings.Contains(replyStr, "Tried to Report to an older epoch") &&
					!strings.Contains(replyStr, "provider lava block") {
					errors = append(errors, replyStr)
				}
			} else if err == nil && !strings.Contains(string(reply), "error") {
				successCount++
			}
		}
	}

	// Only fail if we got ZERO successes - some errors during sync are expected
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}
	return nil
}

// This would submit a proposal, vote then stake providers and clients for that network over lava
func (lt *lavaTest) lavaOverLava(ctx context.Context) {
	utils.LavaFormatInfo("Starting Lava over Lava Tests")
	command := "./scripts/test/init_e2e_lava_over_lava.sh"
	lt.execCommand(ctx, "startJSONRPCConsumer", "07_lavaOverLava", command, true)

	// scripts/test/init_e2e.sh will:
	// - produce 5 specs: ETH1, HOL1, SEP1, IBC,TENDERMINT , COSMOSSDK, LAV1 (via {ethereum,cosmoshub,lava})
	// - produce 2 plans: "DefaultPlan", "EmergencyModePlan"

	lt.checkStakeLava(2, NumberOfSpecsExpectedInE2E, 4, 5, checkedPlansE2E, checkedSpecsE2ELOL, checkedSubscriptionsLOL, "Lava Over Lava Test OK")
}

func (lt *lavaTest) checkRESTConsumer(rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting REST Consumer")
		reply, err := getRequest(fmt.Sprintf("%s/cosmos/base/tendermint/v1beta1/blocks/latest", rpcURL))
		if err != nil || strings.Contains(string(reply), "error") {
			time.Sleep(time.Second)
			continue
		} else {
			utils.LavaFormatInfo("checkRESTConsumer OK")
			return
		}
	}
	panic("REST Check Failed")
}

func restTests(rpcURL string, testDuration time.Duration) error {
	utils.LavaFormatInfo("Starting REST Tests")
	errors := []string{}
	mostImportantApisToTest := []string{
		"%s/cosmos/base/tendermint/v1beta1/blocks/latest",
		"%s/lavanet/lava/pairing/providers/LAV1",
		"%s/lavanet/lava/pairing/clients/LAV1",
		"%s/cosmos/gov/v1beta1/proposals",
		"%s/lavanet/lava/spec/spec",
		"%s/cosmos/base/tendermint/v1beta1/blocks/1",
	}
	// APIs that are expected to return errors (due to unsupported method feature)
	expectedErrorAPIs := map[string]bool{
		"%s/lavanet/lava/pairing/clients/LAV1": true,
	}

	for start := time.Now(); time.Since(start) < testDuration; {
		for _, api := range mostImportantApisToTest {
			reply, err := getRequest(fmt.Sprintf(api, rpcURL))
			allowErrors := expectedErrorAPIs[api]

			if err != nil && !allowErrors {
				errors = append(errors, fmt.Sprintf("%s", err))
			} else if err == nil {
				var jsonReply map[string]interface{}
				err = json.Unmarshal(reply, &jsonReply)
				if err != nil {
					errors = append(errors, fmt.Sprintf("%s", err))
				} else if jsonReply != nil && jsonReply["error"] != nil && !allowErrors {
					errors = append(errors, string(reply))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}
	return nil
}

func restRelayTest(rpcURL string) error {
	errors := []string{}
	apiToTest := "%s/cosmos/base/tendermint/v1beta1/blocks/1"

	fullURL := fmt.Sprintf(apiToTest, rpcURL)
	utils.LavaFormatDebug("restRelayTest: calling getRequest", utils.LogAttr("url", fullURL))
	reply, err := getRequest(fullURL)
	utils.LavaFormatDebug("restRelayTest: getRequest returned", utils.LogAttr("url", fullURL), utils.LogAttr("err", err))
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s", err))
	} else if strings.Contains(string(reply), "error") {
		errors = append(errors, string(reply))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}
	return nil
}

func getRequest(url string) ([]byte, error) {
	// Use an explicit context deadline so requests can't hang
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Surface HTTP errors instead of silently returning a body
	if res.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("http %d: %s", res.StatusCode, string(body))
	}

	return body, nil
}

func (lt *lavaTest) checkGRPCConsumer(rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting GRPC Consumer")
		grpcConn, err := grpc.Dial(rpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			continue
		}
		specQueryClient := specTypes.NewQueryClient(grpcConn)
		_, err = specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
		if err == nil {
			utils.LavaFormatInfo("checkGRPCConsumer OK")
			return
		}
		time.Sleep(time.Second)
	}
	panic("GRPC Check Failed")
}

func grpcTests(rpcURL string, testDuration time.Duration) error {
	ctx := context.Background()
	utils.LavaFormatInfo("Starting GRPC Tests")
	grpcConn, err := grpc.Dial(rpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("error client dial: %s", err.Error())
	}
	errors := []string{}
	specQueryClient := specTypes.NewQueryClient(grpcConn)
	pairingQueryClient := pairingTypes.NewQueryClient(grpcConn)
	for start := time.Now(); time.Since(start) < testDuration; {
		specQueryRes, err := specQueryClient.SpecAll(ctx, &specTypes.QueryAllSpecRequest{})
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		for _, spec := range specQueryRes.Spec {
			_, err = pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
				ChainID: spec.GetIndex(),
			})
			if err != nil {
				errors = append(errors, err.Error())
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}
	return nil
}

func (lt *lavaTest) finishTestSuccessfully() {
	lt.testFinishedProperly.Store(true)

	lt.commandsMu.RLock()

	// CRITICAL FIX: Copy the commands map so we can release the lock BEFORE killing
	// This prevents deadlock when killed processes' monitoring goroutines try to acquire Write lock
	commandsCopy := make(map[string]*exec.Cmd)
	for name, cmd := range lt.commands {
		commandsCopy[name] = cmd
	}

	lt.commandsMu.RUnlock()

	for name, cmd := range commandsCopy { // kill all the project commands
		if cmd != nil && cmd.Process != nil {
			utils.LavaFormatInfo("Killing process", utils.LogAttr("name", name))

			// Kill the entire process group to ensure child processes are also terminated
			// This is critical for processes like "go test" that spawn child processes (e.g., proxy.test)
			pgid, err := syscall.Getpgid(cmd.Process.Pid)

			if err == nil {
				// Kill the process group (negative PID kills the group)
				if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
					utils.LavaFormatWarning("Failed to kill process group, falling back to single process", err,
						utils.LogAttr("name", name), utils.LogAttr("pgid", pgid))
					// Fallback to killing just the process
					if err := cmd.Process.Kill(); err != nil {
						utils.LavaFormatError("Failed to kill process", err, utils.LogAttr("name", name))
					}
				}
			} else {
				// If we can't get the process group, just kill the process
				if err := cmd.Process.Kill(); err != nil {
					utils.LavaFormatError("Failed to kill process", err, utils.LogAttr("name", name))
				}
			}
		}
	}
}

func (lt *lavaTest) saveLogs() {
	// Prevent recursive calls that cause double panics
	if !lt.savingLogs.CompareAndSwap(false, true) {
		utils.LavaFormatWarning("saveLogs already running, skipping recursive call to prevent double panic", nil)
		return
	}
	defer lt.savingLogs.Store(false)

	if _, err := os.Stat(lt.logPath); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(lt.logPath, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	errorFound := false
	errorFiles := []string{}
	errorPrint := make(map[string]string)

	// Create a copy of logs to avoid holding the lock for too long
	lt.logsMu.RLock()
	logsCopy := make(map[string]*sdk.SafeBuffer)
	for k, v := range lt.logs {
		logsCopy[k] = v
	}
	lt.logsMu.RUnlock()

	for fileName, logBuffer := range logsCopy {
		reachedEmergencyModeLine := false
		file, err := os.Create(lt.logPath + fileName + ".log")
		if err != nil {
			panic(err)
		}
		writer := bufio.NewWriter(file)
		var bytesWritten int
		bytesWritten, err = writer.Write(logBuffer.Bytes())
		if err != nil {
			utils.LavaFormatError("Error writing to file", err)
		} else {
			err = writer.Flush()
			if err != nil {
				utils.LavaFormatError("Error flushing writer", err)
			} else {
				utils.LavaFormatDebug("success writing to file",
					utils.LogAttr("fileName", fileName),
					utils.LogAttr("bytesWritten", bytesWritten),
					utils.LogAttr("lines", len(logBuffer.Bytes())),
				)
			}
		}
		file.Close()

		lines := strings.Split(logBuffer.String(), "\n")
		errorLines := []string{}
		for idx, line := range lines {
			if fileName == "00_StartLava" { // TODO remove this and solve the errors
				break
			}
			if strings.Contains(line, EmergencyModeStartLine) {
				utils.LavaFormatInfo("Found Emergency start line", utils.LogAttr("file", fileName), utils.LogAttr("Line index", idx))
				reachedEmergencyModeLine = true
			}
			if strings.Contains(line, EmergencyModeEndLine) {
				utils.LavaFormatInfo("Found Emergency end line", utils.LogAttr("file", fileName), utils.LogAttr("Line index", idx))
				reachedEmergencyModeLine = false
			}
			if strings.Contains(line, " ERR ") || strings.Contains(line, "[Error]" /* sdk errors*/) {
				isAllowedError := false

				for errorSubstring := range allowedErrors {
					if strings.Contains(line, errorSubstring) {
						isAllowedError = true
						break
					}
				}
				// parse emergency possible errors as well.
				if reachedEmergencyModeLine {
					for errorSubstring := range allowedErrorsDuringEmergencyMode {
						if strings.Contains(line, errorSubstring) {
							isAllowedError = true
							break
						}
					}
				}
				// When test did not finish properly save all logs. If test finished properly save only non allowed errors.
				if !isAllowedError {
					errorFound = true
					errorLines = append(errorLines, line)
				} else if !lt.testFinishedProperly.Load() {
					// Save allowed errors for debugging when test didn't finish properly, but don't treat them as failures
					errorLines = append(errorLines, line+" [ALLOWED ERROR]")
				}
			}
		}
		if len(errorLines) == 0 {
			continue
		}

		// dump all errors into the log file
		errors := strings.Join(errorLines, "\n")
		errFile, err := os.Create(lt.logPath + fileName + "_errors.log")
		if err != nil {
			panic(err)
		}
		writer = bufio.NewWriter(errFile)
		writer.Write([]byte(errors))
		writer.Flush()
		errFile.Close()

		// keep at most 5 errors to display
		count := len(errorLines)
		if count > 5 {
			count = 5
		}
		errorPrint[fileName] = strings.Join(errorLines[:count], "\n")
		errorFiles = append(errorFiles, fileName)
	}

	if errorFound {
		fmt.Println("========================================")
		fmt.Println("ERRORS FOUND IN E2E TEST LOGS")
		fmt.Println("========================================")
		for fileName, errLines := range errorPrint {
			fmt.Printf("\n--- File: %s ---\n", fileName)
			fmt.Println(errLines)
		}
		fmt.Println("========================================")
		panic("Error found in logs on " + lt.logPath + strings.Join(errorFiles, ", "))
	}
}

func (lt *lavaTest) checkQoS() error {
	utils.LavaFormatInfo("Starting QoS Tests")
	errors := []string{}

	pairingClient := pairingTypes.NewQueryClient(lt.grpcConn)
	providerCU, err := calculateProviderCU(pairingClient)
	if err != nil {
		panic("Provider CU calculation error!")
	}
	providerIdx := 0
	for provider := range providerCU {
		// Get sequence number of provider
		logNameAcc := "8_authAccount" + fmt.Sprintf("%02d", providerIdx)
		lt.logs[logNameAcc] = &sdk.SafeBuffer{}

		fetchAccCommand := lt.lavadPath + " query account " + provider + " --output=json"
		cmdAcc := exec.CommandContext(context.Background(), "", "")
		cmdAcc.Path = lt.lavadPath
		cmdAcc.Args = strings.Split(fetchAccCommand, " ")
		cmdAcc.Stdout = lt.logs[logNameAcc]
		cmdAcc.Stderr = lt.logs[logNameAcc]
		err := cmdAcc.Start()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s", err))
		}
		lt.commands[logNameAcc] = cmdAcc
		err = cmdAcc.Wait()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s", err))
		}

		var obj map[string]interface{}
		err = json.Unmarshal((lt.logs[logNameAcc].Bytes()), &obj)
		if err != nil {
			panic(err)
		}

		sequence, ok := obj["sequence"].(string)
		if !ok {
			panic("Sequence field is not valid!")
		}
		sequenceInt, err := strconv.ParseInt(sequence, 10, 64)
		if err != nil {
			panic(err)
		}
		sequenceInt--
		sequence = strconv.Itoa(int(sequenceInt))
		//
		logName := "9_QoS_" + fmt.Sprintf("%02d", providerIdx)
		lt.logsMu.Lock()
		lt.logs[logName] = &sdk.SafeBuffer{}
		lt.logsMu.Unlock()

		txQueryCommand := lt.lavadPath + " query tx --type=acc_seq " + provider + "/" + sequence

		cmd := exec.CommandContext(context.Background(), "", "")
		cmd.Path = lt.lavadPath
		cmd.Args = strings.Split(txQueryCommand, " ")
		cmd.Stdout = lt.logs[logName]
		cmd.Stderr = lt.logs[logName]

		err = cmd.Start()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s", err))
		}
		lt.commandsMu.Lock()
		lt.commands[logName] = cmd
		lt.commandsMu.Unlock()
		err = cmd.Wait()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s", err))
		}

		lines := strings.Split(lt.logs[logName].String(), "\n")
		for idx, line := range lines {
			if strings.Contains(line, "key: QoSScore") {
				startIndex := strings.Index(lines[idx+1], "\"") + 1
				endIndex := strings.LastIndex(lines[idx+1], "\"")
				qosScoreStr := lines[idx+1][startIndex:endIndex]
				qosScore, err := strconv.ParseFloat(qosScoreStr, 64)
				if err != nil {
					errors = append(errors, fmt.Sprintf("%s", err))
				}
				if qosScore < 1 {
					errors = append(errors, "QoS score is less than 1 !")
				}
			}
		}
		providerIdx++
	}
	utils.LavaFormatInfo("QOS CHECK OK")

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}
	return nil
}

func (lt *lavaTest) startLavaInEmergencyMode(ctx context.Context, timeoutCommit int) {
	// We need to use the emergency mode script as it properly handles the config update
	// The script modifies the timeout_commit in the existing config without re-initializing the chain
	command := "./scripts/test/emergency_mode.sh " + strconv.Itoa(timeoutCommit)
	logName := "10_StartLavaInEmergencyMode"
	funcName := "startLavaInEmergencyMode"

	// Set up environment to ensure lavad is in PATH
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	lt.logsMu.Lock()
	lt.logs[logName] = &sdk.SafeBuffer{}
	lt.logsMu.Unlock()
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	// Add GOPATH/bin to PATH so the script can find lavad
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s/bin:%s", gopath, os.Getenv("PATH")))

	utils.LavaFormatInfo("Executing emergency mode script", utils.LogAttr("command", command))

	err := cmd.Start()
	if err != nil {
		panic(fmt.Errorf("failed to start emergency mode: %w", err))
	}

	lt.commandsMu.Lock()
	lt.commands[logName] = cmd
	lt.commandsMu.Unlock()

	// Monitor the command in background
	lt.wg.Add(1)
	go func() {
		defer lt.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				utils.LavaFormatError("Panic in emergency mode command", fmt.Errorf("%v", r))
				lt.saveLogs()
			}
		}()

		if err := cmd.Wait(); err != nil {
			if lt.testFinishedProperly.Load() {
				return
			}
			utils.LavaFormatError("Emergency mode process failed", err)
			lt.saveLogs()
			panic(fmt.Sprintf("Emergency mode process returned unexpectedly: %v", err))
		}
	}()

	utils.LavaFormatInfo(funcName + " OK")
}

func (lt *lavaTest) sleepUntilNextEpoch() {
	cmd := exec.Command("/bin/bash", "-c", "source ./scripts/useful_commands.sh && sleep_until_next_epoch")

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	utils.LavaFormatInfo("sleepUntilNextEpoch" + " OK")
}

func (lt *lavaTest) markEmergencyModeLogsStart() {
	// Create a copy of logs to avoid holding the lock for too long
	lt.logsMu.RLock()
	logsCopy := make(map[string]*sdk.SafeBuffer)
	for k, v := range lt.logs {
		logsCopy[k] = v
	}
	lt.logsMu.RUnlock()

	for log, buffer := range logsCopy {
		_, err := buffer.WriteString(EmergencyModeStartLine + "\n")
		utils.LavaFormatInfo("Adding EmergencyMode Start Line to", utils.LogAttr("log_name", log))
		if err != nil {
			utils.LavaFormatError("Failed Writing to buffer", err, utils.LogAttr("key", log))
		}

		// Verify that the EmergencyModeStartLine is in the last 20 lines
		contents := buffer.String()
		lines := strings.Split(contents, "\n")
		start := len(lines) - 21 // -21 because we want to check the last 20 lines, and -1 for 0-indexing
		if start < 0 {
			start = 0
		}
		last20Lines := lines[start : len(lines)-1] // Exclude the last empty string after the final split
		// Check if EmergencyModeStartLine is present in the last 20 lines
		found := false
		indexFound := 0
		for idx, line := range last20Lines {
			if strings.Contains(line, EmergencyModeStartLine) {
				found = true
				indexFound = idx
				break
			}
		}
		if found {
			utils.LavaFormatInfo("Successfully verified EmergencyMode Start Line in the last 20 lines", utils.LogAttr("log_name", log), utils.LogAttr("line", indexFound))
		} else {
			utils.LavaFormatError("Verification failed for EmergencyMode Start Line in the last 20 lines", nil, utils.LogAttr("log_name", log))
		}
	}
}

func (lt *lavaTest) markEmergencyModeLogsEnd() {
	// Create a copy of logs to avoid holding the lock for too long
	lt.logsMu.RLock()

	logsCopy := make(map[string]*sdk.SafeBuffer)
	for k, v := range lt.logs {
		logsCopy[k] = v
	}
	lt.logsMu.RUnlock()

	for log, buffer := range logsCopy {
		utils.LavaFormatInfo("Adding EmergencyMode End Line to", utils.LogAttr("log_name", log))
		_, err := buffer.WriteString(EmergencyModeEndLine + "\n")
		if err != nil {
			utils.LavaFormatError("Failed Writing to buffer", err, utils.LogAttr("key", log))
		}
	}
}

func (lt *lavaTest) stopLava() {
	// we mark the start of the emergency mode
	// as from this line forward connection errors to the node can happen
	// and it makes sense as we are shutting down the node to activate emergency mode
	// but we don't want to fail the test
	lt.markEmergencyModeLogsStart()

	utils.LavaFormatInfo("stopLava: marking daemon exit as expected")
	lt.expectCommandExit(startLavaDaemonLogName)
	cmd := exec.Command("killall", "lavad")
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	utils.LavaFormatInfo("stopLava" + " OK")
}

func (lt *lavaTest) getLatestBlockTime() time.Time {
	cmd := exec.Command(lt.lavadPath, "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
	}

	jsonString := strings.TrimSpace(string(output))

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonString), &data); err != nil {
		panic(err)
	}

	latestBlockRawTime, ok := data["SyncInfo"].(map[string]interface{})["latest_block_time"].(string)
	if !ok {
		panic("failed to get latest block time")
	}

	latestBlockTime, err := time.Parse("2006-01-02T15:04:05Z", latestBlockRawTime)
	if err != nil {
		panic(err)
	}

	utils.LavaFormatInfo("getLastBlockTime" + " OK")

	return latestBlockTime
}

func (lt *lavaTest) checkResponse(tendermintConsumerURL string, restConsumerURL string, grpcConsumerURL string) error {
	utils.LavaFormatInfo("Starting Relay Response Integrity Tests")

	// TENDERMINT:
	tendermintNodeURL := "http://0.0.0.0:26657"
	errors := []string{}
	apiMethodTendermint := "%s/block?height=1"

	providerReply, err := getRequest(fmt.Sprintf(apiMethodTendermint, tendermintConsumerURL))
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s", err))
	} else if strings.Contains(string(providerReply), "error") {
		errors = append(errors, string(providerReply))
	}
	//
	nodeReply, err := getRequest(fmt.Sprintf(apiMethodTendermint, tendermintNodeURL))
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s", err))
	} else if strings.Contains(string(nodeReply), "error") {
		errors = append(errors, string(nodeReply))
	}
	//
	if !bytes.Equal(providerReply, nodeReply) {
		errors = append(errors, "tendermint relay response integrity error!")
	} else {
		utils.LavaFormatInfo("TENDERMINT RESPONSE CROSS VERIFICATION OK")
	}

	// REST:
	restNodeURL := "http://0.0.0.0:1317"
	apiMethodRest := "%s/cosmos/base/tendermint/v1beta1/blocks/1"

	providerReply, err = getRequest(fmt.Sprintf(apiMethodRest, restConsumerURL))
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s", err))
	} else if strings.Contains(string(providerReply), "error") {
		errors = append(errors, string(providerReply))
	}
	//
	nodeReply, err = getRequest(fmt.Sprintf(apiMethodRest, restNodeURL))
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s", err))
	} else if strings.Contains(string(nodeReply), "error") {
		errors = append(errors, string(nodeReply))
	}
	//
	if !bytes.Equal(providerReply, nodeReply) {
		errors = append(errors, "rest relay response integrity error!")
	} else {
		utils.LavaFormatInfo("REST RESPONSE CROSS VERIFICATION OK")
	}

	// gRPC:
	grpcNodeURL := "127.0.0.1:9090"

	grpcConnProvider, err := grpc.Dial(grpcConsumerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errors = append(errors, "error client dial")
	}
	pairingQueryClient := pairingTypes.NewQueryClient(grpcConnProvider)
	if err != nil {
		errors = append(errors, err.Error())
	}
	grpcProviderReply, err := pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
		ChainID: "LAV1",
	})
	if err != nil {
		errors = append(errors, err.Error())
	}

	//
	grpcConnNode, err := grpc.Dial(grpcNodeURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errors = append(errors, "error client dial")
	}
	pairingQueryClient = pairingTypes.NewQueryClient(grpcConnNode)
	if err != nil {
		errors = append(errors, err.Error())
	}
	grpcNodeReply, err := pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
		ChainID: "LAV1",
	})
	if err != nil {
		errors = append(errors, err.Error())
	}
	//

	if strings.TrimSpace(grpcNodeReply.String()) != strings.TrimSpace(grpcProviderReply.String()) {
		errors = append(errors, "grpc relay response integrity error!")
	} else {
		utils.LavaFormatInfo("GRPC RESPONSE CROSS VERIFICATION OK")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ",\n"))
	}
	return nil
}

func (lt *lavaTest) getKeyAddress(key string) string {
	cmd := exec.Command(lt.lavadPath, "keys", "show", key, "-a")

	output, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("could not get %s address %s", key, err.Error()))
	}

	return string(output)
}

func (lt *lavaTest) runWebSocketSubscriptionTest(tendermintConsumerWebSocketURL string) {
	utils.LavaFormatInfo("Starting WebSocket Subscription Test")

	subscriptionsCount := 5

	createWebSocketClient := func() *websocket.Conn {
		websocketDialer := websocket.Dialer{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		header := make(http.Header)

		webSocketClient, resp, err := websocketDialer.DialContext(context.Background(), tendermintConsumerWebSocketURL, header)
		if err != nil {
			panic(err)
		}
		utils.LavaFormatDebug("Dialed WebSocket Successful",
			utils.LogAttr("url", tendermintConsumerWebSocketURL),
			utils.LogAttr("response", resp),
		)

		return webSocketClient
	}

	const (
		SUBSCRIBE   = "subscribe"
		UNSUBSCRIBE = "unsubscribe"
	)

	createSubscriptionJsonRpcMessage := func(method string) map[string]interface{} {
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  method,
			"id":      1,
			"params": map[string]interface{}{
				"query": "tm.event = 'NewBlock'",
			},
		}
	}

	subscribeToNewBlockEvents := func(webSocketClient *websocket.Conn) {
		msgData := createSubscriptionJsonRpcMessage(SUBSCRIBE)
		err := webSocketClient.WriteJSON(msgData)
		if err != nil {
			panic(err)
		}
	}

	webSocketShouldListen := true
	defer func() {
		webSocketShouldListen = false
	}()

	type subscriptionContainer struct {
		newBlockMessageCount int32
		webSocketClient      *websocket.Conn
	}

	incrementNewBlockMessageCount := func(sc *subscriptionContainer) {
		atomic.AddInt32(&sc.newBlockMessageCount, 1)
	}

	readNewBlockMessageCount := func(subscriptionContainer *subscriptionContainer) int32 {
		return atomic.LoadInt32(&subscriptionContainer.newBlockMessageCount)
	}

	startWebSocketReader := func(webSocketName string, webSocketClient *websocket.Conn, subscriptionContainer *subscriptionContainer) {
		for {
			_, message, err := webSocketClient.ReadMessage()
			if err != nil {
				if webSocketShouldListen {
					panic(err)
				}

				// Once the test is done, we can safely ignore the error
				return
			}

			if strings.Contains(string(message), "NewBlock") {
				incrementNewBlockMessageCount(subscriptionContainer)
			}
		}
	}

	startSubscriptions := func(count int) []*subscriptionContainer {
		subscriptionContainers := []*subscriptionContainer{}
		// Start a websocket clients and connect them to tendermint consumer endpoint
		for i := 0; i < count; i++ {
			utils.LavaFormatInfo("Setting up web socket client " + strconv.Itoa(i+1))
			webSocketClient := createWebSocketClient()

			subscriptionContainer := &subscriptionContainer{
				webSocketClient:      webSocketClient,
				newBlockMessageCount: 0,
			}

			// Start a reader for each client to count the number of NewBlock messages received
			utils.LavaFormatInfo("Start listening for NewBlock messages on web socket " + strconv.Itoa(i+1))

			go startWebSocketReader("webSocketClient"+strconv.Itoa(i+1), webSocketClient, subscriptionContainer)

			// Subscribe to new block events
			utils.LavaFormatInfo("Subscribing to NewBlock events on web socket " + strconv.Itoa(i+1))

			subscribeToNewBlockEvents(webSocketClient)
			subscriptionContainers = append(subscriptionContainers, subscriptionContainer)
		}

		return subscriptionContainers
	}

	subscriptions := startSubscriptions(subscriptionsCount)

	// Wait for 10 blocks
	utils.LavaFormatInfo("Sleeping for 12 seconds to receive blocks")
	time.Sleep(12 * time.Second)

	utils.LavaFormatDebug("Looping through subscription containers",
		utils.LogAttr("subscriptionContainers", subscriptions),
	)
	// Check the all web socket clients received at least 10 blocks
	for i := 0; i < subscriptionsCount; i++ {
		utils.LavaFormatInfo("Making sure both clients received at least 10 blocks")
		if subscriptions[i] == nil {
			panic("subscriptionContainers[" + strconv.Itoa(i+1) + "] is nil")
		}
		newBlockMessageCount := readNewBlockMessageCount(subscriptions[i])
		if newBlockMessageCount < 10 {
			panic(fmt.Sprintf("subscription should have received at least 10 blocks, got: %d", newBlockMessageCount))
		}
	}

	// Unsubscribe one client
	utils.LavaFormatInfo("Unsubscribing from NewBlock events on web socket 1")
	msgData := createSubscriptionJsonRpcMessage(UNSUBSCRIBE)
	err := subscriptions[0].webSocketClient.WriteJSON(msgData)
	if err != nil {
		panic(err)
	}

	// Make sure that the unsubscribed client stops receiving blocks
	webSocketClient1NewBlockMsgCountAfterUnsubscribe := readNewBlockMessageCount(subscriptions[0])

	utils.LavaFormatInfo("Sleeping for 7 seconds to make sure unsubscribed client stops receiving blocks")
	time.Sleep(7 * time.Second)

	if readNewBlockMessageCount(subscriptions[0]) != webSocketClient1NewBlockMsgCountAfterUnsubscribe {
		panic("unsubscribed client should not receive new blocks")
	}

	webSocketShouldListen = false

	// Disconnect all websocket clients
	for i := 0; i < subscriptionsCount; i++ {
		utils.LavaFormatInfo("Closing web socket " + strconv.Itoa(i+1))
		subscriptions[i].webSocketClient.Close()
	}

	utils.LavaFormatInfo("WebSocket Subscription Test OK")
}

func calculateProviderCU(pairingClient pairingTypes.QueryClient) (map[string]uint64, error) {
	providerCU := make(map[string]uint64)
	res, err := pairingClient.ProvidersEpochCu(context.Background(), &pairingTypes.QueryProvidersEpochCuRequest{})
	if err != nil {
		return nil, err
	}

	for _, info := range res.Info {
		providerCU[info.Provider] = info.Cu
	}
	return providerCU, nil
}

func runProtocolE2E(timeout time.Duration) {
	os.RemoveAll(protocolLogsFolder)
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	grpcConn, err := grpc.Dial("127.0.0.1:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// Just log because grpc redials
		fmt.Println(err)
	}
	lt := &lavaTest{
		grpcConn:             grpcConn,
		lavadPath:            gopath + "/bin/lavad",
		protocolPath:         gopath + "/bin/lavap",
		lavadArgs:            "--geolocation 1 --log_level debug",
		consumerArgs:         " --allow-insecure-provider-dialing",
		logs:                 make(map[string]*sdk.SafeBuffer),
		commands:             make(map[string]*exec.Cmd),
		expectedCommandExit:  make(map[string]bool),
		providerType:         make(map[string][]epochStorageTypes.Endpoint),
		logPath:              protocolLogsFolder,
		tokenDenom:           commonconsts.TestTokenDenom,
		consumerCacheAddress: "127.0.0.1:2778",
		providerCacheAddress: "127.0.0.1:2777",
	}

	// Ensure cleanup happens
	defer func() {
		// Kill all processes first (before waiting for goroutines)
		// This ensures orphan processes like proxy.test are cleaned up
		// even if the test panics or fails before reaching finishTestSuccessfully()
		lt.commandsMu.RLock()
		for name, cmd := range lt.commands {
			if cmd != nil && cmd.Process != nil {
				utils.LavaFormatInfo("Cleanup: Killing process", utils.LogAttr("name", name))

				// Kill the entire process group to ensure child processes are also terminated
				pgid, err := syscall.Getpgid(cmd.Process.Pid)
				if err == nil {
					// Kill the process group (negative PID kills the group)
					if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
						utils.LavaFormatWarning("Cleanup: Failed to kill process group, falling back to single process", err,
							utils.LogAttr("name", name), utils.LogAttr("pgid", pgid))
						// Fallback to killing just the process
						if err := cmd.Process.Kill(); err != nil {
							utils.LavaFormatError("Cleanup: Failed to kill process", err, utils.LogAttr("name", name))
						}
					}
				} else {
					// If we can't get the process group, just kill the process
					if err := cmd.Process.Kill(); err != nil {
						utils.LavaFormatError("Cleanup: Failed to kill process", err, utils.LogAttr("name", name))
					}
				}
			}
		}
		lt.commandsMu.RUnlock()

		// Wait for all goroutines with timeout
		done := make(chan struct{})
		go func() {
			lt.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			utils.LavaFormatInfo("All goroutines finished cleanly")
		case <-time.After(5 * time.Second):
			utils.LavaFormatWarning("Some goroutines did not finish in time", nil)
		}

		// Close gRPC connection
		if grpcConn != nil {
			if err := grpcConn.Close(); err != nil {
				utils.LavaFormatError("Failed to close gRPC connection", err)
			}
		}

		// Save logs and handle panic
		if r := recover(); r != nil {
			lt.saveLogs()
			panic(fmt.Sprintf("E2E Failed: %v", r))
		} else {
			lt.saveLogs()
		}
	}()

	utils.LavaFormatInfo("Starting Lava")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go lt.startLava(ctx)
	lt.checkLava(timeout)
	utils.LavaFormatInfo("Starting Lava OK")

	utils.LavaFormatInfo("Staking Lava")
	lt.stakeLava(ctx)

	// scripts/test/init_e2e.sh will:
	// - produce 4 specs: ETH1, HOL1, SEP1, IBC, TENDERMINT ,COSMOSSDK, LAV1 (via {ethereum,cosmoshub,lava})
	// - produce 2 plans: "DefaultPlan", "EmergencyModePlan"
	// - produce 5 staked providers (for each of ETH1, LAV1)
	// - produce 1 staked client (for each of ETH1, LAV1)
	// - produce 1 subscription (for both ETH1, LAV1)

	lt.checkStakeLava(2, NumberOfSpecsExpectedInE2E, 4, 5, checkedPlansE2E, checkedSpecsE2E, checkedSubscriptions, "Staking Lava OK")

	utils.LavaFormatInfo("RUNNING TESTS")

	// hereinafter:
	// run each consumer test once for each client/user (staked or subscription)

	// repeat() is a helper to run a given function once per client, passing the
	// iteration (client) number to the function
	repeat := func(n int, f func(int)) {
		for i := 1; i <= n; i++ {
			f(i)
		}
	}

	// ETH1 flow
	lt.startJSONRPCProxy(ctx)
	// checks proxy.
	lt.checkJSONRPCConsumer("http://127.0.0.1:1111", time.Minute*2, "JSONRPCProxy OK")

	// Start json provider
	lt.startJSONRPCProvider(ctx)
	// Start Lava provider
	lt.startLavaProviders(ctx)
	// Wait for providers to finish adding all chain routers.
	if err := contextSleep(ctx, time.Second*7); err != nil {
		utils.LavaFormatWarning("Context cancelled while waiting for providers", err)
	}

	lt.startJSONRPCConsumer(ctx)
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:333%d", n)
		msg := fmt.Sprintf("JSONRPCConsumer%d OK", n)
		lt.checkJSONRPCConsumer(url, time.Minute*2, msg)
	})

	lt.startLavaConsumer(ctx)

	runChecksAndTests := func() {
		// staked client then with subscription
		repeat(1, func(n int) {
			url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
			lt.checkTendermintConsumer(url, time.Second*15)
			url = fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3+1)
			lt.checkRESTConsumer(url, time.Second*15)
			url = fmt.Sprintf("127.0.0.1:334%d", (n-1)*3+2)
			lt.checkGRPCConsumer(url, time.Second*15)
		})

		// staked client then with subscription
		repeat(1, func(n int) {
			url := fmt.Sprintf("http://127.0.0.1:333%d", n)
			if err := jsonrpcTests(url, time.Second*15); err != nil {
				panic(err)
			}
		})
		utils.LavaFormatInfo("JSONRPC TEST OK")

		// staked client then with subscription
		repeat(1, func(n int) {
			url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
			if err := tendermintTests(url, time.Second*15); err != nil {
				panic(err)
			}
		})
		utils.LavaFormatInfo("TENDERMINTRPC TEST OK")

		// staked client then with subscription
		repeat(1, func(n int) {
			url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
			if err := tendermintURITests(url, time.Second*15); err != nil {
				panic(err)
			}
		})
		utils.LavaFormatInfo("TENDERMINTRPC URI TEST OK")

		// staked client then with subscription
		repeat(1, func(n int) {
			url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3+1)
			if err := restTests(url, time.Second*15); err != nil {
				panic(err)
			}
		})
		utils.LavaFormatInfo("REST TEST OK")

		// staked client then with subscription
		repeat(1, func(n int) {
			url := fmt.Sprintf("127.0.0.1:334%d", (n-1)*3+2)
			if err := grpcTests(url, time.Second*15); err != nil {
				panic(err)
			}
		})
		utils.LavaFormatInfo("GRPC TEST OK")
	}

	// run tests without cache
	runChecksAndTests()

	lt.startConsumerCache(ctx)
	lt.startProviderCache(ctx)

	// run tests with cache
	runChecksAndTests()

	lt.lavaOverLava(ctx)

	lt.checkResponse("http://127.0.0.1:3340", "http://127.0.0.1:3341", "127.0.0.1:3342")

	lt.runWebSocketSubscriptionTest("ws://127.0.0.1:3340/websocket")

	lt.checkQoS()

	utils.LavaFormatInfo("Sleeping Until All Rewards are collected")
	lt.sleepUntilNextEpoch()
	lt.sleepUntilNextEpoch()
	lt.sleepUntilNextEpoch()
	lt.sleepUntilNextEpoch()

	// emergency mode
	utils.LavaFormatInfo("Restarting lava to emergency mode")
	lt.stopLava()
	go lt.startLavaInEmergencyMode(ctx, 100000)

	lt.checkLava(timeout)
	utils.LavaFormatInfo("Starting Lava OK")

	// set in init_chain.sh
	var epochDuration int64 = 20 * 1.2

	signalChannel := make(chan bool)
	url := "http://127.0.0.1:3347"

	lt.startLavaEmergencyConsumer(ctx)
	lt.checkRESTConsumer(url, time.Second*30)

	latestBlockTime := lt.getLatestBlockTime()

	// Create a context for the virtual epoch goroutine so it can be cancelled
	epochCtx, epochCancel := context.WithCancel(ctx)
	defer epochCancel() // Ensure the goroutine is stopped when test finishes

	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.LavaFormatError("Panic in virtual epoch goroutine", fmt.Errorf("%v", r))
			}
		}()

		epochCounter := (time.Now().Unix() - latestBlockTime.Unix()) / epochDuration

		for {
			nextEpochTime := latestBlockTime.Add(time.Second * time.Duration(epochDuration*(epochCounter+1)))
			sleepDuration := time.Until(nextEpochTime)

			select {
			case <-epochCtx.Done():
				utils.LavaFormatInfo("Virtual epoch goroutine cancelled")
				return
			case <-time.After(sleepDuration):
				utils.LavaFormatInfo(fmt.Sprintf("%d : VIRTUAL EPOCH ENDED", epochCounter))
				epochCounter++

				// Non-blocking send to avoid goroutine leak if nobody is listening
				select {
				case signalChannel <- true:
				case <-epochCtx.Done():
					return
				default:
					// Channel full or no receiver, just continue
				}
			}
		}
	}()

	// we should have approximately (numOfProviders * epoch_cu_limit * 4) CU
	// skip 1st epoch and 2 virtual epochs
	repeat(3, func(m int) {
		<-signalChannel
	})

	// check that there was an increase CU due to virtual epochs
	// 10 requests is sufficient to validate emergency mode CU allocation
	repeat(10, func(m int) {
		if err := restRelayTest(url); err != nil {
			panic(err)
		}
	})

	utils.LavaFormatInfo("All REST relay tests completed successfully")

	lt.markEmergencyModeLogsEnd()

	utils.LavaFormatInfo("REST RELAY TESTS OK")

	lt.finishTestSuccessfully()
}
