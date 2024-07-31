package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lavanet/lava/v2/utils"
	pairingTypes "github.com/lavanet/lava/v2/x/pairing/types"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
)

type SafeBuffer struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

func (sb *SafeBuffer) WriteString(s string) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buffer.WriteString(s)
}

func (sb *SafeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buffer.String()
}

func (sb *SafeBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buffer.Write(p)
}

func (sb *SafeBuffer) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buffer.Bytes()
}

// PairingList struct is used to store seed provider information for lavaOverLava
type PairingList struct {
	TestNet Geolocations `json:"testnet"`
}

// Geolocations struct is used to store geolocations
type Geolocations struct {
	One []Pair `json:"1"`
}

// Pair struct is used to store provider RPCAddress and PublicAddress
type Pair struct {
	RPCAddress    string `json:"rpcAddress"`
	PublicAddress string `json:"publicAddress"`
}

func RunSDKTest(testFile string, privateKey string, publicKey string, logs *SafeBuffer, badgePort string) error {
	utils.LavaFormatInfo("[+] Starting SDK test", utils.LogAttr("test_file", testFile))
	// Prepare command for running test
	cmd := exec.Command("ts-node", testFile)

	// Get os environment
	cmd.Env = os.Environ()

	// Set the environment variable for the private key
	cmd.Env = append(cmd.Env, "PRIVATE_KEY="+privateKey)
	cmd.Env = append(cmd.Env, "PUBLIC_KEY="+publicKey)

	// Set the environment variable for badge server project id
	cmd.Env = append(cmd.Env, "BADGE_PROJECT_ID="+"alice")

	// Set the environment variable for badge server address
	cmd.Env = append(cmd.Env, "BADGE_SERVER_ADDR="+"http://localhost:"+badgePort)

	// Set the environment variable for badge server address
	cmd.Env = append(cmd.Env, "PAIRING_LIST="+"testutil/e2e/sdk/pairingList.json")

	// Run the command and capture both standard output and standard error
	cmd.Stdout = logs
	cmd.Stderr = logs

	// Run the test.
	utils.LavaFormatInfo(fmt.Sprintf("Running test: %s", testFile))
	err := cmd.Run()
	if err != nil {
		utils.LavaFormatPanic("Failed running test", err, utils.Attribute{Key: "test file", Value: testFile})
	} else {
		utils.LavaFormatInfo(logs.String())
	}
	return nil
}

func RunSDKTests(ctx context.Context, grpcConn *grpc.ClientConn, privateKey string, publicKey string, logs *SafeBuffer, badgePort string) {
	defer func() {
		// Delete the file directly without checking if it exists
		os.Remove("testutil/e2e/sdk/pairingList.json")
	}()

	// Generate pairing list config
	GeneratePairingList(grpcConn, ctx)

	// Get a list of all tests files in the tests folder
	testFiles := []string{}
	err := filepath.Walk("./testutil/e2e/sdk/tests", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "node_modules" {
			// Skip the "node_modules" directory
			return filepath.SkipDir
		}
		if !info.IsDir() && !strings.Contains(path, "emergency") && strings.HasSuffix(path, ".ts") {
			testFiles = append(testFiles, path)
		}
		return nil
	})
	if err != nil {
		utils.LavaFormatError("Error finding test files:", err)
		return
	}

	testFilesThatFailed := []string{}
	// Loop through each test file and execute it
	for _, testFile := range testFiles {
		err := RunSDKTest(testFile, privateKey, publicKey, logs, badgePort)
		if err != nil {
			testFilesThatFailed = append(testFilesThatFailed, testFile)
		}
	}
	if len(testFilesThatFailed) > 0 {
		panic(fmt.Sprintf("Test Files failed: %s\n", testFilesThatFailed))
	}
}

func CheckTsNode() {
	// Attempt to run the "ts-node" command to check if it exists
	cmd := exec.Command("ts-node", "--version")

	// Run the command and capture standard output and standard error
	output, err := cmd.CombinedOutput()

	if err != nil {
		// An error occurred, indicating that "ts-node" may not exist
		fmt.Println("ts-node does not exist or encountered an error:")
		fmt.Println(err)
	} else {
		// The command ran successfully, indicating that "ts-node" exists
		fmt.Println("ts-node exists. Version information:")
		fmt.Println(string(output))
	}

	// You can check the exit status as well
	exitStatus := cmd.ProcessState.ExitCode()

	if exitStatus == 0 {
		fmt.Println("Found ts-node locally")
	} else {
		fmt.Printf("Exit status: %d (Failure)\n", exitStatus)
		panic("Didn't find ts-node in the environment")
	}
}

// generatePairingList pairing list seed file
func GeneratePairingList(grpcConn *grpc.ClientConn, ctx context.Context) {
	c := pairingTypes.NewQueryClient(grpcConn)

	queryResponse, err := c.Providers(ctx, &pairingTypes.QueryProvidersRequest{ChainID: "LAV1", ShowFrozen: false})
	if err != nil {
		log.Fatalf("Could not query providers: %v", err)
	}

	pairingList := PairingList{
		TestNet: Geolocations{
			One: make([]Pair, len(queryResponse.StakeEntry)),
		},
	}

	// Transform stakeEntries to pairingList
	for i, entry := range queryResponse.StakeEntry {
		var tendermintEndpoint string
		for _, endpoint := range entry.Endpoints {
			if slices.Contains(endpoint.ApiInterfaces, "tendermintrpc") {
				tendermintEndpoint = endpoint.IPPORT
			}
		}
		pairingList.TestNet.One[i] = Pair{
			RPCAddress:    tendermintEndpoint,
			PublicAddress: entry.Address,
		}
	}

	utils.LavaFormatInfo("PairingList Created:", utils.Attribute{Key: "Json File:", Value: pairingList})
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(pairingList, "", "  ")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Write to file
	err = os.WriteFile("testutil/e2e/sdk/pairingList.json", jsonData, os.ModePerm)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}
