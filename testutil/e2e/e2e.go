package main

import (
	"bytes"
	"context"
	log "fmt"
	"io"
	"math/big"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ignite/cli/ignite/chainconfig"
	"github.com/ignite/cli/ignite/pkg/cache"
	"github.com/ignite/cli/ignite/services/chain"
	"github.com/lavanet/lava/utils"
	pairingTypes "github.com/lavanet/lava/x/pairing/types"
	specTypes "github.com/lavanet/lava/x/spec/types"
	tmclient "github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
)

type lavaTest struct {
	wg       sync.WaitGroup
	grpcConn *grpc.ClientConn
	ctxs     map[string]context.Context
	execOut  bytes.Buffer
	execErr  bytes.Buffer
}

func (lt *lavaTest) startLava(ctx context.Context) {
	defer lt.wg.Done()
	absPath, err := filepath.Abs(".")
	if err != nil {
		log.Println(err)
	}

	c, err := chain.New(absPath, chain.LogLevel(chain.LogRegular))
	if err != nil {
		log.Println(err)
	}
	cacheRootDir, err := chainconfig.ConfigDirPath()
	if err != nil {
		log.Println(err)
	}

	storage, err := cache.NewStorage(filepath.Join(cacheRootDir, "ignite_cache.db"))
	if err != nil {
		log.Println(err)
	}

	err = c.Serve(ctx, storage, chain.ServeForceReset())
	if err != nil {
		log.Println(err)
	}
}

func (lt *lavaTest) checkLava() {
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

	for {
		// TODO
		// This loop would wait for the lavad server to be up before chain init
		// This one should always end but add a timer
		_, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
		if err != nil && strings.Contains(err.Error(), "rpc error") {
			utils.LavaFormatInfo("Waiting for Lava", nil)
			time.Sleep(time.Second)
		} else if err == nil {
			break
		} else {
			log.Println(err)
			return
		}
	}
}

func (lt *lavaTest) stakeLava() {
	cmd := exec.Cmd{
		Path:   "./scripts/init.sh",
		Args:   []string{"./scripts/init.sh"},
		Stdout: &lt.execOut,
		Stderr: &lt.execErr,
	}
	cmd.Start()
	cmd.Wait()
}

func (lt *lavaTest) checkStakeLava() {
	specQueryClient := specTypes.NewQueryClient(lt.grpcConn)

	// query all specs
	specQueryRes, err := specQueryClient.SpecAll(context.Background(), &specTypes.QueryAllSpecRequest{})
	if err != nil {
		return
	}

	pairingQueryClient := pairingTypes.NewQueryClient(lt.grpcConn)
	// check if all specs added exist
	if len(specQueryRes.Spec) == 0 {
		log.Println("Staking Failed SPEC")
		return
	}
	for _, spec := range specQueryRes.Spec {
		// Query providers

		log.Println(spec.GetIndex())
		providerQueryRes, err := pairingQueryClient.Providers(context.Background(), &pairingTypes.QueryProvidersRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
			log.Println(err)
			return
		}
		if len(providerQueryRes.StakeEntry) == 0 {
			log.Println("Staking Failed PROVIDER")
			return
		}
		for _, providerStakeEntry := range providerQueryRes.StakeEntry {
			// check if number of stakes matches number of providers to be launched
			log.Println("provider", providerStakeEntry)
		}

		// Query clients
		clientQueryRes, err := pairingQueryClient.Clients(context.Background(), &pairingTypes.QueryClientsRequest{
			ChainID: spec.GetIndex(),
		})
		if err != nil {
			log.Println(err)
			return
		}
		if len(clientQueryRes.StakeEntry) == 0 {
			log.Println("Staking Failed CLIENT")
			return
		}
		for _, clientStakeEntry := range clientQueryRes.StakeEntry {
			// check if number of stakes matches number of clients to be launched
			log.Println("client", clientStakeEntry)
		}
	}
}

