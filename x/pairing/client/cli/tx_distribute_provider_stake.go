package cli

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v4/utils/common/types"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	"github.com/lavanet/lava/v4/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDistributeProviderStake() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "distribute-provider-stake chain,%,chain,%",
		Short:   "redistribute providers stake between all the chains it is staked on.",
		Long:    `sends batch of movestake tx to redistribute the total stake according to the users input, the total percentages must be exactly 100`,
		Example: `lavad tx pairing distribute-provider-stake chain0,33,chain1,33,chain2,34"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address := clientCtx.GetFromAddress().String()

			pairingQuerier := types.NewQueryClient(clientCtx)
			ctx := context.Background()
			response, err := pairingQuerier.Provider(ctx, &types.QueryProviderRequest{Address: address})
			if err != nil {
				return err
			}

			if len(response.StakeEntries) == 0 {
				// Check if the address is a vault by querying metadata
				epochStorageQuerier := epochstoragetypes.NewQueryClient(clientCtx)
				metadatasResponse, err := epochStorageQuerier.ProviderMetaData(ctx, &epochstoragetypes.QueryProviderMetaDataRequest{})
				if err != nil {
					return err
				}

				// If this provider has a vault set, try to use the vault address instead
				// TOSO: this is a fix until we add a way to query the vault's metadata
				for _, metadata := range metadatasResponse.MetaData {
					if metadata.Vault == address {
						// Query the vault's stake entries
						response, err = pairingQuerier.Provider(ctx, &types.QueryProviderRequest{Address: metadata.Provider})
						if err != nil {
							return err
						}
						break
					}
				}
			}

			msgs, err := CalculateDistbiruitions(address, response.StakeEntries, args[0])
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

type data struct {
	chain    string
	original math.Int
	percent  sdk.Dec
	target   math.Int
	diff     math.Int
}

func CalculateDistbiruitions(provider string, entries []epochstoragetypes.StakeEntry, distributionsArg string) ([]sdk.Msg, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("provider: %s is not staked on any chain", provider)
	}

	splitedArgs := strings.Split(distributionsArg, ",")
	if len(splitedArgs)%2 != 0 {
		return nil, fmt.Errorf("args must: chain,percent,chain,percent")
	}

	totalStake := sdk.NewCoin(commontypes.TokenDenom, sdk.ZeroInt())
	totalP := sdk.ZeroDec()
	distributions := []data{}
	// First decode the args into chain->percent map
	chainToPercent := make(map[string]sdk.Dec)
	for i := 0; i < len(splitedArgs); i += 2 {
		p, err := sdk.NewDecFromStr(splitedArgs[i+1])
		if err != nil {
			return nil, err
		}
		chainToPercent[splitedArgs[i]] = p
	}

	// Then match entries with percentages and fill the data
	for _, e := range entries {
		if p, ok := chainToPercent[e.Chain]; ok {
			distributions = append(distributions, data{chain: e.Chain, original: e.Stake.Amount, percent: p})
			totalStake = totalStake.Add(e.Stake)
			totalP = totalP.Add(p)
		}
	}

	if len(distributions) != len(entries) {
		// Print out which chains were specified vs which chains have stakes
		specifiedChains := make([]string, 0)
		for chain := range chainToPercent {
			specifiedChains = append(specifiedChains, chain)
		}
		stakedChains := make([]string, 0)
		for _, entry := range entries {
			stakedChains = append(stakedChains, entry.Chain)
		}
		return nil, fmt.Errorf("chains mismatch - specified chains: %v, staked chains: %v", specifiedChains, stakedChains)
	}

	if !totalP.Equal(sdk.NewDec(100)) {
		return nil, fmt.Errorf("total percentages must be 100, total input: %s", totalP.String())
	}

	left := totalStake
	excesses := []data{}
	deficits := []data{}
	for i := 0; i < len(distributions); i++ {
		if i == len(distributions)-1 {
			distributions[i].target = left.Amount
		} else {
			distributions[i].target = distributions[i].percent.MulInt(totalStake.Amount).QuoInt64(100).RoundInt()
		}
		left = left.SubAmount(distributions[i].target)

		if distributions[i].original.GT(distributions[i].target) {
			distributions[i].diff = distributions[i].original.Sub(distributions[i].target)
			excesses = append(excesses, distributions[i])
		} else if distributions[i].original.LT(distributions[i].target) {
			distributions[i].diff = distributions[i].target.Sub(distributions[i].original)
			deficits = append(deficits, distributions[i])
		}
	}

	// Sort excesses and deficits by points, descending order
	sort.Slice(excesses, func(i, j int) bool {
		return excesses[i].diff.GT(excesses[j].diff)
	})
	sort.Slice(deficits, func(i, j int) bool {
		return deficits[i].diff.GT(deficits[j].diff)
	})

	// Match excesses and deficits
	msgs := []sdk.Msg{}
	excessIdx, deficitIdx := 0, 0
	for excessIdx < len(excesses) && deficitIdx < len(deficits) {
		// Move the smaller of the excess or deficit
		tokensToMove := excesses[excessIdx].diff
		if excesses[excessIdx].diff.GT(deficits[deficitIdx].diff) {
			tokensToMove = deficits[deficitIdx].diff
		}

		msg := types.NewMsgMoveProviderStake(provider, excesses[excessIdx].chain, deficits[deficitIdx].chain, sdk.NewCoin(commontypes.TokenDenom, tokensToMove))
		if err := msg.ValidateBasic(); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
		excesses[excessIdx].diff = excesses[excessIdx].diff.Sub(tokensToMove)
		deficits[deficitIdx].diff = deficits[deficitIdx].diff.Sub(tokensToMove)

		// If an excess or deficit is fully resolved, move to the next one
		if excesses[excessIdx].diff.IsZero() {
			excessIdx++
		}
		if deficits[deficitIdx].diff.IsZero() {
			deficitIdx++
		}
	}

	for _, item := range deficits {
		if !item.diff.IsZero() {
			return nil, fmt.Errorf("failed to distribute provider stake")
		}
	}

	for _, item := range excesses {
		if !item.diff.IsZero() {
			return nil, fmt.Errorf("failed to distribute provider stake")
		}
	}

	return msgs, nil
}
