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
	"time"

	tmclient "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	commonconsts "github.com/lavanet/lava/v2/testutil/common/consts"
	"github.com/lavanet/lava/v2/testutil/e2e/sdk"
	"github.com/lavanet/lava/v2/utils"
	epochStorageTypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	pairingTypes "github.com/lavanet/lava/v2/x/pairing/types"
	planTypes "github.com/lavanet/lava/v2/x/plans/types"
	specTypes "github.com/lavanet/lava/v2/x/spec/types"
	subscriptionTypes "github.com/lavanet/lava/v2/x/subscription/types"
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
)

var (
	checkedPlansE2E         = []string{"DefaultPlan", "EmergencyModePlan"}
	checkedSubscriptions    = []string{"user1", "user2", "user3", "user5"}
	checkedSpecsE2E         = []string{"LAV1", "ETH1"}
	checkedSpecsE2ELOL      = []string{"SEP1"}
	checkedSubscriptionsLOL = []string{"user4"}
)

type lavaTest struct {
	testFinishedProperly bool
	grpcConn             *grpc.ClientConn
	lavadPath            string
	protocolPath         string
	lavadArgs            string
	consumerArgs         string
	logs                 map[string]*sdk.SafeBuffer
	commands             map[string]*exec.Cmd
	providerType         map[string][]epochStorageTypes.Endpoint
	wg                   sync.WaitGroup
	logPath              string
	tokenDenom           string
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

func (lt *lavaTest) execCommandWithRetry(ctx context.Context, funcName string, logName string, command string) {
	utils.LavaFormatDebug("Executing command " + command)
	lt.logs[logName] = &sdk.SafeBuffer{}

	cmd := exec.CommandContext(ctx, "", "")
	cmd.Args = strings.Fields(command)
	cmd.Path = cmd.Args[0]
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	lt.commands[logName] = cmd
	retries := 0 // Counter for retries
	maxRetries := 2

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Panic occurred:", r)
				if retries < maxRetries {
					retries++

					utils.LavaFormatInfo(fmt.Sprintln("Restarting goroutine for startJSONRPCProvider. Remaining retries: ", maxRetries-retries))
					go lt.execCommandWithRetry(ctx, funcName, logName, command)
				} else {
					panic(errors.New("maximum number of retries exceeded"))
				}
			}
		}()
		lt.listenCmdCommand(cmd, funcName+" process returned unexpectedly", funcName)
	}()
}

func (lt *lavaTest) execCommand(ctx context.Context, funcName string, logName string, command string, wait bool) {
	defer func() {
		if r := recover(); r != nil {
			lt.saveLogs()
			panic(fmt.Sprintf("Panic happened with command: %s", command))
		}
	}()

	lt.logs[logName] = &sdk.SafeBuffer{}

	cmd := exec.CommandContext(ctx, "", "")
	utils.LavaFormatInfo("Executing Command: " + command)
	cmd.Args = strings.Fields(command)
	cmd.Path = cmd.Args[0]
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	if wait {
		if err = cmd.Wait(); err != nil {
			panic(funcName + " failed " + err.Error())
		}
	} else {
		lt.commands[logName] = cmd
		go func() {
			lt.listenCmdCommand(cmd, funcName+" process returned unexpectedly", funcName)
		}()
	}
}

func (lt *lavaTest) listenCmdCommand(cmd *exec.Cmd, panicReason string, functionName string) {
	err := cmd.Wait()
	if err != nil && !lt.testFinishedProperly {
		utils.LavaFormatError(functionName+" cmd wait err", err)
	}
	if lt.testFinishedProperly {
		return
	}
	lt.saveLogs()
	panic(panicReason)
}

const OKstr = " OK"

