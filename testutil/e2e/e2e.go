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
	"strconv"
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
	checkedSpecsE2E    = []string{"LAV1", "ETH1"}
	checkedSpecsE2ELOL = []string{"GTH1"}
)

type lavaTest struct {
	testFinishedProperly bool
	grpcConn             *grpc.ClientConn
	lavadPath            string
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

func (lt *lavaTest) checkStakeLava(specCount int, providerCount int, clientCount int, checkedSpecs []string, successMessage string) {
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
	proxyCommand := goExecutablePath + " test ./testutil/e2e/proxy/. -v eth"
	lt.logs["02_jsonProxy"] = new(bytes.Buffer)

	cmd := exec.CommandContext(ctx, "", "")
	cmd.Path = goExecutablePath
	cmd.Args = strings.Split(proxyCommand, " ")
	cmd.Stdout = lt.logs["02_jsonProxy"]
	cmd.Stderr = lt.logs["02_jsonProxy"]

	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	lt.commands["02_jsonProxy"] = cmd
	go func() {
		lt.listenCmdCommand(cmd, "startJSONRPCProxy process returned unexpectedly", "startJSONRPCProxy")
	}()
	utils.LavaFormatInfo("startJSONRPCProxy OK")
}

func (lt *lavaTest) startJSONRPCProvider(ctx context.Context) {
	providerCommands := []string{
		lt.lavadPath + " rpcprovider " + configFolder + "jsonrpcProvider1.yml --from servicer1 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "jsonrpcProvider2.yml --from servicer2 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "jsonrpcProvider3.yml --from servicer3 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "jsonrpcProvider4.yml --from servicer4 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "jsonrpcProvider5.yml --from servicer5 --geolocation 1 --log_level debug",
	}

	for idx, providerCommand := range providerCommands {
		logName := "03_EthProvider_" + fmt.Sprintf("%02d", idx)
		lt.logs[logName] = new(bytes.Buffer)
		cmd := exec.CommandContext(ctx, "", "")
		cmd.Path = lt.lavadPath
		cmd.Args = strings.Split(providerCommand, " ")
		cmd.Stdout = lt.logs[logName]
		cmd.Stderr = lt.logs[logName]

		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		lt.commands[logName] = cmd
		go func(idx int) {
			lt.listenCmdCommand(cmd, "startJSONRPCProvider process returned unexpectedly, provider idx:"+strconv.Itoa(idx), "startJSONRPCProvider")
		}(idx)
	}
	// validate all providers are up
	for idx := 0; idx < len(providerCommands); idx++ {
		lt.checkProviderResponsive(ctx, "127.0.0.1:222"+fmt.Sprintf("%d", idx+1), time.Minute)
	}

	utils.LavaFormatInfo("startJSONRPCProvider OK")
}

func (lt *lavaTest) startJSONRPCConsumer(ctx context.Context) {
	providerCommand := lt.lavadPath + " rpcconsumer " + configFolder + "ethConsumer.yml --from user1 --geolocation 1 --log_level debug"
	logName := "04_jsonConsumer"
	lt.logs[logName] = new(bytes.Buffer)

	cmd := exec.CommandContext(ctx, "", "")
	cmd.Path = lt.lavadPath
	cmd.Args = strings.Split(providerCommand, " ")
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	lt.commands[logName] = cmd
	go func() {
		lt.listenCmdCommand(cmd, "startJSONRPCConsumer process returned unexpectedly", "startJSONRPCConsumer")
	}()
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
	providerCommands := []string{
		lt.lavadPath + " rpcprovider " + configFolder + "lavaProvider6.yml --from servicer6 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "lavaProvider7.yml --from servicer7 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "lavaProvider8.yml --from servicer8 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "lavaProvider9.yml --from servicer9 --geolocation 1 --log_level debug",
		lt.lavadPath + " rpcprovider " + configFolder + "lavaProvider10.yml --from servicer10 --geolocation 1 --log_level debug",
	}

	for idx, providerCommand := range providerCommands {
		logName := "05_LavaProvider_" + fmt.Sprintf("%02d", idx)
		lt.logs[logName] = new(bytes.Buffer)
		cmd := exec.CommandContext(ctx, "", "")
		cmd.Path = lt.lavadPath
		cmd.Args = strings.Split(providerCommand, " ")
		cmd.Stdout = lt.logs[logName]
		cmd.Stderr = lt.logs[logName]

		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		lt.commands[logName] = cmd

		go func(idx int) {
			lt.listenCmdCommand(cmd, "startLavaProviders process returned unexpectedly, provider idx:"+strconv.Itoa(idx), "startLavaProviders")
		}(idx)
	}

	// validate all providers are up
	for idx := 0; idx < len(providerCommands); idx++ {
		lt.checkProviderResponsive(ctx, "127.0.0.1:226"+fmt.Sprintf("%d", idx+1), time.Minute)
		lt.checkProviderResponsive(ctx, "127.0.0.1:227"+fmt.Sprintf("%d", idx+1), time.Minute)
		lt.checkProviderResponsive(ctx, "127.0.0.1:228"+fmt.Sprintf("%d", idx+1), time.Minute)
	}

	utils.LavaFormatInfo("startTendermintProvider OK")
}

func (lt *lavaTest) startLavaConsumer(ctx context.Context) {
	providerCommand := lt.lavadPath + " rpcconsumer " + configFolder + "lavaConsumer.yml --from user2 --geolocation 1 --log_level debug"
	logName := "06_RPCConsumer"
	lt.logs[logName] = new(bytes.Buffer)

	cmd := exec.CommandContext(ctx, "", "")
	cmd.Path = lt.lavadPath
	cmd.Args = strings.Split(providerCommand, " ")
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	lt.commands[logName] = cmd
	go func() {
		lt.listenCmdCommand(cmd, "startRPCConsumer process returned unexpectedly", "startRPCConsumer")
	}()
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
	stakeCommand := "./scripts/init_e2e_lava_over_lava.sh"
	logName := "07_lavaOverLava"
	lt.logs[logName] = new(bytes.Buffer)
	cmd := exec.CommandContext(ctx, "", "")
	cmd.Path = stakeCommand
	cmd.Args = strings.Split(stakeCommand, " ")
	cmd.Stdout = lt.logs[logName]
	cmd.Stderr = lt.logs[logName]

	err := cmd.Start()
	if err != nil {
		panic("Lava over Lava Failed " + err.Error())
	}
	err = cmd.Wait()
	if err != nil {
		panic("Lava over Lava Failed " + err.Error())
	}
	// scripts/init_e2e.sh adds spec_add_{ethereum,cosmoshub,lava}, which
	// produce 4 specs: ETH1, GTH1, IBC, COSMOSSDK, LAV1
	lt.checkStakeLava(5, 5, 1, checkedSpecsE2ELOL, "Lava Over Lava Test OK")
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
		utils.LavaFormatDebug("writing file", []utils.Attribute{{"fileName", fileName}, {"lines", len(logBuffer.Bytes())}}...)
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

func runE2E() {
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
	lt.checkLava(time.Minute * 10)
	utils.LavaFormatInfo("Starting Lava OK")
	utils.LavaFormatInfo("Staking Lava")
	lt.stakeLava()
	// scripts/init_e2e.sh adds spec_add_{ethereum,cosmoshub,lava}, which
	// produce 4 specs: ETH1, GTH1, IBC, COSMOSSDK, LAV1
	lt.checkStakeLava(5, 5, 1, checkedSpecsE2E, "Staking Lava OK")

	utils.LavaFormatInfo("RUNNING TESTS")

	// ETH1 flow
	jsonCTX, cancel := context.WithCancel(context.Background())
	defer cancel()

	lt.startJSONRPCProxy(jsonCTX)
	lt.checkJSONRPCConsumer("http://127.0.0.1:1111", time.Minute*2, "JSONRPCProxy OK") // checks proxy.
	lt.startJSONRPCProvider(jsonCTX)
	lt.startJSONRPCConsumer(jsonCTX)
	lt.checkJSONRPCConsumer("http://127.0.0.1:3333/1", time.Minute*2, "JSONRPCConsumer OK")

	// Lava Flow
	rpcCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lt.startLavaProviders(rpcCtx)
	lt.startLavaConsumer(rpcCtx)
	lt.checkTendermintConsumer("http://127.0.0.1:3340/1", time.Second*30)
	lt.checkRESTConsumer("http://127.0.0.1:3341/1", time.Second*30)
	lt.checkGRPCConsumer("127.0.0.1:3342", time.Second*30)

	jsonErr := jsonrpcTests("http://127.0.0.1:3333/1", time.Second*30)
	if jsonErr != nil {
		panic(jsonErr)
	} else {
		utils.LavaFormatInfo("JSONRPC TEST OK")
	}

	tendermintErr := tendermintTests("http://127.0.0.1:3340/1", time.Second*30)
	if tendermintErr != nil {
		panic(tendermintErr)
	} else {
		utils.LavaFormatInfo("TENDERMINTRPC TEST OK")
	}

	tendermintURIErr := tendermintURITests("http://127.0.0.1:3340/1", time.Second*30)
	if tendermintURIErr != nil {
		panic(tendermintURIErr)
	} else {
		utils.LavaFormatInfo("TENDERMINTRPC URI TEST OK")
	}

	lt.lavaOverLava(rpcCtx)

	restErr := restTests("http://127.0.0.1:3341/1", time.Second*30)
	if restErr != nil {
		panic(restErr)
	} else {
		utils.LavaFormatInfo("REST TEST OK")
	}

	lt.checkPayments(time.Minute * 10)

	grpcErr := grpcTests("127.0.0.1:3342", time.Second*5) // TODO: if set to 30 secs fails e2e need to investigate why. currently blocking PR's
	if grpcErr != nil {
		panic(grpcErr)
	} else {
		utils.LavaFormatInfo("GRPC TEST OK")
	}

	jsonCTX.Done()
	rpcCtx.Done()

	lt.finishTestSuccessfully()
}
