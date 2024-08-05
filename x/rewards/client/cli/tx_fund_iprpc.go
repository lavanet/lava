package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdFundIprpc() *cobra.Command {
	cmd := &cobra.Command{
		Use: "fund-iprpc [spec] [duration] [coins] --from <creator>",
		Short: `fund the IPRPC pool to a specific spec with ulava or IBC wrapped tokens. The tokens will be vested for <duration> months.
		Note that the amount of coins you put is the monthly quota (it's not for the total period of time).
		Also, the tokens must include duration*min_iprpc_cost of ulava tokens (min_iprpc_cost is shown with the show-iprpc-data command)`,
		Example: `lavad tx rewards fund-iprpc ETH1 4 100000ulava,50000ibctoken --from alice
		This command will transfer 4*100000ulava and 4*50000ibctoken to the IPRPC pool to be distributed for 4 months`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			spec := args[0]
			durationStr := args[1]
			duration, err := strconv.ParseUint(durationStr, 10, 64)
			if err != nil {
				return err
			}

			fundStr := args[2]
			fund, err := sdk.ParseCoinsNormalized(fundStr)
			if err != nil {
				return err
			}

			msg := types.NewMsgFundIprpc(
				clientCtx.GetFromAddress().String(),
				spec,
				duration,
				fund,
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
