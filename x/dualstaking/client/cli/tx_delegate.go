package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate [provider] [chain-id] [validator] [amount]",
		Short: "delegate to a validator and provider using dualstaking",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			argProvider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			argChainID := args[1]
			argvalidator := args[2]
			argAmount, err := sdk.ParseCoinNormalized(args[3])
			if err != nil {
				return err
			}

			msg := types.NewMsgDelegate(
				clientCtx.GetFromAddress().String(),
				argvalidator,
				argProvider,
				argChainID,
				argAmount,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// GetValidator gets a validator that is delegated by the `from` address
// The dualstaking mecahnism makes providers delegate to a validator when they're staking.
// Assuming that the `from` address (of the clientCtx) is a staked provider address, this
// function returns a validator that the provider is delegated to (the one with the largest delegation).
func GetValidator(clientCtx client.Context) string {
	provider := clientCtx.GetFromAddress().String()
	q := stakingtypes.NewQueryClient(clientCtx)
	ctx := context.Background()
	resD, err := q.DelegatorDelegations(ctx, &stakingtypes.QueryDelegatorDelegationsRequest{DelegatorAddr: provider})

	if err == nil && len(resD.DelegationResponses) > 0 {
		delegationBiggest := resD.DelegationResponses[0]
		for _, delegationResponse := range resD.DelegationResponses {
			if sdk.AccAddress(delegationResponse.Delegation.ValidatorAddress).String() == provider {
				return delegationResponse.Delegation.ValidatorAddress
			}
			if delegationResponse.Balance.IsGTE(delegationBiggest.Balance) {
				delegationBiggest = delegationResponse
			}
		}
		if !delegationBiggest.Balance.IsZero() {
			return delegationBiggest.Delegation.ValidatorAddress
		}
	}

	return ""
}