func (lt *lavaTest) startLava(ctx context.Context) {
	command := "./scripts/start_env_dev.sh"
	logName := "00_StartLava"
	funcName := "startLava"

	lt.execCommand(ctx, funcName, logName, command, true)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) checkLava(timeout time.Duration) {
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

	for start := time.Now(); time.Since(start) < timeout; {
		// This loop would wait for the lavad server to be up before chain init
		_, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
		if err != nil && strings.Contains(err.Error(), "rpc error") {
			utils.LavaFormatInfo("Waiting for Lava")
			time.Sleep(time.Second * 10)
		} else if err == nil {
			return
		} else {
			panic(err)
		}
	}
	panic("Lava Check Failed")
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
		panic("Staking Failed PLAN count" + fmt.Sprintf("expected %d, got %d", planCount, len(planQueryRes.PlansInfo)))
	}

	for _, plan := range planQueryRes.PlansInfo {
		if !slices.Contains(checkedPlans, plan.Index) {
			panic("Staking Failed PLAN names")
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
		panic("Staking Failed SUBSCRIPTION count")
	}

	for _, key := range checkedSubscriptions {
		subscriptionQueryClient := subscriptionTypes.NewQueryClient(lt.grpcConn)
		_, err = subscriptionQueryClient.Current(context.Background(), &subscriptionTypes.QueryCurrentRequest{Consumer: lt.getKeyAddress(key)})
		if err != nil {
			panic("could not get the subscription of " + key)
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
		utils.LavaFormatError("Spec missing", nil, utils.LogAttr("have", len(specQueryRes.Spec)), utils.LogAttr("want", specCount))
		panic("Staking Failed SPEC")
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
			fmt.Println("ProviderQueryRes: ", providerQueryRes.String())
			panic(fmt.Errorf("staking Failed PROVIDER count %d, wanted %d", len(providerQueryRes.StakeEntry), providerCount))
		}
		for _, providerStakeEntry := range providerQueryRes.StakeEntry {
			fmt.Println("provider", providerStakeEntry.Address, providerStakeEntry.Endpoints)
			lt.providerType[providerStakeEntry.Address] = providerStakeEntry.Endpoints
		}
	}
	utils.LavaFormatInfo(successMessage)
}

func (lt *lavaTest) checkBadgeServerResponsive(ctx context.Context, badgeServerAddr string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting for Badge Server " + badgeServerAddr)
		nctx, cancel := context.WithTimeout(ctx, time.Second)

		grpcClient, err := grpc.DialContext(nctx, badgeServerAddr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			cancel()
			time.Sleep(time.Second)
			continue
		}
		cancel()
		grpcClient.Close()
		return
	}
	panic("checkBadgeServerResponsive: Check Failed. Badge server didn't respond on " + badgeServerAddr)
}

func (lt *lavaTest) startJSONRPCProxy(ctx context.Context) {
	goExecutablePath, err := exec.LookPath("go")
	if err != nil {
		panic("Could not find go executable path")
	}
	// force go's test timeout to 0, otherwise the default is 10m; our timeout
	// will be enforced by the given ctx.
	command := goExecutablePath + " test ./testutil/e2e/proxy/. -v -timeout 0 eth"
	logName := "02_jsonProxy"
	funcName := "startJSONRPCProxy"

	lt.execCommand(ctx, funcName, logName, command, false)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) startJSONRPCProvider(ctx context.Context) {
	for idx := 1; idx <= 5; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/jsonrpcProvider%d.yml --chain-id=lava --from servicer%d %s",
			lt.protocolPath, providerConfigsFolder, idx, idx, lt.lavadArgs,
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
			"%s rpcconsumer %s/ethConsumer%d.yml --chain-id=lava --from %s %s",
			lt.protocolPath, consumerConfigsFolder, idx+1, u, lt.lavadArgs+lt.consumerArgs,
		)
		logName := "04_jsonConsumer_" + fmt.Sprintf("%02d", idx+1)
		funcName := fmt.Sprintf("startJSONRPCConsumer (consumer %02d)", idx+1)
		lt.execCommand(ctx, funcName, logName, command, false)
	}
	utils.LavaFormatInfo("startJSONRPCConsumer OK")
}

// If after timeout and the check does not return it means it failed
func (lt *lavaTest) checkJSONRPCConsumer(rpcURL string, timeout time.Duration, message string) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting JSONRPC Consumer")
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			continue
		}
		res, err := client.BlockNumber(context.Background())
		if err == nil {
			utils.LavaFormatInfo(message)
			utils.LavaFormatInfo("Validated proxy is alive got response", utils.Attribute{Key: "res", Value: res})
			return
		}
		time.Sleep(time.Second)
	}
	panic("checkJSONRPCConsumer: JSONRPC Check Failed Consumer didn't respond")
}

