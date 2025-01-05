package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/sigs"
	"github.com/lavanet/lava/v4/x/pairing/types"
	"github.com/spf13/cobra"
)

func CmdEffectivePolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "effective-policy [spec-id] [consumer/project]",
		Short: "Query to show the effective policy of a consumer taking into account plan policy and subscription policy, consumer/project can also be defined as --from walletName",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			specID := args[0]
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			var address string
			if len(args) > 1 {
				address, err = utils.ParseCLIAddress(clientCtx, args[1])
				if err != nil {
					// this should allow project names not only addresses
					address = args[1]
				}
			} else {
				clientCtxForTx, err := client.GetClientQueryContext(cmd)
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
				addressAccount, err := clientKey.GetAddress()
				if err != nil {
					return err
				}
				address = addressAccount.String()
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
