package statetracker

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/goccy/go-json"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/lavanet/lava/v4/app"
	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/rpcprovider/rewardserver"
	updaters "github.com/lavanet/lava/v4/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	"github.com/lavanet/lava/v4/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	"github.com/spf13/cobra"
)

const (
	FlagTimeout                 = "timeout"
	FlagValue                   = "value"
	FlagEventName               = "event"
	FlagBreak                   = "break"
	FlagHasAttributeName        = "has-attribute"
	FlagShowAttributeName       = "show-attribute"
	FlagReset                   = "reset"
	FlagDisableInteractiveShell = "disable-interactive"
)

func eventsLookup(ctx context.Context, clientCtx client.Context, blocks, fromBlock int64, eventName, value string, shouldBreak bool, hasAttributeName string, showAttributeName string, disableInteractive bool) error {
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
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	readEventsFromBlock := func(blockFrom int64, blockTo int64, hash string) {
		for block := blockFrom; block < blockTo; block++ {
			queryInst := updaters.NewStateQueryAccessInst(clientCtx)
			blockResults, err := queryInst.BlockResults(ctx, &block)
			if err != nil {
				utils.LavaFormatError("invalid blockResults status", err)
				return
			}
			for _, event := range blockResults.BeginBlockEvents {
				checkEventForShow(eventName, event, hasAttributeName, value, block, showAttributeName)
			}
			transactionResults := blockResults.TxsResults
			for _, tx := range transactionResults {
				events := tx.Events
				for _, event := range events {
					checkEventForShow(eventName, event, hasAttributeName, value, block, showAttributeName)
				}
			}
			select {
			case <-signalChan:
				return
			case <-ticker.C:
				if !disableInteractive {
					fmt.Printf("Current Block: %d\r", block)
				}
			default:
			}
		}
	}

	if blocks > 0 {
		if fromBlock <= 0 {
			fromBlock = latestHeight - blocks
		}
		utils.LavaFormatInfo("Reading Events", utils.Attribute{Key: "from", Value: fromBlock}, utils.Attribute{Key: "to", Value: fromBlock + blocks})
		readEventsFromBlock(fromBlock, fromBlock+blocks, "")
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
	chainTracker.StartAndServe(ctx)
	_ = chainTracker
	select {
	case <-ctx.Done():
		utils.LavaFormatInfo("events ctx.Done")
	case <-signalChan:
		utils.LavaFormatInfo("events signalChan")
	}
	return nil
}

func exportPaymentsToCSV(data ProviderRewards, fromBlock int64, toBlock int64, currentBlock int64, tryLoadingPreviousInformation bool, resetState bool) (int64, ProviderRewards) {
	// if we don't have reset state, and this is the first loop iteration (first write)
	// we want to attempt reading the csv file from disk, if it exists..
	// if it does, attempt reading the block information and skipping block reads.
	skipBlocks := int64(0) // will indicate the for loop to skip the amount of blocks we already parsed.
	newData := data
	fileNameEnding := "_relay_payments.csv"
	for chainId := range data {
		fileName := chainId + fileNameEnding
		if !resetState && tryLoadingPreviousInformation {
			// utils.LavaFormatDebug("")
			if _, err := os.Stat(fileName); os.IsNotExist(err) {
				utils.LavaFormatInfo("did not find previous csv files to attempt reload")
				continue
			}
			// Open the CSV file
			file, err := os.Open(fileName)
			if err != nil {
				log.Fatalf("Failed to open the CSV file: %v", err)
			}
			defer file.Close()

			// Create a new CSV reader
			reader := csv.NewReader(file)

			// Read all records from the CSV
			records, err := reader.ReadAll()
			if err != nil {
				// by deferring the fatal we make sure file.close will run.
				defer log.Fatalf("Failed to read the CSV file: %v", err)
				return 0, newData
			}
			if len(records) == 0 {
				continue
			}
			successful := false
			// Create a slice to hold the data
			for i, record := range records {
				if i == 0 {
					// Skip the header row
					continue
				}
				cu, ok1 := strconv.ParseUint(record[2], 10, 64)
				numberOfRelay, ok2 := strconv.ParseUint(record[3], 10, 64)
				startBlock, ok3 := strconv.ParseUint(record[4], 10, 64)
				_, ok4 := strconv.ParseUint(record[5], 10, 64)
				latestParsedBlock, ok5 := strconv.ParseUint(record[6], 10, 64)

				if int64(startBlock) != fromBlock {
					utils.LavaFormatWarning("start block on csv doesnt match current start block, cant use info. starting from scratch", nil, utils.LogAttr("start_block", fromBlock), utils.LogAttr("start_block_in_file", startBlock), utils.LogAttr("fileName", fileName))
					break
				}
				if int64(latestParsedBlock) > toBlock {
					utils.LavaFormatWarning("latest parsed block on csv > to block. starting from scratch", nil, utils.LogAttr("latestParsedBlock", latestParsedBlock), utils.LogAttr("toBlock", toBlock), utils.LogAttr("fileName", fileName))
					break
				}

				if ok1 != nil || ok2 != nil || ok3 != nil || ok4 != nil || ok5 != nil {
					utils.LavaFormatError("failed converting one of the records", nil, utils.LogAttr("record", record))
					break
				}
				chainId := record[0]
				providerAddress := record[1]

				_, ok := newData[chainId]
				if !ok {
					newData[chainId] = make(map[string]*providerStats)
				}
				newData[chainId][providerAddress] = &providerStats{cuSum: cu, totalNumberOfRelays: numberOfRelay}
				successful = true
				skipBlocks = int64(latestParsedBlock)
			}
			if successful {
				utils.LavaFormatInfo("Successfully loaded information from csv", utils.LogAttr("filename", fileName), utils.LogAttr("skip_blocks", skipBlocks))
			}
		}
	}

	for chainId, chainData := range newData {
		fileName := chainId + fileNameEnding

		file, err := os.Create(fileName)
		if err != nil {
			utils.LavaFormatError("failed creating file", err, utils.LogAttr("name", fileName))
			continue
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Write CSV data
		values := [][]string{{"chain_id", "provider_address", "cu", "number_of_relay", "start_block", "end_block", "latest_parsed_block"}}
		for providerAddress, providerStats := range chainData {
			values = append(values, []string{chainId, providerAddress, strconv.FormatUint(providerStats.cuSum, 10), strconv.FormatUint(providerStats.totalNumberOfRelays, 10), strconv.FormatInt(fromBlock, 10), strconv.FormatInt(toBlock, 10), strconv.FormatInt(currentBlock, 10)})
		}
		if err := writer.WriteAll(values); err != nil {
			utils.LavaFormatError("failed WriteAll file", err, utils.LogAttr("name", fileName))
			continue
		}
	}
	return skipBlocks, newData
}

type providerStats struct {
	cuSum               uint64
	totalNumberOfRelays uint64
}

// per chain per provider, accumulated cu
type ProviderRewards map[string]map[string]*providerStats

func paymentsLookup(ctx context.Context, clientCtx client.Context, blockStart, blockEnd int64, resetState bool) error {
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
	if latestHeight < blockStart {
		return utils.LavaFormatError("requested blocks is bigger than latest block height", nil, utils.Attribute{Key: "blockStart", Value: blockStart}, utils.Attribute{Key: "latestHeight", Value: latestHeight})
	}
	if latestHeight < blockEnd {
		return utils.LavaFormatError("requested blocks is bigger than latest block height", nil, utils.Attribute{Key: "blockEnd", Value: blockEnd}, utils.Attribute{Key: "latestHeight", Value: latestHeight})
	}
	utils.LavaFormatDebug("chain latest block", utils.LogAttr("block", latestHeight))

	providerRewards := make(ProviderRewards)
	skipToBlock := int64(0) // used when loading csv from disk.
	for block := blockStart; block < blockEnd; block++ {
		if block <= skipToBlock {
			utils.LavaFormatInfo("Skipping blocks due to csv info found on disk", utils.LogAttr("block", block))
			continue
		}
		utils.LavaFormatInfo("fetching block", utils.LogAttr("block", block))
		queryInst := updaters.NewStateQueryAccessInst(clientCtx)
		var blockResults *coretypes.ResultBlockResults
		for retry := 0; retry < 3; retry++ {
			ctxWithTimeout, cancelContextWithTimeout := context.WithTimeout(ctx, time.Second*30)
			blockResults, err = queryInst.BlockResults(ctxWithTimeout, &block)
			cancelContextWithTimeout()
			if err != nil {
				utils.LavaFormatWarning("@@@@ failed fetching block results will retry", err, utils.LogAttr("block_number", block))
				continue
			}
			break
		}
		if blockResults == nil {
			utils.LavaFormatError("Failed fetching block results 3 times, continuing...", err, utils.LogAttr("block_number", block))
			continue
		}
		var payments []*rewardserver.PaymentRequest
		transactionResults := blockResults.TxsResults
		for _, tx := range transactionResults {
			events := tx.Events
			for _, event := range events {
				if event.Type == utils.EventPrefix+pairingtypes.RelayPaymentEventName {
					paymentList, err := rewardserver.BuildPaymentFromRelayPaymentEvent(event, block)
					if err != nil {
						utils.LavaFormatError("failed relay_payment_event parsing", err, utils.Attribute{Key: "event", Value: event}, utils.Attribute{Key: "block", Value: block})
						continue
					}

					utils.LavaFormatTrace("relay_payment_event", utils.Attribute{Key: "payment", Value: paymentList})
					payments = append(payments, paymentList...)
				}
			}
		}

		utils.LavaFormatDebug("Parsing payment", utils.LogAttr("block", block), utils.LogAttr("number_of_payments", len(payments)))
		for _, payment := range payments {
			_, ok := providerRewards[payment.ChainID]
			if !ok {
				providerRewards[payment.ChainID] = make(map[string]*providerStats)
			}
			_, okAddress := providerRewards[payment.ChainID][payment.ProviderAddress]
			if !okAddress {
				providerRewards[payment.ChainID][payment.ProviderAddress] = &providerStats{}
			}
			providerRewards[payment.ChainID][payment.ProviderAddress].cuSum += payment.CU
			providerRewards[payment.ChainID][payment.ProviderAddress].totalNumberOfRelays += payment.RelayNumber
		}
		skipToBlock, providerRewards = exportPaymentsToCSV(providerRewards, blockStart, blockEnd, block, block == blockStart, resetState)
		utils.LavaFormatDebug("saved info to file for block", utils.LogAttr("block", block))
	}

	utils.LavaFormatDebug("finished test")
	return nil
}

func checkEventForShow(eventName string, event types.Event, hasAttributeName string, value string, block int64, showAttributeName string) {
	printEvent := func(event types.Event, showAttributeName string) string {
		attributesFilter := map[string]struct{}{}
		if showAttributeName != "" {
			attributes := strings.Split(showAttributeName, " ")
			for _, attr := range attributes {
				attributesFilter[attr] = struct{}{}
			}
		}
		passFilter := func(attr types.EventAttribute) bool {
			if len(attributesFilter) == 0 {
				return true
			}
			for attrName := range attributesFilter {
				if strings.Contains(attr.Key, attrName) {
					return true
				}
			}
			return false
		}
		st := event.Type + ": "
		sort.Slice(event.Attributes, func(i, j int) bool {
			return event.Attributes[i].Key < event.Attributes[j].Key
		})
		stmore := ""
		for _, attr := range event.Attributes {
			if passFilter(attr) {
				stmore += fmt.Sprintf("%s = %s, ", attr.Key, attr.Value)
			}
		}
		if stmore == "" {
			return ""
		}
		return st + stmore
	}
	if eventName == "" || strings.Contains(event.Type, eventName) {
		printEventTriggerValue := false
		printEventTriggerHasAttr := false
		printEventAttribute := ""
		for _, attribute := range event.Attributes {
			if hasAttributeName == "" || strings.Contains(attribute.Key, hasAttributeName) {
				printEventTriggerHasAttr = true
			}
			if value == "" || strings.Contains(attribute.Value, value) {
				printEventTriggerValue = true
			}
		}
		if printEventTriggerHasAttr && printEventTriggerValue && printEventAttribute == "" {
			printEventData := printEvent(event, showAttributeName)
			if printEventData != "" {
				utils.LavaFormatInfo("Found event", utils.Attribute{Key: "event", Value: printEventData}, utils.Attribute{Key: "height", Value: block})
			}
		}
	}
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
			hasAttirbuteName, err := cmd.Flags().GetString(FlagHasAttributeName)
			if err != nil {
				utils.LavaFormatFatal("failed to read --attribute flag", err)
			}
			showAttirbuteName, err := cmd.Flags().GetString(FlagShowAttributeName)
			if err != nil {
				utils.LavaFormatFatal("failed to read --attribute flag", err)
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

			disableInteractive, err := cmd.Flags().GetBool(FlagDisableInteractiveShell)
			if err != nil {
				utils.LavaFormatFatal("failed to fetch DisableInteractive flag", err)
			}
			utils.LavaFormatInfo("Events Lookup started", utils.Attribute{Key: "blocks", Value: blocks})
			utils.SetGlobalLoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)
			_, err = tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.InitRandomSeed()
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return eventsLookup(ctx, clientCtx, blocks, fromBlock, eventName, value, shouldBreak, hasAttirbuteName, showAttirbuteName, disableInteractive)
		},
	}
	flags.AddQueryFlagsToCmd(cmdEvents)
	flags.AddKeyringFlags(cmdEvents.Flags())
	cmdEvents.Flags().String(flags.FlagFrom, "", "Name or address of wallet from which to read address, and look for it in value")
	cmdEvents.Flags().Duration(FlagTimeout, 5*time.Minute, "the time to listen for events, defaults to 5m")
	cmdEvents.Flags().String(FlagValue, "", "used to show only events that has this value in one of the attributes")
	cmdEvents.Flags().Bool(FlagBreak, false, "if true will break after reading the specified amount of blocks instead of listening forward")
	cmdEvents.Flags().String(FlagEventName, "", "event name/type to look for")
	cmdEvents.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdEvents.Flags().String(FlagHasAttributeName, "", "only show events containing specific attribute name")
	cmdEvents.Flags().String(FlagShowAttributeName, "", "only show a specific attribute name, and no other attributes")
	cmdEvents.Flags().Bool(FlagDisableInteractiveShell, false, "a flag to disable the shell printing interactive prints, used when scripting the command")
	return cmdEvents
}

