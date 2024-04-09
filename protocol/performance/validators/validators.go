package validators

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/version"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	"github.com/spf13/cobra"
)

type RetInfo struct {
	tombstone    int64
	jailed       int64
	missedBlocks int64
	checks       int64
}

func checkValidatorPerformance(ctx context.Context, clientCtx client.Context, valAddr string, blocks int64, fromBlock int64) (retInfo RetInfo, err error) {
	retInfo = RetInfo{}
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	resultStatus, err := clientCtx.Client.Status(ctx)
	if err != nil {
		return retInfo, err
	}
	latestHeight := resultStatus.SyncInfo.LatestBlockHeight
	if latestHeight < blocks {
		return retInfo, utils.LavaFormatError("requested blocks is bigger than latest block height", nil, utils.Attribute{Key: "requested", Value: blocks}, utils.Attribute{Key: "latestHeight", Value: latestHeight})
	}
	slashingQueryClient := slashingtypes.NewQueryClient(clientCtx)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	params, err := slashingQueryClient.Params(timeoutCtx, &slashingtypes.QueryParamsRequest{})
	cancel()
	if err != nil {
		return retInfo, utils.LavaFormatError("invalid slashing params query", err)
	}
	jumpBlocks := params.Params.SignedBlocksWindow
	timeoutCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
	signingInfos, err := slashingQueryClient.SigningInfos(timeoutCtx, &slashingtypes.QuerySigningInfosRequest{})
	cancel()
	if err != nil {
		return retInfo, utils.LavaFormatError("invalid slashing signing infos query", err)
	}
	exampleConsAddress := signingInfos.Info[0].Address
	hrp, _, err := bech32.DecodeAndConvert(exampleConsAddress)
	if err != nil {
		return retInfo, utils.LavaFormatError("error decoding hrp", err)
	}
	ticker := time.NewTicker(3 * time.Second)
	readEventsFromBlock := func(blockFrom int64, blockTo int64) error {
		for block := blockFrom; block < blockTo; block += jumpBlocks {
			select {
			case <-signalChan:
				return nil
			case <-ticker.C:
				fmt.Printf("Current Block: %d\r", block)
			default:
			}
			clientCtxWithHeight := clientCtx.WithHeight(block)
			stakingQueryClient := stakingtypes.NewQueryClient(clientCtxWithHeight)
			slashingQueryClient := slashingtypes.NewQueryClient(clientCtxWithHeight)
			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			// signingInfos, err := stakingQueryClient.SigningInfos(timeoutCtx, &slashingtypes.QuerySigningInfosRequest{})
			validatorResp, err := stakingQueryClient.Validator(timeoutCtx, &stakingtypes.QueryValidatorRequest{
				ValidatorAddr: valAddr,
			})
			cancel()
			if err != nil {
				utils.LavaFormatWarning("failed to find validator at height", err, utils.LogAttr("block", block))
				continue
			}
			if validatorResp.Validator.Jailed {
				retInfo.jailed++
			}
			var pk cryptotypes.PubKey
			if err := clientCtx.Codec.UnmarshalInterfaceJSON(validatorResp.Validator.ConsensusPubkey.GetValue(), &pk); err != nil {
				return err
			}
			valcons, err := bech32.ConvertAndEncode(hrp, pk.Address())
			if err != nil {
				return utils.LavaFormatError("failed to encode cons Address", err)
			}
			timeoutCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
			signingInfo, err := slashingQueryClient.SigningInfo(timeoutCtx, &slashingtypes.QuerySigningInfoRequest{
				ConsAddress: valcons,
			})
			cancel()
			if err != nil {
				utils.LavaFormatError("failed reading signing info at height", err, utils.LogAttr("block", block))
				continue
			}
			retInfo.missedBlocks += signingInfo.ValSigningInfo.MissedBlocksCounter
			if signingInfo.ValSigningInfo.Tombstoned {
				retInfo.tombstone += 1
			}
			retInfo.checks += 1
		}
		return nil
	}

	if blocks > 0 {
		if fromBlock <= 0 {
			fromBlock = latestHeight - blocks
		}
		utils.LavaFormatInfo("Reading validator performance on blocks", utils.Attribute{Key: "from", Value: fromBlock}, utils.Attribute{Key: "to", Value: fromBlock + blocks})
		readEventsFromBlock(fromBlock, fromBlock+blocks)
	}
	return retInfo, nil
}

func CreateValidatorsPerformanceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     `validator-performance <address(string)> <blocks(int)> [start_block(int)] [--node tmRPC]`,
		Short:   `validator-performance checks and prints the statistics of a validator`,
		Long:    `validator-performance checks and prints the statistics of a validator`,
		Example: `validator-performance lava@valcons1abcdefg 100 --node https://public-rpc.lavanet.xyz`,
		Args:    cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			// handle flags, pass necessary fields
			ctx := context.Background()
			if err != nil {
				return err
			}
			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}

			valAddress := args[0]
			blocks, err := strconv.ParseInt(args[1], 0, 64)
			if err != nil {
				utils.LavaFormatFatal("failed to parse blocks as a number", err)
			}
			if blocks < 0 {
				blocks = 0
			}

			fromBlock := int64(-1)
			if len(args) == 3 {
				fromBlock, err = strconv.ParseInt(args[2], 0, 64)
				if err != nil {
					utils.LavaFormatFatal("failed to parse blocks as a number", err)
				}
			}

			utils.SetGlobalLoggingLevel(logLevel)
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.InitRandomSeed()
			retInfo, err := checkValidatorPerformance(ctx, clientCtx, valAddress, blocks, fromBlock)
			if err == nil {
				fmt.Printf("📄----------------------------------------✨SUMMARY✨----------------------------------------📄\n\n🔵 Validator Stats:\n🔹checks: %d\n🔹jailed: %d\n🔹missedBlocks: %d\n🔹tombstone: %d\n\n", retInfo.checks, retInfo.jailed, retInfo.missedBlocks, retInfo.tombstone)
			}
			return err
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddKeyringFlags(cmd.Flags())
	cmd.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	return cmd
}
