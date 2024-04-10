package validators

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/version"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	validatorMonikerFlagName = "regex"
)

type RetInfo struct {
	tombstone    int64
	jailed       int64
	missedBlocks int64
	checks       int64
	unbonded     int64
	tokens       math.Int
}

func extractValcons(codec codec.Codec, validator stakingtypes.Validator, hrp string) (valCons string, err error) {
	var pk cryptotypes.PubKey
	err = codec.UnpackAny(validator.ConsensusPubkey, &pk)
	if err != nil {
		return "", utils.LavaFormatError("failed unpacking", err)
	}
	valcons, err := bech32.ConvertAndEncode(hrp, pk.Address())
	if err != nil {
		return "", utils.LavaFormatError("failed to encode cons Address", err)
	}
	return valcons, nil
}

func checkValidatorPerformance(ctx context.Context, clientCtx client.Context, valAddr string, regex bool, blocks int64, fromBlock int64) (retInfo RetInfo, err error) {
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
	utils.LavaFormatInfo("jump blocks", utils.LogAttr("blocks", jumpBlocks))
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
	valCons := ""
	stakingQueryClient := stakingtypes.NewQueryClient(clientCtx)
	if regex {
		timeoutCtx, cancel = context.WithTimeout(ctx, 60*time.Second)
		allValidators, err := stakingQueryClient.Validators(timeoutCtx, &stakingtypes.QueryValidatorsRequest{
			Pagination: &query.PageRequest{
				Limit: 10000,
			},
		})
		cancel()
		if err != nil {
			return retInfo, utils.LavaFormatError("error reading validators", err)
		}
		re, err := regexp.Compile(valAddr)
		if err != nil {
			return retInfo, utils.LavaFormatError("failed compiling regex", err, utils.LogAttr("regex", valAddr))
		}
		valAddr = ""
		foundMoniker := ""
		for _, validator := range allValidators.GetValidators() {
			if re.MatchString(validator.Description.Moniker) {
				if valAddr == "" {
					foundMoniker = validator.Description.Moniker
					valAddr = validator.OperatorAddress
					valCons, err = extractValcons(clientCtx.Codec, validator, hrp)
					if err != nil {
						continue
					}
				} else {
					return retInfo, utils.LavaFormatError("regex matched two validators", nil, utils.LogAttr("first", foundMoniker), utils.LogAttr("second", validator.Description.Moniker))
				}
			}
		}
		if valAddr == "" {
			return retInfo, utils.LavaFormatError("failed to match a validator with regex", err, utils.LogAttr("regex", re.String()))
		}
		utils.LavaFormatInfo("found validator moniker", utils.LogAttr("moniker", foundMoniker), utils.LogAttr("address", valAddr))
	}
	utils.LavaFormatInfo("looking for validator signing info", utils.LogAttr("valAddr", valAddr), utils.LogAttr("valCons", valCons))
	timeoutCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
	validator, err := stakingQueryClient.Validator(timeoutCtx, &stakingtypes.QueryValidatorRequest{
		ValidatorAddr: valAddr,
	})
	cancel()
	if err != nil {
		return retInfo, utils.LavaFormatError("error reading validator", err)
	}
	retInfo.tokens = validator.Validator.Tokens
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
			if validatorResp.Validator.Status == stakingtypes.Bonded {
				if valCons == "" {
					valCons, err = extractValcons(clientCtx.Codec, validatorResp.Validator, hrp)
					if err != nil {
						return err
					}
				}
				timeoutCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
				signingInfo, err := slashingQueryClient.SigningInfo(timeoutCtx, &slashingtypes.QuerySigningInfoRequest{
					ConsAddress: valCons,
				})
				cancel()
				if err != nil {
					utils.LavaFormatError("failed reading signing info at height", err, utils.LogAttr("block", block), utils.LogAttr("valCons", valCons))
					continue
				}
				retInfo.missedBlocks += signingInfo.ValSigningInfo.MissedBlocksCounter
				if signingInfo.ValSigningInfo.Tombstoned {
					retInfo.tombstone += 1
				}
			} else {
				retInfo.unbonded++
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
		Use:   `validator-performance <address(string)| regex> <blocks(int)> [start_block(int)] [--node tmRPC]`,
		Short: `validator-performance checks and prints the statistics of a validator, either by an operator address or a regex`,
		Long:  `validator-performance checks and prints the statistics of a validator`,
		Example: `validator-performance lava@valoper1abcdefg 100 --node https://public-rpc.lavanet.xyz
validator-performance valida*_monik* --regex 100 --node https://public-rpc.lavanet.xyz`,
		Args: cobra.RangeArgs(2, 3),
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

			regex := viper.GetBool(validatorMonikerFlagName)
			utils.SetGlobalLoggingLevel(logLevel)
			utils.LavaFormatInfo("lavad Binary Version: " + version.Version)
			rand.InitRandomSeed()
			retInfo, err := checkValidatorPerformance(ctx, clientCtx, valAddress, regex, blocks, fromBlock)
			if err == nil {
				fmt.Printf("📄----------------------------------------✨SUMMARY✨----------------------------------------📄\n\n🔵 Validator Stats:\n🔹checks: %d\n🔹unbonded: %d\n🔹jailed: %d\n🔹missedBlocks: %d\n🔹tombstone: %d\n🔹tokens: %s\n\n", retInfo.checks, retInfo.unbonded, retInfo.jailed, retInfo.missedBlocks, retInfo.tombstone, retInfo.tokens.String())
			}
			return err
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddKeyringFlags(cmd.Flags())
	cmd.Flags().String(flags.FlagChainID, app.Name, "network chain id")
	cmd.Flags().Bool(validatorMonikerFlagName, false, "turn on regex parsing for the validator moniker instead of accepting a valoper")
	return cmd
}
