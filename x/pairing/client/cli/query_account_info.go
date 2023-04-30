package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	projecttypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/spf13/cobra"
)

const (
	separator = "--------------------------------------------\n"
)

var _ = strconv.Itoa(0)

func CmdAccountInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account-info {[lava_address] | --from wallet}",
		Short: "Query account information on an address",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			var address string
			if len(args) == 0 {
				keyName, err := sigs.GetKeyName(clientCtx)
				if err != nil {
					utils.LavaFormatFatal("failed getting key name from clientCtx", err)
				}
				clientKey, err := clientCtx.Keyring.Key(keyName)
				if err != nil {
					return err
				}
				address = clientKey.GetAddress().String()
			} else {
				address = args[0]
			}
			specQuerier := spectypes.NewQueryClient(clientCtx)
			ctx := context.Background()
			allChains, err := specQuerier.ShowAllChains(ctx, &spectypes.QueryShowAllChainsRequest{})
			if err != nil {
				return utils.LavaFormatError("failed getting key name from clientCtx, either provider the address in an argument or verify the --from wallet exists", err)
			}
			pairingQuerier := types.NewQueryClient(clientCtx)
			subscriptionQuerier := subscriptiontypes.NewQueryClient(clientCtx)
			projectQuerier := projecttypes.NewQueryClient(clientCtx)
			epochStorageQuerier := epochstoragetypes.NewQueryClient(clientCtx)
			resultStatus, err := clientCtx.Client.Status(ctx)
			if err != nil {
				return err
			}
			currentBlock := resultStatus.SyncInfo.LatestBlockHeight

			// gather information for printing
			stakedProviderChains := []epochstoragetypes.StakeEntry{}
			unstakingProviderChains := []epochstoragetypes.StakeEntry{}
			stakedConsumerChains := []epochstoragetypes.StakeEntry{}
			var subsc *subscriptiontypes.Subscription
			var project *projecttypes.Project

			// fill the objects
			for _, chainStructInfo := range allChains.ChainInfoList {
				chainID := chainStructInfo.ChainID
				response, err := pairingQuerier.Providers(ctx, &types.QueryProvidersRequest{
					ChainID:    chainID,
					ShowFrozen: true,
				})
				if err == nil && len(response.StakeEntry) > 0 {
					for _, provider := range response.StakeEntry {
						if provider.Address == address {
							stakedProviderChains = append(stakedProviderChains, provider)
							break
						}
					}
				}

				// TODO: when userEntry is deprecated this can be removed
				userEntry, err := pairingQuerier.UserEntry(ctx, &types.QueryUserEntryRequest{
					Address: address,
					ChainID: chainID,
					Block:   uint64(currentBlock),
				})
				if err == nil {
					if userEntry.MaxCU > 0 {
						stakedConsumerChains = append(stakedConsumerChains, userEntry.Consumer)
					}
				}
			}

			unstakeEntriesAllChains, err := epochStorageQuerier.StakeStorage(ctx, &epochstoragetypes.QueryGetStakeStorageRequest{
				Index: epochstoragetypes.ProviderKey + epochstoragetypes.StakeStorageKeyUnstakeConst,
			})
			if err == nil {
				if len(unstakeEntriesAllChains.StakeStorage.StakeEntries) > 0 {
					for _, unstakingProvider := range unstakeEntriesAllChains.StakeStorage.StakeEntries {
						if unstakingProvider.Address == address {
							unstakingProviderChains = append(unstakingProviderChains, unstakingProvider)
						}
					}
				}
			}

			response, err := subscriptionQuerier.Current(cmd.Context(), &subscriptiontypes.QueryCurrentRequest{
				Consumer: address,
			})

			if err == nil {
				subsc = &response.Sub
			}

			developer, err := projectQuerier.Developer(cmd.Context(), &projecttypes.QueryDeveloperRequest{Developer: address})
			if err == nil {
				project = developer.Project
			}

			// we finished gathering information, now print it

			// print staked chains:
			output := ""

			if len(stakedProviderChains) > 0 {
				activeChains := []epochstoragetypes.StakeEntry{}
				frozenChains := []epochstoragetypes.StakeEntry{}
				for _, entry := range stakedProviderChains {
					if entry.StakeAppliedBlock > uint64(currentBlock) {
						frozenChains = append(frozenChains, entry)
					} else {
						activeChains = append(activeChains, entry)
					}
				}
				if len(activeChains) > 0 {
					output += separator + "Active Provider Chains\n" + separator
					for _, entry := range activeChains {
						output += fmt.Sprintf("ChainID: %s\n %+v\n\n", entry.Chain, entry)
					}
					output += "\n\n"
				}
				if len(frozenChains) > 0 {
					output += separator + "Frozen Provider Chains\n" + separator
					for _, entry := range frozenChains {
						output += fmt.Sprintf("ChainID: %s\n %+v\n\n", entry.Chain, entry)
					}
					output += "\n\n"
				}
			}
			if len(unstakingProviderChains) > 0 {
				output += separator + "Unstaking Chains\n" + separator
				for _, entry := range unstakingProviderChains {
					output += fmt.Sprintf("ChainID: %s\n %+v\n\n", entry.Chain, entry)
				}
				output += "\n\n"
			}
			if project != nil {
				marshaller := &jsonpb.Marshaler{}
				data, err := marshaller.MarshalToString(project)
				if err != nil {
					return err
				}
				output += separator + "Active Project\n" + separator
				output += data + "\n\n"
			}
			if subsc != nil {
				marshaller := &jsonpb.Marshaler{}
				data, err := marshaller.MarshalToString(subsc)
				if err != nil {
					return err
				}
				output += separator + "Active Subscription\n" + separator
				output += data + "\n\n"
			}

			if len(stakedConsumerChains) > 0 {
				output += separator + "Active Consumer Chains\n" + separator
				for _, entry := range stakedConsumerChains {
					output += fmt.Sprintf("ChainID: %s\n %+v\n\n", entry.Chain, entry)
				}
				output += "\n\n"
			}
			fmt.Println(output)
			return nil
		},
	}
	cmd.Flags().String(flags.FlagFrom, "", "Name or address of private key with which to sign")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
