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
	"google.golang.org/grpc"
)

type lavaTest struct {
	testFinishedProperly bool
	grpcConn             *grpc.ClientConn
	lavadPath            string
	logs                 map[string]*bytes.Buffer
	commands             []exec.Cmd
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
			utils.LavaFormatInfo("Waiting for Lava", nil)
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
	stakeCommand := "./scripts/init.sh"
	lt.logs["1_stakeLava"] = new(bytes.Buffer)
	cmd := exec.Cmd{
		Path:   stakeCommand,
		Args:   strings.Split(stakeCommand, " "),
		Stdout: lt.logs["1_stakeLava"],
		Stderr: lt.logs["1_stakeLava"],
	}
	err := cmd.Start()
	if err != nil {
		panic("Staking Failed " + err.Error())
	}
	cmd.Wait()
}

func (lt *lavaTest) checkStakeLava() {
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

	// query all specs
	specQueryRes, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
	if err != nil {
		panic(err)
	}

	pairingQueryClient := pairingTypes.NewQueryClient(lt.grpcConn)
	// check if all specs added exist
	if len(specQueryRes.Spec) == 0 {
		panic("Staking Failed SPEC")
	}
	for _, spec := range specQueryRes.Spec {
		// Query providers

		fmt.Println(spec.GetIndex())
		providerQueryRes, err := pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
			panic(err)
		}
		if len(providerQueryRes.StakeEntry) == 0 {
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
		if len(clientQueryRes.StakeEntry) == 0 {
			panic("Staking Failed CLIENT")
		}
		for _, clientStakeEntry := range clientQueryRes.StakeEntry {
			fmt.Println("client", clientStakeEntry)
		}
	}
}

func (lt *lavaTest) startJSONRPCProxy() {
	goExecutablePath, err := exec.LookPath("go")
	if err != nil {
		panic("Could not find go executable path")
	}
	proxyCommand := goExecutablePath + " test ./testutil/e2e/proxy/. -v eth"
	lt.logs["2_jsonProxy"] = new(bytes.Buffer)
	cmd := exec.Cmd{
		Path:   goExecutablePath,
		Args:   strings.Split(proxyCommand, " "),
		Stdout: lt.logs["2_jsonProxy"],
		Stderr: lt.logs["2_jsonProxy"],
	}
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	lt.commands = append(lt.commands, cmd)
	go func() {
		lt.listenCmdCommand(cmd, "startJSONRPCProxy process returned unexpectedly")
	}()
}

func (lt *lavaTest) startJSONRPCProvider(rpcURL string) {
	providerCommands := []string{
		lt.lavadPath + " server 127.0.0.1 2221 " + rpcURL + " ETH1 jsonrpc --from servicer1 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2222 " + rpcURL + " ETH1 jsonrpc --from servicer2 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2223 " + rpcURL + " ETH1 jsonrpc --from servicer3 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2224 " + rpcURL + " ETH1 jsonrpc --from servicer4 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2225 " + rpcURL + " ETH1 jsonrpc --from servicer5 --log_level debug",
	}
	lt.logs["3_jsonProvider"] = new(bytes.Buffer)
	for idx, providerCommand := range providerCommands {
		cmd := exec.Cmd{
			Path:   lt.lavadPath,
			Args:   strings.Split(providerCommand, " "),
			Stdout: lt.logs["3_jsonProvider"],
			Stderr: lt.logs["3_jsonProvider"],
		}
		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		lt.commands = append(lt.commands, cmd)
		go func(idx int) {
			lt.listenCmdCommand(cmd, "startJSONRPCProvider process returned unexpectedly, provider idx:"+strconv.Itoa(idx))
		}(idx)
	}
}

func (lt *lavaTest) startJSONRPCGateway() {
	utils.LavaFormatInfo("startJSONRPCGateway:", nil)
	providerCommand := lt.lavadPath + " portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1 --log_level debug"
	lt.logs["4_jsonGateway"] = new(bytes.Buffer)
	cmd := exec.Cmd{
		Path:   lt.lavadPath,
		Args:   strings.Split(providerCommand, " "),
		Stdout: lt.logs["4_jsonGateway"],
		Stderr: lt.logs["4_jsonGateway"],
	}
	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	lt.commands = append(lt.commands, cmd)
	go func() {
		lt.listenCmdCommand(cmd, "startJSONRPCGateway process returned unexpectedly")
	}()
}

func (lt *lavaTest) finishTestSuccessfully() {
	lt.testFinishedProperly = true
	for _, cmd := range lt.commands { // kill all the project commands
		cmd.Process.Kill()
	}
}

func (lt *lavaTest) listenCmdCommand(cmd exec.Cmd, panicReason string) {
	err := cmd.Wait()
	if err != nil {
		utils.LavaFormatError("cmd wait err", err, nil)
	}
	if lt.testFinishedProperly {
		return
	}
	lt.saveLogs()
	panic(panicReason)
}

// If after timeout and the check does not return it means it failed
func (lt *lavaTest) checkJSONRPCGateway(rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting JSONRPC Gateway", nil)
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			continue
		}
		_, err = client.BlockNumber(context.Background())
		if err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	panic("checkJSONRPCGateway: JSONRPC Check Failed Gateway didn't respond")
}