func CreateRelayPaymentCSVCobraCommand() *cobra.Command {
	cmdEvents := &cobra.Command{
		Use:   `relay-payment-csv [block_start(int)] [block_end(int)]`,
		Short: `reads relay payment events from the start block to end block, creating a reward csv for all chains`,
		Long:  `reads relay payment events from the start block to end block, creating a reward csv for all chains`,
		Example: `lavap test relay-payment-csv 100 200
		`,
		Args: cobra.RangeArgs(2, 2),
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
			resetState, err := cmd.Flags().GetBool(FlagReset)
			if err != nil {
				utils.LavaFormatFatal("failed to read reset flag", err)
			}

			blockStart, err := strconv.ParseInt(args[0], 0, 64)
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			utils.LavaFormatInfo("block start", utils.LogAttr("block_start", blockStart))
			if blockStart < 0 {
				utils.LavaFormatFatal("block start is below zero", nil, utils.LogAttr("blockStart", blockStart))
			}

			blockEnd, err := strconv.ParseInt(args[1], 0, 64)
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			utils.LavaFormatInfo("block end", utils.LogAttr("block_end", blockEnd))
			if blockEnd < 0 {
				utils.LavaFormatFatal("block end is below zero", nil, utils.LogAttr("blockEnd", blockEnd))
			}

			utils.SetGlobalLoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)
			_, err = tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.InitRandomSeed()
			return paymentsLookup(ctx, clientCtx, blockStart, blockEnd, resetState)
		},
	}
	flags.AddQueryFlagsToCmd(cmdEvents)
	flags.AddKeyringFlags(cmdEvents.Flags())
	cmdEvents.Flags().String(flags.FlagFrom, "", "Name or address of wallet from which to read address, and look for it in value")
	cmdEvents.Flags().Duration(FlagTimeout, 5*time.Minute, "the time to listen for events, defaults to 5m")
	cmdEvents.Flags().Bool(FlagReset, false, "if true will remove existing information and replace it with new, meaning it wont try to load existing information from the csv file.")
	cmdEvents.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmdEvents.Flags().String(FlagHasAttributeName, "", "only show events containing specific attribute name")
	cmdEvents.Flags().String(FlagShowAttributeName, "", "only show a specific attribute name, and no other attributes")
	return cmdEvents
}