func (lt *lavaTest) checkProviderResponsive(ctx context.Context, rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting Provider " + rpcURL)
		nctx, cancel := context.WithTimeout(ctx, time.Second)
		var tlsConf tls.Config
		tlsConf.InsecureSkipVerify = true // skip CA validation
		credentials := credentials.NewTLS(&tlsConf)
		grpcClient, err := grpc.DialContext(nctx, rpcURL, grpc.WithBlock(), grpc.WithTransportCredentials(credentials))
		if err != nil {
			// utils.LavaFormatInfo(fmt.Sprintf("Provider is still intializing %s", err), nil)
			cancel()
			time.Sleep(time.Second)
			continue
		}
		cancel()
		grpcClient.Close()
		return
	}
	panic("checkProviderResponsive: Check Failed Provider didn't respond" + rpcURL)
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
		var latestBlock *types.Block
		var latestBlockNumber *big.Int
		var latestBlockTxs types.Transactions
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

		targetTxMsg, _ := targetTx.AsMessage(types.LatestSignerForChainID(targetTx.ChainId()), nil)

		// eth_getBalance
		_, err = client.BalanceAt(ctx, targetTxMsg.From(), nil)
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

		previousBlock := big.NewInt(int64(latestBlockNumberUint - 1))

		callMsg := ethereum.CallMsg{
			From:       targetTxMsg.From(),
			To:         targetTxMsg.To(),
			Gas:        targetTxMsg.Gas(),
			GasPrice:   targetTxMsg.GasPrice(),
			GasFeeCap:  targetTxMsg.GasFeeCap(),
			GasTipCap:  targetTxMsg.GasTipCap(),
			Value:      targetTxMsg.Value(),
			Data:       targetTxMsg.Data(),
			AccessList: targetTxMsg.AccessList(),
		}

		// eth_call
		_, err = client.CallContract(ctx, callMsg, previousBlock)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			errors = append(errors, "error JSONRPC_eth_call")
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
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}

func (lt *lavaTest) startLavaProviders(ctx context.Context) {
	for idx := 6; idx <= 10; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/lavaProvider%d --chain-id=lava --from servicer%d %s",
			lt.protocolPath, providerConfigsFolder, idx, idx, lt.lavadArgs,
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
			"%s rpcconsumer %s/lavaConsumer%d.yml --chain-id=lava --from %s %s",
			lt.protocolPath, consumerConfigsFolder, idx+1, u, lt.lavadArgs+lt.consumerArgs,
		)
		logName := "06_RPCConsumer_" + fmt.Sprintf("%02d", idx+1)
		funcName := fmt.Sprintf("startRPCConsumer (consumer %02d)", idx+1)
		lt.execCommand(ctx, funcName, logName, command, false)
	}
	utils.LavaFormatInfo("startRPCConsumer OK")
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
	client, err := tmclient.New(rpcURL, "/websocket")
	if err != nil {
		errors = append(errors, "error client dial")
	}
	for start := time.Now(); time.Since(start) < testDuration; {
		_, err := client.Status(ctx)
		if err != nil {
			errors = append(errors, err.Error())
		}
		_, err = client.Health(ctx)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}
	return nil
}

func tendermintURITests(rpcURL string, testDuration time.Duration) error {
	utils.LavaFormatInfo("Starting TENDERMINTRPC URI Tests")
	errors := []string{}
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
				errors = append(errors, fmt.Sprintf("%s", err))
			} else if strings.Contains(string(reply), "error") && noFail {
				errors = append(errors, string(reply))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
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
	for start := time.Now(); time.Since(start) < testDuration; {
		for _, api := range mostImportantApisToTest {
			reply, err := getRequest(fmt.Sprintf(api, rpcURL))
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s", err))
			} else if strings.Contains(string(reply), "error") {
				errors = append(errors, string(reply))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}
	return nil
}

func restRelayTest(rpcURL string) error {
	errors := []string{}
	apiToTest := "%s/cosmos/base/tendermint/v1beta1/blocks/1"

	reply, err := getRequest(fmt.Sprintf(apiToTest, rpcURL))
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s", err))
	} else if strings.Contains(string(reply), "error") {
		errors = append(errors, string(reply))
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}
	return nil
}

func getRequest(url string) ([]byte, error) {
	res, err := http.Get(url)
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
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}
	return nil
}

func (lt *lavaTest) finishTestSuccessfully() {
	lt.testFinishedProperly = true
	for _, cmd := range lt.commands { // kill all the project commands
		cmd.Process.Kill()
	}
}

