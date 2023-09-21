package e2e

import (
	"bytes"
	"context"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/lavanet/lava/testutil/e2e/sdk"
	"github.com/lavanet/lava/utils"
	epochStorageTypes "github.com/lavanet/lava/x/epochstorage/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const sdkLogsFolder = "./testutil/e2e/sdkLogs/"

// startBadgeServer starts badge server
func (lt *lavaTest) startBadgeServer(ctx context.Context, privateKey, publicKey string) {
	badgeUserData := fmt.Sprintf(`{"1":{"default":{"project_public_key":"%s","private_key":"%s","epochs_max_cu":3333333333}},"2":{"default":{"project_public_key":"%s","private_key":"%s","epochs_max_cu":3333333333}}}`, publicKey, privateKey, publicKey, privateKey)
	err := os.Setenv("BADGE_USER_DATA", badgeUserData)
	if err != nil {
		panic(err)
	}

	command := fmt.Sprintf("%s badgegenerator --port=7070 --grpc-url=127.0.0.1:9090 --log_level=debug --chain-id lava", lt.protocolPath)
	err = os.Setenv("BADGE_DEFAULT_GEOLOCATION", "1")
	if err != nil {
		panic(err)
	}
	logName := "01_BadgeServer"
	funcName := "startBadgeServer"
	lt.execCommandWithRetry(ctx, funcName, logName, command)

	lt.checkBadgeServerResponsive(ctx, "127.0.0.1:7070", time.Minute)
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
		logs:         make(map[string]*bytes.Buffer),
		commands:     make(map[string]*exec.Cmd),
		providerType: make(map[string][]epochStorageTypes.Endpoint),
		logPath:      sdkLogsFolder,
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

	lt.checkStakeLava(1, 5, 3, 5, checkedPlansE2E, checkedSpecsE2E, checkedSubscriptions, "Staking Lava OK")

	utils.LavaFormatInfo("RUNNING TESTS")

	// Export user1 private key
	privateKey := exportUserPrivateKey(lt.lavadPath, "user1")

	// Export user1 public key
	publicKey := exportUserPublicKey(lt.lavadPath, "user1")

	// Start Badge server
	lt.startBadgeServer(ctx, privateKey, publicKey)

	// ETH1 flow
	lt.startJSONRPCProxy(ctx)
	// Check proxy is up
	lt.checkJSONRPCConsumer("http://127.0.0.1:1111", time.Minute*2, "JSONRPCProxy OK") // checks proxy.
	// Start Eth providers
	lt.startJSONRPCProvider(ctx)

	// Lava Flow
	lt.startLavaProviders(ctx)

	// Test SDK
	lt.logs["01_sdkTest"] = new(bytes.Buffer)
	sdk.RunSDKTests(ctx, grpcConn, privateKey, publicKey, lt.logs["01_sdkTest"])

	lt.finishTestSuccessfully()

	// Cancel lava network using context
	cancelLava()

	// Wait for all processes to be done
	lt.wg.Wait()
}
