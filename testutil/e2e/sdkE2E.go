package e2e

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	commonconsts "github.com/lavanet/lava/v2/testutil/common/consts"
	"github.com/lavanet/lava/v2/testutil/e2e/sdk"
	"github.com/lavanet/lava/v2/utils"
	epochStorageTypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const sdkLogsFolder = "./testutil/e2e/sdkLogs/"

// startBadgeServer starts badge server
func (lt *lavaTest) startBadgeServer(ctx context.Context, walletName, port, configPath string) {
	command := fmt.Sprintf(
		"%s badgeserver %s --port %s --chain-id=lava --from %s --log_level debug",
		lt.protocolPath, configPath, port, walletName,
	)

	logName := "01_BadgeServer_" + port
	funcName := "startBadgeServer_" + port
	lt.execCommand(ctx, funcName, logName, command, false)

	lt.checkBadgeServerResponsive(ctx, fmt.Sprintf("127.0.0.1:%s", port), time.Minute)

	utils.LavaFormatInfo(funcName + " OK")
}

// exportUserPublicKey exports public key from specific user
func exportUserPublicKey(lavaPath, user string) string {
	cmdString := fmt.Sprintf("%s keys show %s ", lavaPath, user)
	cmd := exec.Command("bash", "-c", cmdString)

	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	// Regex to match the 'public key'
	re := regexp.MustCompile(`address: (\S+)`)
	match := re.FindStringSubmatch(string(out))

	if len(match) < 2 {
		panic("No public key found")
	}

	// Return the 'public key'
	return match[1]
}

// exportUserPrivateKey exports raw private keys from specific user
func exportUserPrivateKey(lavaPath, user string) string {
	cmdString := fmt.Sprintf("yes | %s keys export %s --unsafe --unarmored-hex", lavaPath, user)
	cmd := exec.Command("bash", "-c", cmdString)

	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(out))
}

func runSDKE2E(timeout time.Duration) {
	sdk.CheckTsNode()
	os.RemoveAll(sdkLogsFolder)
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
		logPath:      sdkLogsFolder,
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
	lavaContext, cancelLava := context.WithCancel(context.Background())
	go lt.startLava(lavaContext)
	lt.checkLava(timeout)
	utils.LavaFormatInfo("Starting Lava OK")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	utils.LavaFormatInfo("Staking Lava")
	lt.stakeLava(ctx)

	lt.checkStakeLava(2, NumberOfSpecsExpectedInE2E, 4, 5, checkedPlansE2E, checkedSpecsE2E, checkedSubscriptions, "Staking Lava OK")

	utils.LavaFormatInfo("RUNNING TESTS")

	userWallet := "user1"

	// Start Badge server
	lt.startBadgeServer(ctx, userWallet, "7070", badgeserverConfigFolder+"1")

	// ETH1 flow
	lt.startJSONRPCProxy(ctx)
	// Check proxy is up
	lt.checkJSONRPCConsumer("http://127.0.0.1:1111", time.Minute*2, "JSONRPCProxy OK") // checks proxy.
	// Start Eth providers
	lt.startJSONRPCProvider(ctx)

	// Lava Flow
	lt.startLavaProviders(ctx)

	// Export user private key
	privateKey := exportUserPrivateKey(lt.lavadPath, userWallet)

	// Export user public key
	publicKey := exportUserPublicKey(lt.lavadPath, userWallet)

	// Test SDK
	lt.logs["01_sdkTest"] = &sdk.SafeBuffer{}
	sdk.RunSDKTests(ctx, grpcConn, privateKey, publicKey, lt.logs["01_sdkTest"], "7070")

	// Emergency mode tests
	utils.LavaFormatInfo("Sleeping Until All Rewards are collected")
	lt.sleepUntilNextEpoch()
	lt.sleepUntilNextEpoch()
	lt.sleepUntilNextEpoch()
	lt.sleepUntilNextEpoch()

	utils.LavaFormatInfo("Restarting lava to emergency mode")
	lt.stopLava()
	go lt.startLavaInEmergencyMode(lavaContext, 100000)

	lt.checkLava(timeout)
	utils.LavaFormatInfo("Starting Lava OK")

	var epochDuration int64 = 20 * 1.2
	signalChannel := make(chan bool)
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

	utils.LavaFormatInfo("Waiting for finishing current epoch 1")

	// we should have approximately (numOfProviders * epoch_cu_limit * 2) CU
	// skip current epoch
	<-signalChannel

	userWallet = "user5"
	// Export user private key
	privateKey = exportUserPrivateKey(lt.lavadPath, userWallet)

	// Export user public key
	publicKey = exportUserPublicKey(lt.lavadPath, userWallet)

	lt.startBadgeServer(ctx, userWallet, "5050", badgeserverConfigFolder+"2")

	defer func() {
		// Delete the file directly without checking if it exists
		os.Remove("testutil/e2e/sdk/pairingList.json")
	}()
	sdk.GeneratePairingList(grpcConn, ctx)

	// Test without badge server
	utils.LavaFormatInfo("Waiting for finishing current epoch 2")
	err = sdk.RunSDKTest("testutil/e2e/sdk/tests/emergency_mode_fetch.ts", privateKey, publicKey, lt.logs["01_sdkTest"], "5050")
	if err != nil {
		panic(fmt.Sprintf("Test File failed: %s\n", "testutil/e2e/sdk/tests/emergency_mode_fetch.ts"))
	}

	// Trying to exceed CU limit
	err = sdk.RunSDKTest("testutil/e2e/sdk/tests/emergency_mode_fetch_err.ts", privateKey, publicKey, lt.logs["01_sdkTest"], "5050")
	if err != nil {
		panic(fmt.Sprintf("Test File failed while trying to exceed CU limit: %s\n", "testutil/e2e/sdk/tests/emergency_mode_fetch_err.ts"))
	}

	utils.LavaFormatInfo("KEYS EMERGENCY MODE TEST OK")

	utils.LavaFormatInfo("Waiting for finishing current epoch 3")

	// we should have approximately (numOfProviders * epoch_cu_limit * 3) CU
	// skip current epoch
	<-signalChannel
	<-signalChannel
	<-signalChannel

	// Test with badge server
	err = sdk.RunSDKTest("testutil/e2e/sdk/tests/emergency_mode_badge.ts", privateKey, publicKey, lt.logs["01_sdkTest"], "5050")
	if err != nil {
		panic(fmt.Sprintf("Test File failed: %s\n", "testutil/e2e/sdk/tests/emergency_mode_badge.ts"))
	}

	// Trying to exceed CU limit
	err = sdk.RunSDKTest("testutil/e2e/sdk/tests/emergency_mode_badge_err.ts", privateKey, publicKey, lt.logs["01_sdkTest"], "5050")
	if err != nil {
		panic(fmt.Sprintf("Test File failed while trying to exceed CU limit: %s\n", "testutil/e2e/sdk/tests/emergency_mode_badge_err.ts"))
	}

	utils.LavaFormatInfo("BADGE EMERGENCY MODE TEST OK")

	lt.finishTestSuccessfully()

	// Cancel lava network using context
	cancelLava()

	// Wait for all processes to be done
	lt.wg.Wait()
}