func CreateTxCounterCobraCommand() *cobra.Command {
	cmdTxCounter := &cobra.Command{
		Use:   `txcounter  [number_of_days_to_count(int)] [average_block_time_in_seconds(int)]`,
		Short: `txcounter  [number_of_days_to_count(int)] [average_block_time_in_seconds(int)]`,
		Long: `txcounter  [number_of_days_to_count(int)] [average_block_time_in_seconds(int)] counting the number of
transactions for a certain amount of days given the average block time for that chain`,
		Example: `lavad test txcounter 1 15 -- will count 1 day worth of blocks where each block is 15 seconds`,
		Args:    cobra.RangeArgs(1, 2),
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

			numberOfDays := int64(-1)
			if len(args) == 2 {
				numberOfDays, err = strconv.ParseInt(args[0], 0, 64)
				if err != nil {
					utils.LavaFormatFatal("failed to parse blocks as a number", err)
				}
			}

			blockTime, err := strconv.ParseInt(args[1], 0, 64)
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			if blockTime < 0 {
				blockTime = 0
			}

			utils.LavaFormatInfo("Events Lookup started", utils.Attribute{Key: "blocks", Value: blockTime})
			utils.SetGlobalLoggingLevel(logLevel)
			clientCtx = clientCtx.WithChainID(networkChainId)
			_, err = tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.InitRandomSeed()
			return countTransactionsPerDay(ctx, clientCtx, blockTime, numberOfDays)
		},
	}
	flags.AddQueryFlagsToCmd(cmdTxCounter)
	flags.AddKeyringFlags(cmdTxCounter.Flags())
	cmdTxCounter.Flags().String(flags.FlagFrom, "", "Name or address of wallet from which to read address, and look for it in value")
	cmdTxCounter.Flags().Duration(FlagTimeout, 5*time.Minute, "the time to listen for events, defaults to 5m")
	cmdTxCounter.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	return cmdTxCounter
}