func (lt *lavaTest) startJSONRPCProvider(rpcURL string) {
	// TODO
	// remove ugly path
	// pipe output to array
	providerCommands := []string{
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2221 " + rpcURL + " ETH1 jsonrpc --from servicer1",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2222 " + rpcURL + " ETH1 jsonrpc --from servicer2",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2223 " + rpcURL + " ETH1 jsonrpc --from servicer3",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2224 " + rpcURL + " ETH1 jsonrpc --from servicer4",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2225 " + rpcURL + " ETH1 jsonrpc --from servicer5",
	}
	for _, providerCommand := range providerCommands {
		cmd := exec.Cmd{
			Path:   "/Users/jaketagnepis/go/bin/lavad",
			Args:   strings.Split(providerCommand, " "),
			Stdout: &lt.execOut,
			Stderr: &lt.execErr,
		}
		err := cmd.Start()
		if err != nil {
			log.Println(err)
		}
	}
}

func (lt *lavaTest) startJSONRPCProxy() {
	proxyCommand := "/opt/homebrew/bin/go run ./testutil/e2e/proxy/. eth"
	cmd := exec.Cmd{
		Path: "/opt/homebrew/bin/go",
		Args: strings.Split(proxyCommand, " "),
		// Stdout: os.Stdout,
		// Stderr: os.Stdout,
		Stdout: &lt.execOut,
		Stderr: &lt.execErr,
	}
	err := cmd.Start()
	if err != nil {
		log.Println(err)

	}
}

func (lt *lavaTest) startJSONRPCGateway() {
	// TODO
	// remove ugly path
	// pipe output to array
	providerCommand :=
		"/Users/jaketagnepis/go/bin/lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1"
	cmd := exec.Cmd{
		Path: "/Users/jaketagnepis/go/bin/lavad",
		Args: strings.Split(providerCommand, " "),
		// Stdout: os.Stdout,
		// Stderr: os.Stdout,
		Stdout: &lt.execOut,
		Stderr: &lt.execErr,
	}
	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}
}

func (lt *lavaTest) checkJSONRPCGateway(rpcURL string) {
	for {
		utils.LavaFormatInfo("Waiting JSONRPC Gateway", nil)
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			continue
		}
		_, err = client.BlockNumber(context.Background())
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
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
		return log.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}

func (lt *lavaTest) startRESTProvider(rpcURL string) {
	// TODO
	// remove ugly path
	// pipe output to array
	providerCommands := []string{
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2271 " + rpcURL + " LAV1 rest --from servicer1",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2272 " + rpcURL + " LAV1 rest --from servicer2",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2273 " + rpcURL + " LAV1 rest --from servicer3",
	}
	for _, providerCommand := range providerCommands {
		cmd := exec.Cmd{
			Path:   "/Users/jaketagnepis/go/bin/lavad",
			Args:   strings.Split(providerCommand, " "),
			Stdout: &lt.execOut,
			Stderr: &lt.execErr,
		}
		err := cmd.Start()
		if err != nil {
			log.Println(err)
		}
	}
}

func (lt *lavaTest) startRESTGateway() {
	// TODO
	// remove ugly path
	// pipe output to array
	providerCommand := "/Users/jaketagnepis/go/bin/lavad portal_server 127.0.0.1 3340 LAV1 rest --from user4"
	cmd := exec.Cmd{
		Path:   "/Users/jaketagnepis/go/bin/lavad",
		Args:   strings.Split(providerCommand, " "),
		Stdout: &lt.execOut,
		Stderr: &lt.execErr,
	}
	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}
}

func (lt *lavaTest) checkRESTGateway(rpcURL string) {
	for {
		utils.LavaFormatInfo("Waiting REST Gateway", nil)
		reply, err := getRequest(log.Sprintf("%s/blocks/latest", rpcURL))
		if err != nil || strings.Contains(string(reply), "error") {
			time.Sleep(time.Second)
			continue
		} else {
			break
		}
	}
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
			reply, err := getRequest(log.Sprintf(api, rpcURL))
			if err != nil {
				errors = append(errors, log.Sprintf("%s", err))
			} else if strings.Contains(string(reply), "error") {
				errors = append(errors, string(reply))
			}
		}
	}

	if len(errors) > 0 {
		return log.Errorf(strings.Join(errors, ",\n"))
	}
	return nil
}

