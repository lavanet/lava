package relayer

import (
	context "context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/pflag"
)

func PortalServer(
	ctx context.Context,
	clientCtx client.Context,
	listenAddr string,
	chainID string,
	apiInterface string,
	flagSet *pflag.FlagSet,
) {
	//
	utils.LavaFormatInfo("lavad Binary Version: "+version.Version, nil)
	rand.Seed(time.Now().UnixNano())
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
	g_sentry = sentry
	g_serverChainID = chainID

	// Node
	pLogs, err := chainproxy.NewPortalLogs()
	if err != nil {
		log.Fatalln("error: NewPortalLogs", err)
	}
	chainProxy, err := chainproxy.GetChainProxy("", 1, sentry, pLogs)
	if err != nil {
		log.Fatalln("error: GetChainProxy", err)
	}
	// Setting up the sentry callback
	err = sentry.SetupConsumerSessionManager(ctx, chainProxy.GetConsumerSessionManager())
	if err != nil {
		log.Fatalln("error: SetupConsumerSessionManager", err)
	}
	//
	// Set up a connection to the server.
	utils.LavaFormatInfo("PortalServer"+apiInterface, nil)
	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		log.Fatalln("error: getKeyName", err)
	}
	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)

	utils.LavaFormatInfo("Client pubkey: "+fmt.Sprintf("%s", clientKey.GetPubKey().Address()), nil)

	//
	//
	chainProxy.PortalStart(ctx, privKey, listenAddr)
}
