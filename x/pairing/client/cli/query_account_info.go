package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/sigs"
	dualstakingtypes "github.com/lavanet/lava/v4/x/dualstaking/types"
	"github.com/lavanet/lava/v4/x/pairing/types"
	projecttypes "github.com/lavanet/lava/v4/x/projects/types"
	subscriptiontypes "github.com/lavanet/lava/v4/x/subscription/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAccountInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account-info {[lava_address] | --from wallet}",
		Short: "Query account information on an address",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			var address string
			if len(args) == 0 {
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
				addressAccount, err := clientKey.GetAddress()
				if err != nil {
					return err
				}
				address = addressAccount.String()
			} else {
				address, err = utils.ParseCLIAddress(clientCtx, args[0])
				if err != nil {
					return err
				}
			}
			ctx := context.Background()

			pairingQuerier := types.NewQueryClient(clientCtx)
			subscriptionQuerier := subscriptiontypes.NewQueryClient(clientCtx)
			projectQuerier := projecttypes.NewQueryClient(clientCtx)
			dualstakingQuerier := dualstakingtypes.NewQueryClient(clientCtx)
			stakingQuerier := stakingtypes.NewQueryClient(clientCtx)
			resultStatus, err := clientCtx.Client.Status(ctx)
			if err != nil {
				return err
			}
			currentBlock := resultStatus.SyncInfo.LatestBlockHeight

			// gather information for printing
			var info types.QueryAccountInfoResponse

			// fill the objects

			response, err := pairingQuerier.Provider(ctx, &types.QueryProviderRequest{
				Address: address,
			})
			if err == nil {
				for _, provider := range response.StakeEntries {
					if provider.StakeAppliedBlock > uint64(currentBlock) {
						info.Frozen = append(info.Frozen, provider)
					} else {
						info.Provider = append(info.Provider, provider)
					}
				}
			}

			subresponse, err := subscriptionQuerier.Current(cmd.Context(), &subscriptiontypes.QueryCurrentRequest{
				Consumer: address,
			})

			if err == nil {
				info.Subscription = subresponse.Sub
			}

			developer, err := projectQuerier.Developer(cmd.Context(), &projecttypes.QueryDeveloperRequest{Developer: address})
			if err == nil {
				info.Project = developer.Project
			}

			providers, err := dualstakingQuerier.DelegatorProviders(cmd.Context(), &dualstakingtypes.QueryDelegatorProvidersRequest{Delegator: address, WithPending: true})
			if err == nil {
				info.DelegationsProviders = providers.Delegations
			}

			var totalDelegations uint64
			for _, p := range providers.Delegations {
				totalDelegations += p.Amount.Amount.Uint64()
			}
			info.TotalDelegations = totalDelegations

			validators, err := stakingQuerier.DelegatorDelegations(ctx, &stakingtypes.QueryDelegatorDelegationsRequest{DelegatorAddr: address})
			if err == nil {
				info.DelegationsValidators = validators.DelegationResponses
			}

			// we finished gathering information, now print it
			return clientCtx.PrintProto(&info)
		},
	}
	cmd.Flags().String(flags.FlagFrom, "", "Name or address of private key with which to sign")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
