package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	dualstakingtypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDebugQuery() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug-query [provider] [chain-id]",
		Short: "debug query",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			provider, err := utils.ParseCLIAddress(clientCtx, args[0])
			if err != nil {
				return err
			}

			chainID := args[1]
			ctx := context.Background()

			// qureiers - rewards, dualstaking, pairing
			pairingQuerier := types.NewQueryClient(clientCtx)
			dualstakingQuerier := dualstakingtypes.NewQueryClient(clientCtx)
			rewardsQuerier := rewardstypes.NewQueryClient(clientCtx)
			distributionQuerier := distributiontypes.NewQueryClient(clientCtx)
			stakingQuerier := stakingtypes.NewQueryClient(clientCtx)
			resultStatus, err := clientCtx.Client.Status(ctx)
			if err != nil {
				return err
			}

			// gather information for printing
			var info types.QueryDebugQueryResponse

			// fill the objects

			// current block
			currentBlock := resultStatus.SyncInfo.LatestBlockHeight
			info.Block = uint64(currentBlock)

			// validator block rewards
			resBlockReward, err := rewardsQuerier.BlockReward(ctx, &rewardstypes.QueryBlockRewardRequest{})
			if err != nil {
				fmt.Printf("block reward error: %s\n", err.Error())
				return err
			}
			info.BlockReward = resBlockReward.Reward.Amount.Uint64()

			// pools
			resPools, err := rewardsQuerier.Pools(ctx, &rewardstypes.QueryPoolsRequest{})
			if err != nil {
				fmt.Printf("pools error: %s\n", err.Error())
				return err
			}
			for _, pool := range resPools.Pools {
				switch pool.Name {
				case string(rewardstypes.ValidatorsRewardsAllocationPoolName):
					info.ValAllocPoolBalance = pool.Balance.String()
				case string(rewardstypes.ValidatorsRewardsDistributionPoolName):
					info.ValDistPoolBalance = pool.Balance.String()
				case string(rewardstypes.ProvidersRewardsAllocationPool):
					info.ProviderAllocPoolBalance = pool.Balance.String()
				case string(rewardstypes.ProviderRewardsDistributionPool):
					info.ProviderDistPoolBalance = pool.Balance.String()
				}
			}

			// months left
			info.MonthsLeft = uint64(resPools.AllocationPoolMonthsLeft)

			// provider full rewards
			resDelegatorRewards, err := dualstakingQuerier.DelegatorRewards(ctx, &dualstakingtypes.QueryDelegatorRewardsRequest{
				Delegator: provider,
				Provider:  provider,
				ChainId:   chainID,
			})
			if err != nil {
				fmt.Printf("delegator rewards error: %s\n", err.Error())
				return err
			} else if len(resDelegatorRewards.Rewards) != 1 {
				info.ProviderFullReward = 0
			} else {
				info.ProviderFullReward = resDelegatorRewards.Rewards[0].Amount.AmountOf(commontypes.TokenDenom).Uint64()
			}

			// provider rewards without bonuses
			resPayout, err := pairingQuerier.ProviderMonthlyPayout(ctx, &types.QueryProviderMonthlyPayoutRequest{
				Provider: provider,
			})
			if err != nil {
				fmt.Printf("provider monthly payout error: %s\n", err.Error())
				return err
			}
			info.ProviderRewardNoBonus = resPayout.Total

			// validator reward
			resValidators, err := stakingQuerier.Validators(ctx, &stakingtypes.QueryValidatorsRequest{Status: stakingtypes.BondStatusBonded})
			if err != nil {
				fmt.Printf("validators error: %s\n", err.Error())
				return err
			}
			if len(resValidators.Validators) == 0 {
				return fmt.Errorf("no validators")
			}
			resValidatorRewards, err := distributionQuerier.ValidatorOutstandingRewards(ctx, &distributiontypes.QueryValidatorOutstandingRewardsRequest{
				ValidatorAddress: resValidators.Validators[0].OperatorAddress,
			})
			if err != nil {
				fmt.Printf("validator rewards error: %s\n", err.Error())
				return err
			}
			info.ValidatorReward = resValidatorRewards.Rewards.Rewards.AmountOf(commontypes.TokenDenom).TruncateInt().Uint64()

			// we finished gathering information, now print it
			// block,validator_rewards,val_alloc_pool,val_dist_pool,prov_alloc_pool,prov_dist_pool,months_left,provider_full_reward,provider_reward_no_bonus
			strToPrint := strings.Join([]string{
				strconv.FormatUint(info.Block, 10),
				strconv.FormatUint(info.BlockReward, 10),
				info.ValAllocPoolBalance,
				info.ValDistPoolBalance,
				info.ProviderAllocPoolBalance,
				info.ProviderDistPoolBalance,
				strconv.FormatUint(info.MonthsLeft, 10),
				strconv.FormatUint(info.ProviderFullReward, 10),
				strconv.FormatUint(info.ProviderRewardNoBonus, 10),
				strconv.FormatUint(info.ValidatorReward, 10),
			}, ",")
			fmt.Printf("%s", strToPrint)

			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
