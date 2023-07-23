package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdEffectivePolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "effective-policy [spec-id] [consumer/project]",
		Short: "Query to show the effective policy of a consumer taking into account plan policy and subscription policy, consumer/project can also be defined as --from walletName",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			specID := args[0]

			var address string
			if len(args) > 1 {
				address = args[1]
			} else {
				clientCtxForTx, err := client.GetClientTxContext(cmd)
				if err != nil {
					return err
				}
				keyName, err := sigs.GetKeyName(clientCtxForTx)
				if err != nil {
					utils.LavaFormatFatal("failed getting key name from clientCtx", err)
				}
				clientKey, err := clientCtxForTx.Keyring.Key(keyName)
				if err != nil {
					return err
				}
				address = clientKey.GetAddress().String()
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryEffectivePolicyRequest{
				Consumer: address,
				SpecID:   specID,
			}

			res, err := queryClient.EffectivePolicy(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().String(flags.FlagFrom, "", "wallet name or address of the developer to query for, to be used instead of [consumer/project]")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
