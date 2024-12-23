package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	dualstakingtypes "github.com/lavanet/lava/v4/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	"github.com/lavanet/lava/v4/x/subscription/types"
	"github.com/spf13/cobra"
)

func CmdEstimatedProviderRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimated-provider-rewards [provider] {optional: amount/delegator}",
		Short: "Calculates estimated rewards for a provider delegation.",
		Long: `Estimates the rewards a delegator will earn from a provider over one month. If no optional arguments are provided, the calculation is for the provider itself.
		The estimation considers subscription rewards, bonus rewards, IPRPC rewards, provider delegation limits, spec contribution, and tax.
		It does not factor in addons, geolocation, or provider service quality.
		
		The optional argument can either be: 
			- amount: The delegation amount for a new delegation
			- delegator: The address of an existing delegator.
		`,
		Example: ` The query can be used in 3 ways:
		1. estimated-provider-rewards <provider_address>: estimates the monthly reward of a provider.
		2. estimated-provider-rewards <provider_address> <delegator_address>: estimates the monthly reward of a delegator from a specific provider.
		3. estimated-provider-rewards <provider_address> <delegation_amount>: estimates the monthly reward of a delegator that has a specific amount of delegation to a specific provider.
		`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := types.QueryEstimatedProviderRewardsRequest{}
			req.Provider, err = utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			if len(args) == 2 {
				address, err := utils.ParseCLIAddress(clientCtx, args[1])
				if err != nil {
					req.AmountDelegator = args[1]
				} else {
					req.AmountDelegator = address
				}
			}

			res, err := queryClient.EstimatedProviderRewards(cmd.Context(), &req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdPoolRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool-rewards-breakdown",
		Short: "Calculates estimated rewards for all pools",
		Long:  `estimate the total rewards a pool will give to all providers if the month ends now`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// get all provider addresses
			provider := ""
			epochStorageEueryClient := epochstoragetypes.NewQueryClient(clientCtx)
			res, err := epochStorageEueryClient.ProviderMetaData(cmd.Context(), &epochstoragetypes.QueryProviderMetaDataRequest{Provider: provider})
			if err != nil {
				return err
			}
			addresses := []string{}

			for _, meta := range res.GetMetaData() {
				addresses = append(addresses, meta.GetProvider())
			}
			runEstimateWithRetries := func(req types.QueryEstimatedProviderRewardsRequest) (*types.QueryEstimatedRewardsResponse, error) {
				res, err := queryClient.EstimatedProviderRewards(cmd.Context(), &req)
				if err != nil {
					res, err = queryClient.EstimatedProviderRewards(cmd.Context(), &req)
					if err != nil {
						res, err = queryClient.EstimatedProviderRewards(cmd.Context(), &req)
						if err != nil {
							return nil, err
						}
					}
				}
				return res, err
			}

			summary := map[string]types.EstimatedRewardInfo{}
			total := sdk.DecCoins{}
			for idx, provider := range addresses {
				fmt.Printf("\rProgress: %d/%d", idx+1, len(addresses))
				req := types.QueryEstimatedProviderRewardsRequest{Provider: provider}
				res, err := runEstimateWithRetries(req)
				if err != nil {
					utils.LavaFormatError("failed to query provider", err, utils.Attribute{Key: "provider", Value: provider})
					continue
				}
				total = summarizeForRes(res, summary, total)
				dualStakingQueryClient := dualstakingtypes.NewQueryClient(clientCtx)
				resDel, err := dualStakingQueryClient.ProviderDelegators(cmd.Context(), &dualstakingtypes.QueryProviderDelegatorsRequest{Provider: provider})
				if err != nil {
					resDel, err = dualStakingQueryClient.ProviderDelegators(cmd.Context(), &dualstakingtypes.QueryProviderDelegatorsRequest{Provider: provider})
					if err != nil {
						resDel, err = dualStakingQueryClient.ProviderDelegators(cmd.Context(), &dualstakingtypes.QueryProviderDelegatorsRequest{Provider: provider})
						if err != nil {
							return err
						}

					}
				}
				delegations := resDel.GetDelegations()
				for idx2, del := range delegations {
					fmt.Printf("\rProgress: %d/%d %d/%d", idx+1, len(addresses), idx2+1, len(delegations))
					delegatorName := del.Delegator
					req := types.QueryEstimatedProviderRewardsRequest{Provider: provider, AmountDelegator: delegatorName}
					res, err := runEstimateWithRetries(req)
					if err != nil {
						utils.LavaFormatError("failed to query delegator rewards", err, utils.LogAttr("provider", provider), utils.LogAttr("delegator", delegatorName))
						continue
					}
					total = summarizeForRes(res, summary, total)
				}
			}
			fmt.Printf("\n---- results ----\n\n")
			info := []types.EstimatedRewardInfo{}
			for _, sumEntry := range summary {
				info = append(info, sumEntry)
			}
			printMe := &types.QueryEstimatedRewardsResponse{
				Info:  info,
				Total: total,
			}
			return clientCtx.PrintProto(printMe)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func summarizeForRes(res *types.QueryEstimatedRewardsResponse, summary map[string]types.EstimatedRewardInfo, total sdk.DecCoins) sdk.DecCoins {
	info := res.Info
	for _, entry := range info {
		if _, ok := summary[entry.Source]; !ok {
			summary[entry.Source] = entry
		} else {
			entryIn := summary[entry.Source]
			coinsArr := entry.Amount
			for _, coin := range coinsArr {
				entryIn.Amount = entryIn.Amount.Add(coin)
			}
			summary[entry.Source] = entryIn
		}
	}
	for _, coin := range res.Total {
		total = total.Add(coin)
	}
	return total
}