func jsonrpcTests(rpcURL string, testDuration time.Duration) error {
	ctx := context.Background()
	utils.LavaFormatInfo("Starting JSONRPC Tests", nil)
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

func (lt *lavaTest) startTendermintProvider(rpcURL string) {
	providerCommands := []string{
		lt.lavadPath + " server 127.0.0.1 2261 " + rpcURL + " LAV1 tendermintrpc --from servicer6 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2262 " + rpcURL + " LAV1 tendermintrpc --from servicer7 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2263 " + rpcURL + " LAV1 tendermintrpc --from servicer8 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2264 " + rpcURL + " LAV1 tendermintrpc --from servicer9 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2265 " + rpcURL + " LAV1 tendermintrpc --from servicer10 --log_level debug",
	}
	lt.logs["7_tendermintProvider"] = new(bytes.Buffer)
	for idx, providerCommand := range providerCommands {
		cmd := exec.Cmd{
			Path:   lt.lavadPath,
			Args:   strings.Split(providerCommand, " "),
			Stdout: lt.logs["7_tendermintProvider"],
			Stderr: lt.logs["7_tendermintProvider"],
		}
		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		lt.commands = append(lt.commands, cmd)
		go func(idx int) {
			lt.listenCmdCommand(cmd, "startTendermintProvider process returned unexpectedly, provider idx:"+strconv.Itoa(idx))
		}(idx)
	}
}

func (lt *lavaTest) startTendermintGateway() {
	providerCommand := lt.lavadPath + " portal_server 127.0.0.1 3340 LAV1 tendermintrpc --from user2 --log_level debug"
	lt.logs["8_tendermintGateway"] = new(bytes.Buffer)
	cmd := exec.Cmd{
		Path:   lt.lavadPath,
		Args:   strings.Split(providerCommand, " "),
		Stdout: lt.logs["8_tendermintGateway"],
		Stderr: lt.logs["8_tendermintGateway"],
	}
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	lt.commands = append(lt.commands, cmd)
	go func() {
		lt.listenCmdCommand(cmd, "startTendermintGateway process returned unexpectedly")
	}()
}

func (lt *lavaTest) checkTendermintGateway(rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting TENDERMINT Gateway", nil)
		client, err := tmclient.New(rpcURL, "/websocket")
		if err != nil {
			continue
		}
		_, err = client.Status(context.Background())
		if err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	panic("TENDERMINT Check Failed")
}

func tendermintTests(rpcURL string, testDuration time.Duration) error {
	ctx := context.Background()
	utils.LavaFormatInfo("Starting TENDERMINT Tests", nil)
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
	utils.LavaFormatInfo("Starting TENDERMINTRPC URI Tests", nil)
	errors := []string{}
	mostImportantApisToTest := map[string]bool{
		"%s/health":                              true,
		"%s/status":                              true,
		"%s/block?height=1":                      true,
		"%s/blockchain?minHeight=0&maxHeight=10": true,
		"%s/dial_peers?persistent=true&unconditional=true&private=true": false, // this is a rpc affecting query and is not available on the spec so it should fail
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

func (lt *lavaTest) startRESTProvider(rpcURL string) {
	providerCommands := []string{
		lt.lavadPath + " server 127.0.0.1 2271 " + rpcURL + " LAV1 rest --from servicer6 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2272 " + rpcURL + " LAV1 rest --from servicer7 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2273 " + rpcURL + " LAV1 rest --from servicer8 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2274 " + rpcURL + " LAV1 rest --from servicer9 --log_level debug",
		lt.lavadPath + " server 127.0.0.1 2275 " + rpcURL + " LAV1 rest --from servicer10 --log_level debug",
	}
	lt.logs["5_restProvider"] = new(bytes.Buffer)
	for idx, providerCommand := range providerCommands {
		cmd := exec.Cmd{
			Path:   lt.lavadPath,
			Args:   strings.Split(providerCommand, " "),
			Stdout: lt.logs["5_restProvider"],
			Stderr: lt.logs["5_restProvider"],
		}
		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		lt.commands = append(lt.commands, cmd)
		go func(idx int) {
			lt.listenCmdCommand(cmd, "startRESTProvider process returned unexpectedly, provider idx:"+strconv.Itoa(idx))
		}(idx)
	}
}

func (lt *lavaTest) startRESTGateway() {
	providerCommand := lt.lavadPath + " portal_server 127.0.0.1 3341 LAV1 rest --from user3 --log_level debug"
	lt.logs["6_restGateway"] = new(bytes.Buffer)
	cmd := exec.Cmd{
		Path:   lt.lavadPath,
		Args:   strings.Split(providerCommand, " "),
		Stdout: lt.logs["6_restGateway"],
		Stderr: lt.logs["6_restGateway"],
	}
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	lt.commands = append(lt.commands, cmd)
	go func() {
		lt.listenCmdCommand(cmd, "startRESTGateway process returned unexpectedly")
	}()
}

func (lt *lavaTest) checkRESTGateway(rpcURL string, timeout time.Duration) {
	for start := time.Now(); time.Since(start) < timeout; {
		utils.LavaFormatInfo("Waiting REST Gateway", nil)
		reply, err := getRequest(fmt.Sprintf("%s/blocks/latest", rpcURL))
		if err != nil || strings.Contains(string(reply), "error") {
			time.Sleep(time.Second)
			continue
		} else {
			return
		}
	}
	panic("REST Check Failed")
}

func restTests(rpcURL string, testDuration time.Duration) error {
	utils.LavaFormatInfo("Starting REST Tests", nil)
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

func (lt *lavaTest) saveLogs() {
	logsFolder := "./testutil/e2e/logs/"
	if _, err := os.Stat(logsFolder); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(logsFolder, os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
	}
	for fileName, logBuffer := range lt.logs {
		file, err := os.Create(logsFolder + fileName + ".log")
		if err != nil {
			panic(err)
		}
		writer := bufio.NewWriter(file)
		writer.Write(logBuffer.Bytes())
		writer.Flush()
		file.Close()
		utils.LavaFormatInfo(logBuffer.String(), nil)
	}
}

func (lt *lavaTest) checkPayments(testDuration time.Duration) {
	utils.LavaFormatInfo("Checking Payments", nil)
	ethPaid := false
	lavaPaid := false
	for start := time.Now(); time.Since(start) < testDuration; {
		pairingClient := pairingTypes.NewQueryClient(lt.grpcConn)
		pairingRes, err := pairingClient.EpochPaymentsAll(context.Background(), &pairingTypes.QueryAllEpochPaymentsRequest{})
		if err != nil {
			panic(err)
		}

		if len(pairingRes.EpochPayments) == 0 {
			utils.LavaFormatInfo("Waiting Payments", nil)
			time.Sleep(time.Second)
			continue
		}
		for _, epochPayment := range pairingRes.EpochPayments {
			for _, clientsPayment := range epochPayment.ClientsPayments {
				if strings.Contains(clientsPayment.Index, "ETH") {
					ethPaid = true
				} else if strings.Contains(clientsPayment.Index, "LAV") {
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
		utils.LavaFormatInfo("PAYMENT SUCCESSFUL FOR ETH", nil)
	} else {
		panic("PAYMENT FAILED FOR ETH")
	}

	if lavaPaid {
		utils.LavaFormatInfo("PAYMENT SUCCESSFUL FOR LAVA", nil)
	} else {
		panic("PAYMENT FAILED FOR LAVA")
	}
}

func runE2E() {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	grpcConn, err := grpc.Dial("127.0.0.1:9090", grpc.WithInsecure())
	if err != nil {
		// Just log because grpc redials
		fmt.Println(err)
	}
	lt := &lavaTest{
		grpcConn:     grpcConn,
		lavadPath:    gopath + "/bin/lavad",
		logs:         make(map[string]*bytes.Buffer),
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

	utils.LavaFormatInfo("Starting Lava", nil)
	go lt.startLava(context.Background())
	lt.checkLava(time.Minute * 10)
	utils.LavaFormatInfo("Starting Lava OK", nil)
	utils.LavaFormatInfo("Staking Lava", nil)
	lt.stakeLava()
	lt.checkStakeLava()
	utils.LavaFormatInfo("Staking Lava OK", nil)
	lt.startJSONRPCProxy()
	lt.startJSONRPCProvider("http://127.0.0.1:1111")
	lt.startJSONRPCGateway()
	lt.checkJSONRPCGateway("http://127.0.0.1:3333/1", time.Second*30)

	lt.startTendermintProvider("http://0.0.0.0:26657")
	lt.startTendermintGateway()
	lt.checkTendermintGateway("http://127.0.0.1:3340/1", time.Second*30)

	lt.startRESTProvider("http://127.0.0.1:1317")
	lt.startRESTGateway()
	lt.checkRESTGateway("http://127.0.0.1:3341/1", time.Second*30)

	utils.LavaFormatInfo("RUNNING TESTS", nil)

	jsonErr := jsonrpcTests("http://127.0.0.1:3333/1", time.Second*30)
	tendermintErr := tendermintTests("http://127.0.0.1:3340/1", time.Second*30)
	tendermintURIErr := tendermintURITests("http://127.0.0.1:3340/1", time.Second*30)
	restErr := restTests("http://127.0.0.1:3341/1", time.Second*30)

	if jsonErr != nil {
		panic(jsonErr)
	} else {
		utils.LavaFormatInfo("JSONRPC TEST OK", nil)
	}

	if tendermintErr != nil {
		panic(tendermintErr)
	} else {
		utils.LavaFormatInfo("TENDERMINTRPC TEST OK", nil)
	}

	if tendermintURIErr != nil {
		panic(tendermintURIErr)
	} else {
		utils.LavaFormatInfo("TENDERMINTRPC URI TEST OK", nil)
	}

	if restErr != nil {
		panic(restErr)
	} else {
		utils.LavaFormatInfo("REST TEST OK", nil)
	}

	lt.checkPayments(time.Minute * 10)
	lt.finishTestSuccessfully()
}
