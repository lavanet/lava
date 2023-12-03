package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/x/dualstaking/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDelegate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate [validator] provider chain-id amount",
		Short: "delegate to a validator and provider",
		Args:  cobra.RangeArgs(3, 4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			index := 0
			argvalidator := args[index]
			if len(args) == 4 {
				index++
			} else {
				argvalidator = GetValidator(clientCtx)
			}

			argProvider := args[index]
			index++
			argChainID := args[index]
			index++
			argAmount, err := sdk.ParseCoinNormalized(args[index])
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

func GetValidator(clientCtx client.Context) string {
	provider := clientCtx.GetFromAddress().String()
	q := stakingtypes.NewQueryClient(clientCtx)
	ctx := context.Background()
	resD, err := q.DelegatorValidators(ctx, &stakingtypes.QueryDelegatorValidatorsRequest{DelegatorAddr: provider})

	if err == nil && len(resD.Validators) > 0 {
		validatorBiggest := resD.Validators[0]
		for _, validator := range resD.Validators {
			if sdk.AccAddress(validator.OperatorAddress).String() == provider {
				return validator.OperatorAddress
			}
			if validator.Tokens.GT(validatorBiggest.Tokens) {
				validatorBiggest = validator
			}
		}
		return validatorBiggest.OperatorAddress
	}

	resV, err := q.Validators(ctx, &stakingtypes.QueryValidatorsRequest{})
	if err != nil {
		panic("failed to fetch list of validators")
	}
	validatorBiggest := resV.Validators[0]
	for _, validator := range resV.Validators {
		if sdk.AccAddress(validator.OperatorAddress).String() == provider {
			return validator.OperatorAddress
		}
		if validator.Tokens.GT(validatorBiggest.Tokens) {
			validatorBiggest = validator
		}
	}
	return validatorBiggest.OperatorAddress
}
