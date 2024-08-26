package e2e

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go/build"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/cmd/lavad/cmd"
	commonconsts "github.com/lavanet/lava/v2/testutil/common/consts"
	e2esdk "github.com/lavanet/lava/v2/testutil/e2e/sdk"
	"github.com/lavanet/lava/v2/utils"
	dualstakingTypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	epochStorageTypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	pairingTypes "github.com/lavanet/lava/v2/x/pairing/types"
	subscriptionTypes "github.com/lavanet/lava/v2/x/subscription/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var startLavaLogName = "00_StartLava"

func (lt *lavaTest) startLavaForPayment(ctx context.Context) {
	command := "./scripts/test/start_env_dev_for_payment_e2e.sh"
	logName := startLavaLogName
	funcName := "startLava"

	lt.execCommand(ctx, funcName, logName, command, true)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) stakeLavaForPayment(ctx context.Context) {
	command := "./scripts/test/init_payment_e2e.sh"
	logName := "01_stakeLavaForPayment"
	funcName := "stakeLavaForPayment"

	lt.execCommand(ctx, funcName, logName, command, true)
	utils.LavaFormatInfo(funcName + OKstr)
}

func (lt *lavaTest) startLavaProvidersForPayment(ctx context.Context) {
	for idx := 1; idx <= 2; idx++ {
		command := fmt.Sprintf(
			"%s rpcprovider %s/lavaProvider%d --chain-id=lava --from servicer%d %s",
			lt.protocolPath, providerConfigsFolder, idx+5, idx, lt.lavadArgs,
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
			lt.protocolPath, consumerConfigsFolder, idx+1, u, lt.lavadArgs+lt.consumerArgs,
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

// getRewards gets the current balances of the input addresses
func (lt *lavaTest) getRewards(addresses []string) ([]sdk.Coin, error) {
	dualstakingQueryClient := dualstakingTypes.NewQueryClient(lt.grpcConn)

	var rewards []sdk.Coin
	for _, addr := range addresses {
		sdkAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return nil, fmt.Errorf("could not get balance of address %s. err: %s", addr, err.Error())
		}

		rewardsRequest := dualstakingTypes.QueryDelegatorRewardsRequest{Delegator: addr, Provider: addr, ChainId: ""}
		res, err := dualstakingQueryClient.DelegatorRewards(context.Background(), &rewardsRequest)
		if err != nil {
			return nil, fmt.Errorf("could not get rewards of address %s. err: %s", sdkAddr.String(), err.Error())
		}
		total := sdk.NewCoin(commonconsts.TestTokenDenom, sdk.ZeroInt())
		for _, r := range res.Rewards {
			total = total.AddAmount(r.Amount.AmountOf(commonconsts.TestTokenDenom))
		}
		rewards = append(rewards, total)
	}

	return rewards, nil
}

// checkPayment checks that at least one providers' balance increased (can't be known
// in test time since pairing is pseudo-random)
// with the monthly payment mechanism, we just wait and the providers get the rewards automatically
func (lt *lavaTest) checkPayment(providers []string, startRewards []sdk.Coin) {
	pairingQueryClient := pairingTypes.NewQueryClient(lt.grpcConn)

	// wait for month+blocksToSave pass (debug_month = 2min, debug_epochsToSave = 5) and query for expected payout
	expectedPayoutArr := make([]uint64, len(providers))
	for i := 0; i < 24; i++ {
		for j := range providers {
			payoutRequest := pairingTypes.QueryProviderMonthlyPayoutRequest{Provider: providers[j]}
			res, err := pairingQueryClient.ProviderMonthlyPayout(context.Background(), &payoutRequest)
			if err != nil {
				panic(err)
			}

			// keep the max amount
			if expectedPayoutArr[j] < res.Total {
				expectedPayoutArr[j] = res.Total
			}
		}
		time.Sleep(time.Second * 10)
	}

	// get new balance and checks that at least one provider's balance was increased
	newRewards, err := lt.getRewards(providers)
	if err != nil {
		panic(err)
	}

	for i := range newRewards {
		newAmount := newRewards[i].Amount
		startAmount := startRewards[i].Amount
		payout := newAmount.Sub(startAmount)
		if payout.IsNegative() || !withinRange(payout.Uint64(), expectedPayoutArr[i], 80) {
			panic(utils.LavaFormatError("payment check failed", fmt.Errorf("provider did not get expected payment"),
				utils.Attribute{Key: "provider", Value: providers[i]},
				utils.Attribute{Key: "start_balance", Value: startRewards[i].String()},
				utils.Attribute{Key: "expected_payout", Value: expectedPayoutArr[i]},
				utils.Attribute{Key: "start_balance+expected_payout", Value: startRewards[i].AddAmount(sdk.NewIntFromUint64(expectedPayoutArr[i])).String()},
				utils.Attribute{Key: "actual_balance", Value: newRewards[i]},
			))
		}
	}
}

func (lt *lavaTest) saveLogsForPayment() {
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
		for _, line := range lines {
			if fileName == "00_StartLava" { // TODO remove this and solve the errors
				break
			}
			if strings.Contains(line, " ERR ") || strings.Contains(line, "[Error]" /* sdk errors*/) {
				isAllowedError := false
				for errorSubstring := range allowedErrors {
					if strings.Contains(line, errorSubstring) {
						isAllowedError = true
						break
					}
				}

				for errorSubstring := range allowedErrorsPaymentE2E {
					if strings.Contains(line, errorSubstring) {
						isAllowedError = true
						break
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

func withinRange(value1, value2, percentage uint64) bool {
	maxDifference := value1 * percentage / 100
	return math.Abs(float64(value1)-float64(value2)) <= float64(maxDifference)
}

var (
	lavadPath = "/bin/lavad"
	lavapPath = "/bin/lavap"
)

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
		lavadPath:    gopath + lavadPath,
		protocolPath: gopath + lavapPath,
		lavadArgs:    "--geolocation 1 --log_level debug",
		consumerArgs: " --allow-insecure-provider-dialing",
		logs:         make(map[string]*e2esdk.SafeBuffer),
		commands:     make(map[string]*exec.Cmd),
		providerType: make(map[string][]epochStorageTypes.Endpoint),
		logPath:      protocolLogsFolder,
		tokenDenom:   commonconsts.TestTokenDenom,
	}
	// use defer to save logs in case the tests fail
	defer func() {
		if r := recover(); r != nil {
			lt.saveLogsForPayment()
			for _, cmd := range lt.commands {
				cmd.Process.Kill()
			}
			panic("E2E Failed")
		} else {
			lt.saveLogsForPayment()
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

	// scripts/test/init_payment_e2e.sh will:
	// - produce 2 spec: LAV1, COSMOS-SDK, IBC
	// - produce 1 plan: "DefaultPlan"
	// - produce 2 staked providers (for LAV1)
	// - produce 1 staked client (for LAV1)
	// - produce 1 subscription (for LAV1)

	lt.checkStakeLava(1, 6, 1, 2, checkedPlansE2E, []string{"LAV1"}, []string{"user1"}, "Staking Lava OK")

	// get balance of providers right after stake for payment check later
	providers, err := lt.getProvidersAddresses()
	if err != nil {
		panic(err)
	}
	startBalances, err := lt.getRewards(providers)
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
		lt.checkTendermintConsumer(url, time.Second*30)
	})

	// check the node's Tendermint port is up
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		if err := tendermintTests(url, time.Second*20); err != nil {
			panic(err)
		}
	})
	utils.LavaFormatInfo("TENDERMINTRPC TEST OK")

	// send relays using Tendermint-RPC
	repeat(1, func(n int) {
		url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
		if err := tendermintURITests(url, time.Second*20); err != nil {
			panic(err)
		}
	})

	// sometimes not enough relays are transmitted to influence the project's CU
	// if the total CU is equal to the remaining CU, send more relays
	subQ := subscriptionTypes.NewQueryClient(lt.grpcConn)
	res, err := subQ.List(context.Background(), &subscriptionTypes.QueryListRequest{})
	if err != nil {
		panic(err)
	}
	if res.SubsInfo[0].MonthCuTotal == res.SubsInfo[0].MonthCuLeft {
		repeat(1, func(n int) {
			url := fmt.Sprintf("http://127.0.0.1:334%d", (n-1)*3)
			if err := tendermintURITests(url, time.Second*30); err != nil {
				panic(err)
			}
		})
	}

	utils.LavaFormatInfo("TENDERMINTRPC URI TEST OK")

	utils.LavaFormatInfo("CHECKING PAYMENT")
	lt.checkPayment(providers, startBalances)
	utils.LavaFormatInfo("PAYMENT TEST OK")

	lt.finishTestSuccessfully()
}
