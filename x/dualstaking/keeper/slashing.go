package keeper

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evidenceTypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k Keeper) HandleSlashedValidators(ctx sdk.Context, req abci.RequestBeginBlock) {
	for _, tmEvidence := range req.ByzantineValidators {
		switch tmEvidence.Type {
		case abci.MisbehaviorType_DUPLICATE_VOTE, abci.MisbehaviorType_LIGHT_CLIENT_ATTACK:
			evidence := evidenceTypes.FromABCIEvidence(tmEvidence)
			k.BalanceValidatorsDelegators(ctx, evidence.(*evidenceTypes.Equivocation))

		default:
			k.Logger(ctx).Error(fmt.Sprintf("ignored unknown evidence type: %s", tmEvidence.Type))
		}
	}
}

func (k Keeper) BalanceValidatorsDelegators(ctx sdk.Context, evidence *evidenceTypes.Equivocation) {
	consAddr := evidence.GetConsensusAddress()

	validator := k.stakingKeeper.ValidatorByConsAddr(ctx, consAddr)
	if validator == nil || validator.GetOperator().Empty() {
		return
	}

	delegators := k.stakingKeeper.GetValidatorDelegations(ctx, validator.GetOperator())
	for _, delegator := range delegators {
		delAddr := delegator.GetDelegatorAddr()
		diff, _, err := k.VerifyDelegatorBalance(ctx, delAddr)
		if err != nil {
			utils.LavaFormatError("failed to verify delegator balance after slashing", err)
			return
		}

		// if diff is zero, do nothing, this is a redelegate
		if diff.IsZero() {
			return
		} else if diff.IsPositive() {
			// less provider delegations,a delegation operation was done, delegate to empty provider
			err = k.delegate(ctx, delAddr.String(), types.EMPTY_PROVIDER, types.EMPTY_PROVIDER_CHAINID,
				sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), diff))
			if err != nil {
				utils.LavaFormatError("failed to delegate delegator balance after slashing", err)
				return
			}
		} else if diff.IsNegative() {
			// more provider delegation, unbond operation was done, unbond from providers
			err = k.UnbondUniformProviders(ctx, delAddr.String(), sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), diff.Neg()))
			if err != nil {
				utils.LavaFormatError("failed to unbond delegattor balance after slashing", err)
				return
			}
		}
	}
}