func (lt *lavaTest) startTendermintProvider(rpcURL string) {
	// TODO
	// remove ugly path
	// pipe output to array
	providerCommands := []string{
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2261 " + rpcURL + " LAV1 tendermintrpc --from servicer1",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2262 " + rpcURL + " LAV1 tendermintrpc --from servicer2",
		"/Users/jaketagnepis/go/bin/lavad server 127.0.0.1 2263 " + rpcURL + " LAV1 tendermintrpc --from servicer3",
	}
	for _, providerCommand := range providerCommands {
		cmd := exec.Cmd{
			Path:   "/Users/jaketagnepis/go/bin/lavad",
			Args:   strings.Split(providerCommand, " "),
			Stdout: &lt.execOut,
			Stderr: &lt.execErr,
		}
		err := cmd.Start()
		if err != nil {
			log.Println(err)
		}
	}
}

func (lt *lavaTest) startTendermintGateway() {
	// TODO
	// remove ugly path
	// pipe output to array
	providerCommand := "/Users/jaketagnepis/go/bin/lavad portal_server 127.0.0.1 3341 LAV1 tendermintrpc --from user4"
	cmd := exec.Cmd{
		Path:   "/Users/jaketagnepis/go/bin/lavad",
		Args:   strings.Split(providerCommand, " "),
		Stdout: &lt.execOut,
		Stderr: &lt.execErr,
	}
	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}
}

func (lt *lavaTest) checkTendermintGateway(rpcURL string) {
	for {
		utils.LavaFormatInfo("Waiting TENDERMINT Gateway", nil)
		client, err := tmclient.New(rpcURL, "/websocket")
		if err != nil {
			continue
		}
		_, err = client.Status(context.Background())
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
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
		return log.Errorf(strings.Join(errors, ",\n"))
	}
	return nil
}

func getRequest(url string) ([]byte, error) {
	res, err := http.Get(url) //nolint:gosec
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

func main() {
	grpcConn, err := grpc.Dial("127.0.0.1:9090", grpc.WithInsecure())
	if err != nil {
		// this check does not get triggered even if server is down
		log.Println(err)
		return
	}
	lt := &lavaTest{
		grpcConn: grpcConn,
		ctxs:     make(map[string]context.Context),
	}
	lt.ctxs["lavaMain"] = context.Background()
	lt.wg.Add(1)
	utils.LavaFormatInfo("Starting Lava", nil)
	go lt.startLava(lt.ctxs["lavaMain"])
	lt.checkLava()
	utils.LavaFormatInfo("Starting Lava OK", nil)
	utils.LavaFormatInfo("Staking Lava", nil)
	lt.stakeLava()
	lt.checkStakeLava()
	utils.LavaFormatInfo("Staking Lava OK", nil)

	lt.startJSONRPCProxy()
	lt.startJSONRPCProvider("http://127.0.0.1:1111")
	lt.startJSONRPCGateway()
	lt.checkJSONRPCGateway("http://127.0.0.1:3333/1")

	lt.startRESTProvider("http://127.0.0.1:1317")
	lt.startRESTGateway()
	lt.checkRESTGateway("http://127.0.0.1:3340/1")

	lt.startTendermintProvider("http://0.0.0.0:26657")
	lt.startTendermintGateway()
	lt.checkTendermintGateway("http://127.0.0.1:3341/1")
	utils.LavaFormatInfo("RUNNING TESTS", nil)

	err = jsonrpcTests("http://127.0.0.1:3333/1", time.Second*30)
	log.Println(err)
	err = restTests("http://127.0.0.1:3340/1", time.Second*30)
	log.Println(err)
	err = tendermintTests("http://127.0.0.1:3341/1", time.Second*30)
	log.Println(err)
	lt.wg.Wait()
}
