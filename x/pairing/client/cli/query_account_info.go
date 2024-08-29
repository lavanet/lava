package cli

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/sigs"
	dualstakingtypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	projecttypes "github.com/lavanet/lava/v2/x/projects/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/v2/x/subscription/types"
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
			specQuerier := spectypes.NewQueryClient(clientCtx)
			ctx := context.Background()
			allChains, err := specQuerier.ShowAllChains(ctx, &spectypes.QueryShowAllChainsRequest{})
			if err != nil {
				return utils.LavaFormatError("failed getting key name from clientCtx, either provide the address in an argument or verify the --from wallet exists", err)
			}
			pairingQuerier := types.NewQueryClient(clientCtx)
			subscriptionQuerier := subscriptiontypes.NewQueryClient(clientCtx)
			projectQuerier := projecttypes.NewQueryClient(clientCtx)
			epochStorageQuerier := epochstoragetypes.NewQueryClient(clientCtx)
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
			for _, chainStructInfo := range allChains.ChainInfoList {
				chainID := chainStructInfo.ChainID
				response, err := pairingQuerier.Providers(ctx, &types.QueryProvidersRequest{
					ChainID:    chainID,
					ShowFrozen: true,
				})
				if err == nil && len(response.StakeEntry) > 0 {
					for _, provider := range response.StakeEntry {
						if provider.IsAddressVaultOrProvider(address) {
							if provider.StakeAppliedBlock > uint64(currentBlock) {
								info.Frozen = append(info.Frozen, provider)
							} else {
								info.Provider = append(info.Provider, provider)
							}
							break
						}
					}
				}
			}

			unstakeEntriesAllChains, err := epochStorageQuerier.StakeStorage(ctx, &epochstoragetypes.QueryGetStakeStorageRequest{
				Index: epochstoragetypes.StakeStorageKeyUnstakeConst,
			})
			if err == nil {
				if len(unstakeEntriesAllChains.StakeStorage.StakeEntries) > 0 {
					for _, unstakingProvider := range unstakeEntriesAllChains.StakeStorage.StakeEntries {
						if unstakingProvider.IsAddressVaultOrProvider(address) {
							info.Unstaked = append(info.Unstaked, unstakingProvider)
						}
					}
				}
			}

			response, err := subscriptionQuerier.Current(cmd.Context(), &subscriptiontypes.QueryCurrentRequest{
				Consumer: address,
			})

			if err == nil {
				info.Subscription = response.Sub
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