func (lt *lavaTest) saveLogs() {
	if _, err := os.Stat(lt.logPath); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(lt.logPath, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	errorFound := false
	errorFiles := []string{}
	errorPrint := make(map[string]string)
	for fileName, logBuffer := range lt.logs {
		reachedEmergencyModeLine := false
		file, err := os.Create(lt.logPath + fileName + ".log")
		if err != nil {
			panic(err)
		}
		writer := bufio.NewWriter(file)
		writer.Write(logBuffer.Bytes())
		writer.Flush()
		utils.LavaFormatDebug("writing file", []utils.Attribute{{Key: "fileName", Value: fileName}, {Key: "lines", Value: len(logBuffer.Bytes())}}...)
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
				if !lt.testFinishedProperly || !isAllowedError {
					errorFound = true
					errorLines = append(errorLines, line)
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
		for _, errLine := range errorPrint {
			fmt.Println("ERROR: ", errLine)
		}
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
		lt.logs[logName] = &sdk.SafeBuffer{}

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
		lt.commands[logName] = cmd
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
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}
	return nil
}

func (lt *lavaTest) startLavaInEmergencyMode(ctx context.Context, timeoutCommit int) {
	command := "./scripts/test/emergency_mode.sh " + strconv.Itoa(timeoutCommit)
	logName := "10_StartLavaInEmergencyMode"
	funcName := "startLavaInEmergencyMode"

	lt.execCommand(ctx, funcName, logName, command, true)
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
	for log, buffer := range lt.logs {
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
	for log, buffer := range lt.logs {
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

	cmd := exec.Command("killall", "lavad")
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	utils.LavaFormatInfo("stopLava" + " OK")
}

func (lt *lavaTest) getLatestBlockTime() time.Time {
	cmd := exec.Command("lavad", "status")
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
		return fmt.Errorf(strings.Join(errors, ",\n"))
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
		grpcConn:     grpcConn,
		lavadPath:    gopath + "/bin/lavad",
		protocolPath: gopath + "/bin/lavap",
		lavadArgs:    "--geolocation 1 --log_level debug",
		consumerArgs: " --allow-insecure-provider-dialing",
		logs:         make(map[string]*sdk.SafeBuffer),
		commands:     make(map[string]*exec.Cmd),
		providerType: make(map[string][]epochStorageTypes.Endpoint),
		logPath:      protocolLogsFolder,
		tokenDenom:   commonconsts.TestTokenDenom,
	}
	// use defer to save logs in case the tests fail
	defer func() {
		if r := recover(); r != nil {
			lt.saveLogs()
			panic("E2E Failed")
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
	lt.checkJSONRPCConsumer("http://127.0.0.1:1111", time.Minute*2, "JSONRPCProxy OK") // checks proxy.
	lt.startJSONRPCProvider(ctx)
	lt.startJSONRPCConsumer(ctx)

	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:333%d", n)
		msg := fmt.Sprintf("JSONRPCConsumer%d OK", n)
		lt.checkJSONRPCConsumer(url, time.Minute*2, msg)
	})

	// Lava Flow
	lt.startLavaProviders(ctx)
	lt.startLavaConsumer(ctx)

	// staked client then with subscription
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		lt.checkTendermintConsumer(url, time.Second*30)
		url = fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3+1)
		lt.checkRESTConsumer(url, time.Second*30)
		url = fmt.Sprintf("127.0.0.1:334%d", (n-1)*3+2)
		lt.checkGRPCConsumer(url, time.Second*30)
	})

	// staked client then with subscription
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:333%d", n)
		if err := jsonrpcTests(url, time.Second*30); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("JSONRPC TEST OK")

	// staked client then with subscription
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		if err := tendermintTests(url, time.Second*30); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("TENDERMINTRPC TEST OK")

	// staked client then with subscription
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		if err := tendermintURITests(url, time.Second*30); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("TENDERMINTRPC URI TEST OK")

	lt.lavaOverLava(ctx)

	// staked client then with subscription
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3+1)
		if err := restTests(url, time.Second*30); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("REST TEST OK")

	// staked client then with subscription
	// TODO: if set to 30 secs fails e2e need to investigate why. currently blocking PR's
	repeat(1, func(n int) {
		url := fmt.Sprintf("127.0.0.1:334%d", (n-1)*3+2)
		if err := grpcTests(url, time.Second*5); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("GRPC TEST OK")

	lt.checkResponse("http://127.0.0.1:3340", "http://127.0.0.1:3341", "127.0.0.1:3342")

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

	go func() {
		epochCounter := (time.Now().Unix() - latestBlockTime.Unix()) / epochDuration

		for {
			time.Sleep(time.Until(latestBlockTime.Add(time.Second * time.Duration(epochDuration*(epochCounter+1)))))
			utils.LavaFormatInfo(fmt.Sprintf("%d : VIRTUAL EPOCH ENDED", epochCounter))

			epochCounter++
			signalChannel <- true
		}
	}()

	utils.LavaFormatInfo("Waiting for finishing current epoch and waiting for 2 more virtual epochs")

	// we should have approximately (numOfProviders * epoch_cu_limit * 4) CU
	// skip 1st epoch and 2 virtual epochs
	repeat(3, func(m int) {
		<-signalChannel
	})

	// check that there was an increase CU due to virtual epochs
	repeat(70, func(m int) {
		if err := restRelayTest(url); err != nil {
			utils.LavaFormatError(fmt.Sprintf("Error while sending relay number %d: ", m), err)
			panic(err)
		}
	})

	lt.markEmergencyModeLogsEnd()

	utils.LavaFormatInfo("REST RELAY TESTS OK")

	lt.finishTestSuccessfully()
}
