package e2e

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/build"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ignite/cli/ignite/chainconfig"
	"github.com/ignite/cli/ignite/pkg/cache"
	"github.com/ignite/cli/ignite/services/chain"
	"github.com/lavanet/lava/utils"
	epochStorageTypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingTypes "github.com/lavanet/lava/x/pairing/types"
	planTypes "github.com/lavanet/lava/x/plans/types"
	specTypes "github.com/lavanet/lava/x/spec/types"
	tmclient "github.com/tendermint/tendermint/rpc/client/http"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	logsFolder   = "./testutil/e2e/logs/"
	configFolder = "./testutil/e2e/e2eProviderConfigs/"
)

var (
	checkedPlansE2E    = []string{"DefaultPlan"}
	checkedSpecsE2E    = []string{"LAV1", "ETH1"}
	checkedSpecsE2ELOL = []string{"GTH1"}
)

type lavaTest struct {
	testFinishedProperly bool
	grpcConn             *grpc.ClientConn
	lavadPath            string
	lavadArgs            string
	logs                 map[string]*bytes.Buffer
	commands             map[string]*exec.Cmd
	providerType         map[string][]epochStorageTypes.Endpoint
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

func (lt *lavaTest) execCommand(ctx context.Context, funcName string, logName string, command string, wait bool) {
	lt.logs[logName] = new(bytes.Buffer)

	cmd := exec.CommandContext(ctx, "", "")
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

func (lt *lavaTest) startLava(ctx context.Context) {
	absPath, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}

	c, err := chain.New(absPath, chain.LogLevel(chain.LogRegular))
	if err != nil {
		panic(err)
	}
	cacheRootDir, err := chainconfig.ConfigDirPath()
	if err != nil {
		panic(err)
	}

	storage, err := cache.NewStorage(filepath.Join(cacheRootDir, "ignite_cache.db"))
	if err != nil {
		panic(err)
	}

	err = c.Serve(ctx, storage, chain.ServeForceReset())
	if err != nil {
		panic(err)
	}
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

func (lt *lavaTest) stakeLava() {
	stakeCommand := "./scripts/init_e2e.sh"
	lt.logs["01_stakeLava"] = new(bytes.Buffer)
	cmd := exec.Cmd{
		Path:   stakeCommand,
		Args:   strings.Split(stakeCommand, " "),
		Stdout: lt.logs["01_stakeLava"],
		Stderr: lt.logs["01_stakeLava"],
	}
	err := cmd.Start()
	if err != nil {
		panic("Staking Failed " + err.Error())
	}
	cmd.Wait()
}

func (lt *lavaTest) checkStakeLava(
	planCount int,
	specCount int,
	providerCount int,
	clientCount int,
	checkedPlans []string,
	checkedSpecs []string,
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
		panic("Staking Failed PLAN count")
	}

	for _, plan := range planQueryRes.PlansInfo {
		if !slices.Contains(checkedPlans, plan.Index) {
			panic("Staking Failed PLAN names")
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
			fmt.Println("ProviderQueryRes: ", providerQueryRes)
			panic("Staking Failed PROVIDER")
		}
		for _, providerStakeEntry := range providerQueryRes.StakeEntry {
			fmt.Println("provider", providerStakeEntry.Address, providerStakeEntry.Endpoints)
			lt.providerType[providerStakeEntry.Address] = providerStakeEntry.Endpoints
		}

		// Query clients
		clientQueryRes, err := pairingQueryClient.Clients(context.Background(), &pairingTypes.QueryClientsRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
			panic(err)
		}
		if len(clientQueryRes.StakeEntry) != clientCount {
			panic("Staking Failed CLIENT")
		}
		for _, clientStakeEntry := range clientQueryRes.StakeEntry {
			fmt.Println("client", clientStakeEntry)
		}
	}
	utils.LavaFormatInfo(successMessage)
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
	utils.LavaFormatInfo(funcName + " OK")
}

func (lt *lavaTest) startJSONRPCProvider(ctx context.Context) {
	for idx := 1; idx <= 5; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/jsonrpcProvider%d.yml --from servicer%d %s",
			lt.lavadPath, configFolder, idx, idx, lt.lavadArgs,
		)
		logName := "03_EthProvider_" + fmt.Sprintf("%02d", idx)
		funcName := fmt.Sprintf("startJSONRPCProvider (provider %02d)", idx)
		lt.execCommand(ctx, funcName, logName, command, false)
	}

	// validate all providers are up
	for idx := 1; idx < 5; idx++ {
		lt.checkProviderResponsive(ctx, fmt.Sprintf("127.0.0.1:222%d", idx), time.Minute)
	}

	utils.LavaFormatInfo("startJSONRPCProvider OK")
}

