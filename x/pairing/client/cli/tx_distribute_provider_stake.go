package cli

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v3/utils/common/types"
	"github.com/lavanet/lava/v3/x/pairing/types"
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

			provider := clientCtx.GetFromAddress().String()

			pairingQuerier := types.NewQueryClient(clientCtx)
			ctx := context.Background()
			response, err := pairingQuerier.Provider(ctx, &types.QueryProviderRequest{Address: provider})
			if err != nil {
				return err
			}
			type data struct {
				chain    string
				original sdk.Int
				percent  int
				target   sdk.Int
				diff     sdk.Int
			}

			splitedArgs := strings.Split(args[0], ",")
			if len(splitedArgs)%2 != 0 {
				return fmt.Errorf("args must: chain,percent,chain,percent")
			}

			totalStake := sdk.NewCoin(commontypes.TokenDenom, sdk.ZeroInt())
			totalP := 0
			distributions := []data{}
			for i := 0; i < len(splitedArgs); i += 2 {
				p, err := strconv.Atoi(splitedArgs[i+1])
				if err != nil {
					return err
				}
				for _, e := range response.StakeEntries {
					if splitedArgs[i] == e.Chain {
						distributions = append(distributions, data{chain: e.Chain, original: e.Stake.Amount, percent: p})
						totalStake = totalStake.Add(e.Stake)
						totalP += p
					}
				}
			}

			if len(distributions) != len(response.StakeEntries) {
				return fmt.Errorf("must specify percentages for all chains the provider is staked on")
			}
			if totalP != 100 {
				return fmt.Errorf("total percentages must be 100")
			}

			left := totalStake
			excesses := []data{}
			deficits := []data{}
			for i := 0; i < len(distributions); i++ {
				if i == len(distributions)-1 {
					distributions[i].target = left.Amount
				} else {
					distributions[i].target = totalStake.Amount.MulRaw(int64(distributions[i].percent)).QuoRaw(100)
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
					return err
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
					return fmt.Errorf("failed to distribute provider stake")
				}
			}

			for _, item := range excesses {
				if !item.diff.IsZero() {
					return fmt.Errorf("failed to distribute provider stake")
				}
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