func countTransactionsPerDay(ctx context.Context, clientCtx client.Context, blockTime, numberOfDays int64) error {
	resultStatus, err := clientCtx.Client.Status(ctx)
	if err != nil {
		return err
	}
	latestHeight := resultStatus.SyncInfo.LatestBlockHeight
	// 1 block for blockTime (lets say 15 seconds)
	// number of seconds in a day: 24 * 60 * 60
	numberOfSecondsInADay := int64(24 * 60 * 60)
	numberOfBlocksInADay := numberOfSecondsInADay / blockTime
	utils.LavaFormatInfo("Starting counter",
		utils.LogAttr("latest_block", latestHeight),
		utils.LogAttr("numberOfSecondsInADay", numberOfSecondsInADay),
		utils.LogAttr("numberOfBlocksInADay", numberOfBlocksInADay),
		utils.LogAttr("starting_block", latestHeight-numberOfBlocksInADay),
	)

	queryInst := updaters.NewStateQueryAccessInst(clientCtx)
	// i is days
	// j are blocks in that day
	// starting from current day and going backwards
	var wg sync.WaitGroup
	totalTxPerDay := &common.SafeSyncMap[int64, int]{}

	// Process each day from the earliest to the latest
	for i := int64(1); i <= numberOfDays; i++ {
		startBlock := latestHeight - (numberOfBlocksInADay * numberOfDays) + (numberOfBlocksInADay * (i - 1)) + 1
		endBlock := latestHeight - (numberOfBlocksInADay * numberOfDays) + (numberOfBlocksInADay * i)

		utils.LavaFormatInfo("Parsing day", utils.LogAttr("Day", i), utils.LogAttr("starting block", startBlock), utils.LogAttr("ending block", endBlock))

		// Process blocks in batches of 20
		for j := startBlock; j < endBlock; j += 20 {
			// Calculate the end of the batch
			end := j + 20
			if end > endBlock {
				end = endBlock
			}

			// Determine how many routines to start (could be less than 20 near the end of the loop)
			count := (end - j)

			// Add the count of goroutines to be waited on
			wg.Add(int(count))

			// Launch goroutines for each block in the current batch
			for k := j; k < end; k++ {
				go func(k int64) {
					defer wg.Done()
					ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
					defer cancel()
					blockResults, err := queryInst.BlockResults(ctxWithTimeout, &k)
					if err != nil {
						utils.LavaFormatError("invalid blockResults status", err)
						return
					}
					transactionResults := blockResults.TxsResults
					utils.LavaFormatInfo("Number of tx for block", utils.LogAttr("_routine", end-k), utils.LogAttr("block_number", k), utils.LogAttr("number_of_tx", len(transactionResults)))
					// Update totalTxPerDay safely
					actual, loaded, err := totalTxPerDay.LoadOrStore(i, len(transactionResults))
					if err != nil {
						utils.LavaFormatError("failed to load or store", err)
						return
					}
					if loaded {
						totalTxPerDay.Store(i, actual+len(transactionResults))
					}
				}(k)
			}

			// Wait for all goroutines of the current batch to complete
			utils.LavaFormatInfo("Waiting routine batch to finish", utils.LogAttr("block_from", j), utils.LogAttr("block_to", end))
			wg.Wait()
		}
	}

	// Log the transactions per day results
	totalTxPerDay.Range(func(day int64, totalTx int) bool {
		utils.LavaFormatInfo("transactions per day results", utils.LogAttr("Day", day), utils.LogAttr("totalTx", totalTx))
		return true // continue iteration
	})

	// Prepare the JSON data
	jsonData := make(map[string]int)
	totalTxPerDay.Range(func(day int64, totalTx int) bool {
		date := time.Now().AddDate(0, 0, -int(day)+1).Format("2006-01-02")
		dateKey := fmt.Sprintf("date_%s", date)
		jsonData[dateKey] = totalTx
		return true
	})

	// Convert the JSON data to JSON format
	jsonBytes, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	// Write JSON data to a file
	fileName := "dates.json"
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonBytes)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	utils.LavaFormatInfo("JSON data has been written to:" + fileName)
	return nil

	// "https://testnet2-rpc.lavapro.xyz:443/"
}
