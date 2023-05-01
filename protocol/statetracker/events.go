package statetracker

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/abci/types"
)

const (
	FlagTimeout   = "timeout"
	FlagValue     = "value"
	FlagEventName = "--event"
)

func eventsLookup(ctx context.Context, clientCtx client.Context, blocks int64, eventName string, value string) error {
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	resultStatus, err := clientCtx.Client.Status(ctx)
	if err != nil {
		return err
	}
	latestHeight := resultStatus.SyncInfo.LatestBlockHeight
	if latestHeight < blocks {
		return utils.LavaFormatError("requested blocks is bigger than latest block height", nil, utils.Attribute{Key: "requested", Value: blocks}, utils.Attribute{Key: "latestHeight", Value: latestHeight})
	}

	printEvent := func(event types.Event) string {
		st := event.Type + ": "
		for _, attr := range event.Attributes {
			st += fmt.Sprintf("- %s = %s; ", string(attr.Key), string(attr.Value))
		}
		return st
	}

	readEventsFromBlock := func(block int64, hash string) {
		blockResults, err := clientCtx.Client.BlockResults(ctx, &block)
		if err != nil {
			utils.LavaFormatError("invalid blockResults status", err)
			return
		}
		transactionResults := blockResults.TxsResults
		for _, tx := range transactionResults {
			events := tx.Events
			for _, event := range events {
				if eventName == "" || event.Type == eventName {
					for _, attribute := range event.Attributes {
						if value == "" || string(attribute.Value) == value {
							utils.LavaFormatInfo("Found a matching event", utils.Attribute{Key: "event", Value: printEvent(event)}, utils.Attribute{Key: "height", Value: block})
						}
					}
				}
			}
		}
	}

	if blocks > 0 {
		utils.LavaFormatInfo("Reading Events", utils.Attribute{Key: "from", Value: latestHeight}, utils.Attribute{Key: "to", Value: latestHeight - blocks})
		for block := latestHeight; block >= latestHeight-blocks; block-- {
			readEventsFromBlock(block, "")
		}
	}
	utils.LavaFormatInfo("Reading blocks Forward")
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	blocksToSaveChainTracker := uint64(10) // to avoid reading the same thing twice
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		BlocksToSave:      blocksToSaveChainTracker,
		AverageBlockTime:  10 * time.Second,
		ServerBlockMemory: 100 + blocksToSaveChainTracker,
		NewLatestCallback: readEventsFromBlock,
	}
	chainTracker, err := chaintracker.NewChainTracker(ctx, lavaChainFetcher, chainTrackerConfig)
	if err != nil {
		return utils.LavaFormatError("failed setting up chain tracker", err)
	}
	_ = chainTracker
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("events ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("events signalChan")
	}
	return nil
}

func CreateEventsCobraCommand() *cobra.Command {
	cmdEvents := &cobra.Command{
		Use:   `events blocks {--value keyword | --event event_name | --from <wallet>} [--timeout duration]`,
		Short: `reads events from the current block and backwards and prints on match criteria, after it's done reads events forward`,
		Long: `reads events from the current block and backwards and prints on match criteria, after it's done reads events forward
need to provider either provider_address or --from wallet_name
optional flag: --endpoints in order to validate provider process before submitting a stake command`,
		Example: `rpcprovider lava@myprovideraddress
		rpcprovider --from providerWallet
		rpcprovider --from providerWallet --endpoints "provider-public-grpc:port,jsonrpc,ETH1 provider-public-grpc:port,rest,LAV1"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			// handle flags, pass necessary fields
			ctx := context.Background()
			networkChainId, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}

			value, err := cmd.Flags().GetString(FlagValue)
			if err != nil {
				utils.LavaFormatFatal("failed to read value flag", err)
			}
			if value == "" {
				// look for a value that is from the --from flag
				from, err := cmd.Flags().GetString(flags.FlagFrom)
				if err != nil {
					utils.LavaFormatFatal("failed to read from flag", err)
				}
				if from != "" {
					keyName, err := sigs.GetKeyName(clientCtx)
					if err != nil {
						utils.LavaFormatFatal("failed getting key name from clientCtx, either provider the address in an argument or verify the --from wallet exists", err)
					}
					clientKey, err := clientCtx.Keyring.Key(keyName)
					if err != nil {
						return err
					}
					value = clientKey.GetAddress().String()
				}
			}
			eventName, err := cmd.Flags().GetString(FlagEventName)
			if err != nil {
				utils.LavaFormatFatal("failed to read --event flag", err)
			}
			// check at least one filter is up
			if eventName == "" && value == "" {
				utils.LavaFormatFatal("it is necessary to define either an event name or a value for lookup", err)
			}
			blocks, err := strconv.ParseInt(args[0], 0, 64)
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			if blocks < 0 {
				blocks = 0
			}
			timeout, err := cmd.Flags().GetDuration(FlagTimeout)
			if err != nil {
				utils.LavaFormatFatal("failed to fetch timeout flag", err)
			}
			utils.LavaFormatInfo("Events Lookup started", utils.Attribute{Key: "blocks", Value: blocks})
			utils.LoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)
			_ = tx.NewFactoryCLI(clientCtx, cmd.Flags())
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.Seed(time.Now().UnixNano())
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return eventsLookup(ctx, clientCtx, blocks, eventName, value)
		},
	}
	flags.AddQueryFlagsToCmd(cmdEvents)
	cmdEvents.Flags().String(flags.FlagFrom, "", "Name or address of wallet from which to read address, and look for it in value")
	cmdEvents.Flags().Duration(FlagTimeout, 5*time.Minute, "the time to listen for events, defaults to 5m")
	cmdEvents.Flags().String(FlagValue, "", "the value to look for inside all event attributes")
	cmdEvents.Flags().String(FlagEventName, "", "event name/type to look for")
	cmdEvents.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdEvents.Flags().String(common.EndpointsConfigName, "", "endpoints to check, overwrites reading it from the blockchain")
	return cmdEvents
}
