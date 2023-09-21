package statetracker

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"time"

	"github.com/cometbft/cometbft/abci/types"
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
)

const (
	FlagTimeout   = "timeout"
	FlagValue     = "value"
	FlagEventName = "event"
	FlagBreak     = "break"
)

func eventsLookup(ctx context.Context, clientCtx client.Context, blocks, fromBlock int64, eventName, value string, shouldBreak bool) error {
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
		sort.Slice(event.Attributes, func(i, j int) bool {
			return event.Attributes[i].Key < event.Attributes[j].Key
		})
		for _, attr := range event.Attributes {
			st += fmt.Sprintf("%s = %s, ", attr.Key, attr.Value)
		}
		return st
	}

	readEventsFromBlock := func(block int64, hash string) {
		brp, err := tryIntoTendermintRPC(clientCtx.Client)
		if err != nil {
			utils.LavaFormatFatal("invalid blockResults provider", err)
		}
		blockResults, err := brp.BlockResults(ctx, &block)
		if err != nil {
			utils.LavaFormatError("invalid blockResults status", err)
			return
		}
		for _, event := range blockResults.BeginBlockEvents {
			if eventName == "" || event.Type == eventName {
				for _, attribute := range event.Attributes {
					if value == "" || attribute.Value == value {
						utils.LavaFormatInfo("Found BBlock event", utils.Attribute{Key: "event", Value: printEvent(event)}, utils.Attribute{Key: "height", Value: block})
					}
				}
			}
		}
		transactionResults := blockResults.TxsResults
		for _, tx := range transactionResults {
			events := tx.Events
			for _, event := range events {
				if eventName == "" || event.Type == eventName {
					for _, attribute := range event.Attributes {
						if value == "" || attribute.Value == value {
							utils.LavaFormatInfo("Found Tx event", utils.Attribute{Key: "event", Value: printEvent(event)}, utils.Attribute{Key: "height", Value: block})
						}
					}
				}
			}
		}
	}

	if blocks > 0 {
		if fromBlock <= 0 {
			fromBlock = latestHeight - blocks
		}
		ticker := time.NewTicker(5 * time.Second)
		utils.LavaFormatInfo("Reading Events", utils.Attribute{Key: "from", Value: fromBlock}, utils.Attribute{Key: "to", Value: fromBlock + blocks})
		for block := fromBlock; block < fromBlock+blocks; block++ {
			readEventsFromBlock(block, "")
			// if the user aborted stop
			select {
			case <-signalChan:
				return nil
			case <-ticker.C:
				fmt.Printf("Current Block: %d\r", block)
			default:
			}
		}
	}
	lavaChainFetcher := chainlib.NewLavaChainFetcher(ctx, clientCtx)
	latestBlock, err := lavaChainFetcher.FetchLatestBlockNum(ctx)
	if err != nil {
		return utils.LavaFormatError("failed reading latest block", err)
	}
	if shouldBreak {
		return nil
	}
	utils.LavaFormatInfo("Reading blocks Forward", utils.Attribute{Key: "current", Value: latestBlock})
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
		Use:   `events <blocks(int)> [start_block(int)] {--value keyword | --event event_name | --from <wallet>} [--timeout duration]`,
		Short: `reads events from the current block and backwards and prints on match criteria, after it's done reads events forward`,
		Long: `reads events from the current block and backwards and prints on match criteria, after it's done reads events forward
blocks is the amount of blocks to read, when provided without a start_block will read the last X blocks going back from the current one, 0 will only read forward from now
start_blocks is an optional argument to specify the block you want to start reading events from, in case you have a specific block range you need
you must specify either: --value/--event/--from flags
--value & --event can be used at the same time, from & value conflict`,
		Example: `lavad test events 100 --event lava_relay_payment // show all events of the name lava_relay_payment from current-block - 100 and forwards
lavad test events 0 --from servicer1 // show all events from current block forwards that has my wallet address in one of their fields
lavad test events 100 5000 --value banana // show all events from 5000-5100 and current block forward that has in one of their fields the string banana
		`,
		Args: cobra.RangeArgs(1, 2),
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
					addr, err := clientKey.GetAddress()
					if err != nil {
						return err
					}
					value = addr.String()
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

			fromBlock := int64(-1)
			if len(args) == 2 {
				fromBlock, err = strconv.ParseInt(args[1], 0, 64)
				if err != nil {
					utils.LavaFormatFatal("failed to parse blocks as a number", err)
				}
			}

			timeout, err := cmd.Flags().GetDuration(FlagTimeout)
			if err != nil {
				utils.LavaFormatFatal("failed to fetch timeout flag", err)
			}

			shouldBreak, err := cmd.Flags().GetBool(FlagBreak)
			if err != nil {
				utils.LavaFormatFatal("failed to fetch break flag", err)
			}
			utils.LavaFormatInfo("Events Lookup started", utils.Attribute{Key: "blocks", Value: blocks})
			utils.LoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)
			_, err = tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.Seed(time.Now().UnixNano())
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return eventsLookup(ctx, clientCtx, blocks, fromBlock, eventName, value, shouldBreak)
		},
	}
	flags.AddQueryFlagsToCmd(cmdEvents)
	cmdEvents.Flags().String(flags.FlagFrom, "", "Name or address of wallet from which to read address, and look for it in value")
	cmdEvents.Flags().Duration(FlagTimeout, 5*time.Minute, "the time to listen for events, defaults to 5m")
	cmdEvents.Flags().String(FlagValue, "", "the value to look for inside all event attributes")
	cmdEvents.Flags().Bool(FlagBreak, false, "if true will break after reading the specified amount of blocks instead of listening forward")
	cmdEvents.Flags().String(FlagEventName, "", "event name/type to look for")
	cmdEvents.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdEvents.Flags().String(common.EndpointsConfigName, "", "endpoints to check, overwrites reading it from the blockchain")
	return cmdEvents
}
