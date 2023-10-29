package e2e

import (
	"bytes"
	"context"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/lavanet/lava/cmd/lavad/cmd"
	"github.com/lavanet/lava/utils"
	epochStorageTypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingTypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var startLavaLogName = "00_StartLava"

func (lt *lavaTest) startLavaForPayment(ctx context.Context) {
	command := "./scripts/start_env_dev_for_payment_e2e.sh"
	logName := startLavaLogName
	funcName := "startLava"

	lt.execCommand(ctx, funcName, logName, command, true)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) stakeLavaForPayment(ctx context.Context) {
	command := "./scripts/init_payment_e2e.sh"
	logName := "01_stakeLavaForPayment"
	funcName := "stakeLavaForPayment"

	lt.execCommand(ctx, funcName, logName, command, true)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) startLavaProvidersForPayment(ctx context.Context) {
	for idx := 1; idx <= 2; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/lavaProvider%d --chain-id=lava --from servicer%d %s",
			lt.protocolPath, configFolder, idx+5, idx, lt.lavadArgs,
		)
		logName := "05_LavaProvider_" + fmt.Sprintf("%02d", idx)
		funcName := fmt.Sprintf("startLavaProvidersForPayment (provider %02d)", idx)
		lt.execCommand(ctx, funcName, logName, command, false)
	}

	// validate all providers are up
	for idx := 1; idx <= 2; idx++ {
		lt.checkProviderResponsive(ctx, fmt.Sprintf("127.0.0.1:226%d", idx), time.Minute)
	}

	utils.LavaFormatInfo("startLavaProvidersForPayment OK")
}

func (lt *lavaTest) startLavaConsumerForPayment(ctx context.Context) {
	for idx, u := range []string{"user1"} {
		command := fmt.Sprintf(
			"%s rpcconsumer %s/lavaConsumer%d.yml --chain-id=lava --from %s %s --concurrent-providers 1",
			lt.protocolPath, configFolder, idx+1, u, lt.lavadArgs+lt.consumerArgs,
		)
		logName := "06_RPCConsumer_" + fmt.Sprintf("%02d", idx+1)
		funcName := fmt.Sprintf("startLavaConsumerForPayment (consumer %02d)", idx+1)
		lt.execCommand(ctx, funcName, logName, command, false)
	}
	utils.LavaFormatInfo("startLavaConsumerForPayment OK")
}

// getProvidersAddresses gets the addresses of the staked providers (on LAV1)
func (lt *lavaTest) getProvidersAddresses() ([]string, error) {
	chainID := "LAV1"
	var addresses []string

	pairingQueryClient := pairingTypes.NewQueryClient(lt.grpcConn)
	providersRequest := pairingTypes.QueryProvidersRequest{ChainID: chainID}
	res, err := pairingQueryClient.Providers(context.Background(), &providersRequest)
	if err != nil {
		return nil, fmt.Errorf("could not get provider address. providers query failed: %s", err.Error())
	}

	providers := res.StakeEntry
	if len(providers) < 1 {
		return nil, fmt.Errorf("could not get provider address. no providers staked on %s", chainID)
	}

	for _, p := range providers {
		addresses = append(addresses, p.Address)
	}

	return addresses, nil
}

// getBalances gets the current balances of the input addresses
func (lt *lavaTest) getBalances(addresses []string) ([]sdk.Coin, error) {
	bankQueryClient := bankTypes.NewQueryClient(lt.grpcConn)

	var balances []sdk.Coin
	for _, addr := range addresses {
		sdkAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return nil, fmt.Errorf("could not get balance of address %s. err: %s", addr, err.Error())
		}

		balanceRequest := bankTypes.NewQueryBalanceRequest(sdkAddr, epochStorageTypes.TokenDenom)
		res, err := bankQueryClient.Balance(context.Background(), balanceRequest)
		if err != nil {
			return nil, fmt.Errorf("could not get balance of address %s. err: %s", sdkAddr.String(), err.Error())
		}

		balances = append(balances, *res.Balance)
	}

	return balances, nil
}

// checkPayment checks that at least one providers' balance increased (can't be known
// in test time since pairing is pseudo-random)
// with the monthly payment mechanism, we just wait and the providers get the rewards automatically
func (lt *lavaTest) checkPayment(providers []string, startBalances []sdk.Coin) {
	// wait for month+blocksToSave pass (debug_month = 2min, debug_epochsToSave = 5)
	time.Sleep(time.Minute * 2)

	// get new balance and checks that at least one provider's balance was increased
	newBalances, err := lt.getBalances(providers)
	if err != nil {
		panic(err)
	}

	balanceIncreased := false
	for i := range newBalances {
		if !newBalances[i].IsLTE(startBalances[i]) {
			balanceIncreased = true
		}
	}

	if !balanceIncreased {
		panic("all providers did not get rewards")
	}
}

func runPaymentE2E(timeout time.Duration) {
	cmd.InitSDKConfig()
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
		logs:         make(map[string]*bytes.Buffer),
		commands:     make(map[string]*exec.Cmd),
		providerType: make(map[string][]epochStorageTypes.Endpoint),
		logPath:      protocolLogsFolder,
	}
	// use defer to save logs in case the tests fail
	defer func() {
		if r := recover(); r != nil {
			lt.saveLogs()
			for _, cmd := range lt.commands {
				cmd.Process.Kill()
			}
			panic("E2E Failed")
		} else {
			lt.saveLogs()
		}
	}()

	utils.LavaFormatInfo("Starting Lava")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go lt.startLavaForPayment(ctx)
	lt.checkLava(timeout)
	utils.LavaFormatInfo("Starting Lava OK")

	utils.LavaFormatInfo("Staking Lava")
	lt.stakeLavaForPayment(ctx)

	// scripts/init_payment_e2e.sh will:
	// - produce 2 spec: LAV1, COSMOS-SDK, IBC
	// - produce 1 plan: "DefaultPlan"
	// - produce 2 staked providers (for LAV1)
	// - produce 1 staked client (for LAV1)
	// - produce 1 subscription (for LAV1)

	lt.checkStakeLava(1, 3, 1, 2, checkedPlansE2E, []string{"LAV1"}, []string{"user1"}, "Staking Lava OK")

	// get balance of providers right after stake for payment check later
	providers, err := lt.getProvidersAddresses()
	if err != nil {
		panic(err)
	}
	startBalances, err := lt.getBalances(providers)
	if err != nil {
		panic(err)
	}

	utils.LavaFormatInfo("RUNNING TESTS")

	// repeat() is a helper to run a given function once per client, passing the
	// iteration (client) number to the function
	repeat := func(n int, f func(int)) {
		for i := 1; i <= n; i++ {
			f(i)
		}
	}

	// start the providers and consumer processes
	lt.startLavaProvidersForPayment(ctx)
	lt.startLavaConsumerForPayment(ctx)

	// check the client's Tendermint port is up
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		lt.checkTendermintConsumer(url, time.Second*5)
	})

	// check the node's Tendermint port is up
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		if err := tendermintTests(url, time.Second*10); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("TENDERMINTRPC TEST OK")

	// send relays using Tendermint-RPC
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		if err := tendermintURITests(url, time.Second*10); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("TENDERMINTRPC URI TEST OK")

	utils.LavaFormatInfo("CHECKING PAYMENT")
	lt.checkPayment(providers, startBalances)
	utils.LavaFormatInfo("PAYMENT TEST OK")

	lt.finishTestSuccessfully()
}