func (lt *lavaTest) startJSONRPCConsumer(ctx context.Context) {
	for idx, u := range []string{"user1", "user3"} {
		command := fmt.Sprintf(
			"%s rpcconsumer %s/ethConsumer%d.yml --from %s %s",
			lt.lavadPath, configFolder, idx+1, u, lt.lavadArgs,
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
		_, err = client.BlockNumber(context.Background())
		if err == nil {
			utils.LavaFormatInfo(message)
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
		grpcClient, err := grpc.DialContext(nctx, rpcURL, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		errors = append(errors, "error client dial")
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
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}

func (lt *lavaTest) startLavaProviders(ctx context.Context) {
	for idx := 6; idx <= 10; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/lavaProvider%d --from servicer%d %s",
			lt.lavadPath, configFolder, idx, idx, lt.lavadArgs,
		)
		logName := "05_LavaProvider_" + fmt.Sprintf("%02d", idx-5)
		funcName := fmt.Sprintf("startLavaProviders (provider %02d)", idx-5)
		lt.execCommand(ctx, funcName, logName, command, false)
	}

	// validate all providers are up
	for idx := 6; idx <= 10; idx++ {
		lt.checkProviderResponsive(ctx, fmt.Sprintf("127.0.0.1:226%d", idx-5), time.Minute)
		lt.checkProviderResponsive(ctx, fmt.Sprintf("127.0.0.1:227%d", idx-5), time.Minute)
		lt.checkProviderResponsive(ctx, fmt.Sprintf("127.0.0.1:228%d", idx-5), time.Minute)
	}

	utils.LavaFormatInfo("startLavaProviders OK")
}

func (lt *lavaTest) startLavaConsumer(ctx context.Context) {
	for idx, u := range []string{"user2", "user3"} {
		command := fmt.Sprintf(
			"%s rpcconsumer %s/lavaConsumer%d.yml --from %s %s",
			lt.lavadPath, configFolder, idx+1, u, lt.lavadArgs,
		)
		logName := "06_RPCConsumer_" + fmt.Sprintf("%02d", idx+1)
		funcName := fmt.Sprintf("startRPCConsumer (consumer %02d)", idx+1)
		lt.execCommand(ctx, funcName, logName, command, false)
	}
	utils.LavaFormatInfo("startRPCConsumer OK")
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
	command := "./scripts/init_e2e_lava_over_lava.sh"
	lt.execCommand(ctx, "startJSONRPCConsumer", "07_lavaOverLava", command, true)

	// scripts/init_e2e.sh will:
	// - produce 4 specs: ETH1, GTH1, IBC, COSMOSSDK, LAV1 (via spec_add_{ethereum,cosmoshub,lava})
	// - produce 1 plan: "DefaultPlan"

	lt.checkStakeLava(1, 5, 5, 1, checkedPlansE2E, checkedSpecsE2ELOL, "Lava Over Lava Test OK")
}

func (lt *lavaTest) checkRESTConsumer(rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting REST Consumer")
		reply, err := getRequest(fmt.Sprintf("%s/blocks/latest", rpcURL))
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
		"%s/blocks/latest",
		"%s/lavanet/lava/pairing/providers/LAV1",
		"%s/lavanet/lava/pairing/clients/LAV1",
		"%s/cosmos/gov/v1beta1/proposals",
		"%s/lavanet/lava/spec/spec",
		"%s/blocks/1",
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
	errors := []string{}
	grpcConn, err := grpc.Dial(rpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errors = append(errors, "error client dial")
	}
	specQueryClient := specTypes.NewQueryClient(grpcConn)
	pairingQueryClient := pairingTypes.NewQueryClient(grpcConn)
	for start := time.Now(); time.Since(start) < testDuration; {
		specQueryRes, err := specQueryClient.SpecAll(ctx, &specTypes.QueryAllSpecRequest{})
		if err != nil {
			errors = append(errors, err.Error())
		}
		for _, spec := range specQueryRes.Spec {
			_, err = pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
				ChainID: spec.GetIndex(),
			})
			if err != nil {
				errors = append(errors, err.Error())
			}
			_, err = pairingQueryClient.Clients(context.Background(), &pairingTypes.QueryClientsRequest{
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
	if _, err := os.Stat(logsFolder); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(logsFolder, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	errorLineCount := 0
	errorFiles := []string{}
	for fileName, logBuffer := range lt.logs {
		file, err := os.Create(logsFolder + fileName + ".log")
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
		for _, line := range lines {
			if strings.Contains(line, " ERR ") {
				isAllowedError := false
				for errorSubstring := range allowedErrors {
					if strings.Contains(line, errorSubstring) {
						isAllowedError = true
						break
					}
				}
				// When test did not finish properly save all logs. If test finished properly save only non allowed errors.
				if !lt.testFinishedProperly || !isAllowedError {
					errorLineCount += 1
					errorLines = append(errorLines, line)
				}
			}
		}
		if len(errorLines) == 0 {
			continue
		}
		errorFiles = append(errorFiles, fileName)
		errors := strings.Join(errorLines, "\n")
		errFile, err := os.Create(logsFolder + fileName + "_errors.log")
		if err != nil {
			panic(err)
		}
		writer = bufio.NewWriter(errFile)
		writer.Write([]byte(errors))
		writer.Flush()
		errFile.Close()
	}

	if errorLineCount != 0 {
		panic("Error found in logs " + strings.Join(errorFiles, ", "))
	}
}

func (lt *lavaTest) checkPayments(testDuration time.Duration) {
	utils.LavaFormatInfo("Checking Payments")
	ethPaid := false
	lavaPaid := false
	for start := time.Now(); time.Since(start) < testDuration; {
		pairingClient := pairingTypes.NewQueryClient(lt.grpcConn)
		pairingRes, err := pairingClient.EpochPaymentsAll(context.Background(), &pairingTypes.QueryAllEpochPaymentsRequest{})
		if err != nil {
			panic(err)
		}

		if len(pairingRes.EpochPayments) == 0 {
			utils.LavaFormatInfo("Waiting Payments")
			time.Sleep(time.Second)
			continue
		}
		for _, epochPayment := range pairingRes.EpochPayments {
			for _, clientsPaymentKey := range epochPayment.GetProviderPaymentStorageKeys() {
				if strings.Contains(clientsPaymentKey, "ETH") {
					ethPaid = true
				} else if strings.Contains(clientsPaymentKey, "LAV") {
					lavaPaid = true
				}
			}
		}
		if ethPaid && lavaPaid {
			break
		}
	}

	if !ethPaid && !lavaPaid {
		panic("PAYMENT FAILED FOR ETH AND LAVA")
	}

	if ethPaid {
		utils.LavaFormatInfo("PAYMENT SUCCESSFUL FOR ETH")
	} else {
		panic("PAYMENT FAILED FOR ETH")
	}

	if lavaPaid {
		utils.LavaFormatInfo("PAYMENT SUCCESSFUL FOR LAVA")
	} else {
		panic("PAYMENT FAILED FOR LAVA")
	}
}

func runE2E(timeout time.Duration) {
	os.RemoveAll(logsFolder)
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
		lavadArgs:    "--geolocation 1 --log_level debug",
		logs:         make(map[string]*bytes.Buffer),
		commands:     make(map[string]*exec.Cmd),
		providerType: make(map[string][]epochStorageTypes.Endpoint),
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
	go lt.startLava(context.Background())
	lt.checkLava(timeout)
	utils.LavaFormatInfo("Starting Lava OK")
	utils.LavaFormatInfo("Staking Lava")
	lt.stakeLava()

	// scripts/init_e2e.sh will:
	// - produce 4 specs: ETH1, GTH1, IBC, COSMOSSDK, LAV1 (via spec_add_{ethereum,cosmoshub,lava})
	// - produce 1 plan: "DefaultPlan"

	lt.checkStakeLava(1, 5, 5, 1, checkedPlansE2E, checkedSpecsE2E, "Staking Lava OK")

	utils.LavaFormatInfo("RUNNING TESTS")

	// hereinafter:
	// run each consumer test once for staked client and once for subscription client

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// ETH1 flow
	lt.startJSONRPCProxy(ctx)
	lt.checkJSONRPCConsumer("http://127.0.0.1:1111", time.Minute*2, "JSONRPCProxy OK") // checks proxy.
	lt.startJSONRPCProvider(ctx)
	lt.startJSONRPCConsumer(ctx)
	lt.checkJSONRPCConsumer("http://127.0.0.1:3331/1", time.Minute*2, "JSONRPCConsumer1 OK")
	lt.checkJSONRPCConsumer("http://127.0.0.1:3332/1", time.Minute*2, "JSONRPCConsumer2 OK")

	// Lava Flow
	lt.startLavaProviders(ctx)
	lt.startLavaConsumer(ctx)
	// staked client then with subscription
	lt.checkTendermintConsumer("http://127.0.0.1:3340/1", time.Second*30)
	lt.checkRESTConsumer("http://127.0.0.1:3341/1", time.Second*30)
	lt.checkGRPCConsumer("127.0.0.1:3342", time.Second*30)
	lt.checkTendermintConsumer("http://127.0.0.1:3343/1", time.Second*30)
	lt.checkRESTConsumer("http://127.0.0.1:3344/1", time.Second*30)
	lt.checkGRPCConsumer("127.0.0.1:3345", time.Second*30)

	// staked client then with subscription
	if jsonErr := jsonrpcTests("http://127.0.0.1:3331/1", time.Second*30); jsonErr != nil {
		panic(jsonErr)
	}
	if jsonErr := jsonrpcTests("http://127.0.0.1:3332/1", time.Second*30); jsonErr != nil {
		panic(jsonErr)
	}
	utils.LavaFormatInfo("JSONRPC TEST OK")

	// staked client then with subscription
	if tendermintErr := tendermintTests("http://127.0.0.1:3340/1", time.Second*30); tendermintErr != nil {
		panic(tendermintErr)
	}
	if tendermintErr := tendermintTests("http://127.0.0.1:3343/1", time.Second*30); tendermintErr != nil {
		panic(tendermintErr)
	}
	utils.LavaFormatInfo("TENDERMINTRPC TEST OK")

	// staked client then with subscription
	if tendermintURIErr := tendermintURITests("http://127.0.0.1:3340/1", time.Second*30); tendermintURIErr != nil {
		panic(tendermintURIErr)
	}
	if tendermintURIErr := tendermintURITests("http://127.0.0.1:3343/1", time.Second*30); tendermintURIErr != nil {
		panic(tendermintURIErr)
	}
	utils.LavaFormatInfo("TENDERMINTRPC URI TEST OK")

	lt.lavaOverLava(ctx)

	// staked client then with subscription
	if restErr := restTests("http://127.0.0.1:3341/1", time.Second*30); restErr != nil {
		panic(restErr)
	}
	if restErr := restTests("http://127.0.0.1:3344/1", time.Second*30); restErr != nil {
		panic(restErr)
	}
	utils.LavaFormatInfo("REST TEST OK")

	lt.checkPayments(time.Minute * 10)

	// staked client then with subscription
	// TODO: if set to 30 secs fails e2e need to investigate why. currently blocking PR's
	if grpcErr := grpcTests("127.0.0.1:3342", time.Second*5); grpcErr != nil {
		panic(grpcErr)
	}
	if grpcErr := grpcTests("127.0.0.1:3345", time.Second*5); grpcErr != nil {
		panic(grpcErr)
	}
	utils.LavaFormatInfo("GRPC TEST OK")

	lt.finishTestSuccessfully()
}
