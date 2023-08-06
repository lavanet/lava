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

	"github.com/lavanet/lava/utils"
	pairingTypes "github.com/lavanet/lava/x/pairing/types"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
)

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

func RunSDKTests(ctx context.Context, grpcConn *grpc.ClientConn, privateKey string, logs *bytes.Buffer) {
	defer func() {
		// Delete the file directly without checking if it exists
		os.Remove("testutil/e2e/sdk/pairingList.json")
	}()

	// Generate pairing list config
	generatePairingList(grpcConn, ctx)

	// Get a list of all tests files in the tests folder
	testFiles := []string{}
	err := filepath.Walk("./testutil/e2e/sdk/tests", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".js") {
			testFiles = append(testFiles, path)
		}
		return nil
	})
	if err != nil {
		utils.LavaFormatError("Error finding test files:", err)
		return
	}

	// Loop through each test file and execute it
	for _, testFile := range testFiles {
		// Prepare command for running test
		cmd := exec.Command("node", testFile)

		// Get os environment
		cmd.Env = os.Environ()

		// Set the environment variable for the private key
		cmd.Env = append(cmd.Env, "PRIVATE_KEY="+privateKey)

		// Set the environment variable for badge server project id
		cmd.Env = append(cmd.Env, "BADGE_PROJECT_ID="+"alice")

		// Set the environment variable for badge server address
		cmd.Env = append(cmd.Env, "BADGE_SERVER_ADDR="+"http://localhost:8080")

		// Set the environment variable for badge server address
		cmd.Env = append(cmd.Env, "PAIRING_LIST="+"testutil/e2e/sdk/pairingList.json")

		// Run the command and capture both standard output and standard error
		cmd.Stdout = logs
		cmd.Stderr = logs

		// Run the test.
		utils.LavaFormatInfo(fmt.Sprintf("Running test: %s", testFile))
		err := cmd.Run()
		if err != nil {
			panic(fmt.Sprintf("Error running test %s: %v\n", testFile, err))
		}
	}
}

// generatePairingList pairing list seed file
func generatePairingList(grpcConn *grpc.ClientConn, ctx context.Context) {
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
