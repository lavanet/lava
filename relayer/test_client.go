package relayer

import (
	context "context"
	"log"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/relayer/testclients"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/pflag"
)

func TestClient(
	ctx context.Context,
	clientCtx client.Context,
	chainID string,
	apiInterface string,
	duration int64,
	flagSet *pflag.FlagSet,
) {
	// Every client must preseed
	rand.Seed(time.Now().UnixNano())

	//
	sk, _, err := utils.GetOrCreateVRFKey(clientCtx)
	if err != nil {
		log.Fatalln("error: GetOrCreateVRFKey", err)
	}
	// Start sentry
	sentry := sentry.NewSentry(clientCtx, chainID, true, nil, nil, apiInterface, sk, flagSet, 0)
	err = sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go sentry.Start(ctx)
	for sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}

	//
	// Node
	chainProxy, err := chainproxy.GetChainProxy("", 1, sentry)
	if err != nil {
		log.Fatalln("error: GetChainProxy", err)
	}

	//
	// Set up a connection to the server.
	log.Println("TestClient connecting")

	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		log.Fatalln("error: getKeyName", err)
	}

	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)
	log.Println("Client pubkey", clientKey.GetPubKey().Address())

	testDuration := time.Second * time.Duration(duration)
	// Run tests
	var testErrors error = nil
	switch chainID {
	case "ETH1":
		testErrors = testclients.EthTests(ctx, chainID, "http://127.0.0.1:3333/1", testDuration)
	case "GTH1":
		testErrors = testclients.EthTests(ctx, chainID, "http://127.0.0.1:3339/1", testDuration)
	case "FTM250":
		testErrors = testclients.EthTests(ctx, chainID, "http://127.0.0.1:3336/1", testDuration)
	case "COS1":
		testErrors = testclients.TerraTests(ctx, chainProxy, privKey, apiInterface)
	case "COS3", "COS4":
		testErrors = testclients.OsmosisTests(ctx, chainProxy, privKey, apiInterface)
	case "LAV1":
		testErrors = testclients.LavaTests(ctx, chainProxy, privKey, apiInterface, sentry, clientCtx)
	case "APT1":
		testErrors = testclients.AptosTests(ctx, chainProxy, privKey, apiInterface, sentry, clientCtx)
	case "JUN1":
		testErrors = testclients.JunoTests(ctx, chainProxy, privKey, apiInterface)
	case "COS5":
		testErrors = testclients.CosmoshubTests(ctx, chainProxy, privKey, apiInterface, sentry, clientCtx)
	}

	if testErrors != nil {
		log.Fatalf("%s Client test failed with errors %s\n", chainID, testErrors)
	} else {
		log.Printf("%s Client test  complete \n", chainID)
	}
}
