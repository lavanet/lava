package keeper

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evidenceTypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
)

// balance delegators dualstaking after potential validators slashing
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
		k.BalanceDelegator(ctx, delAddr)
	}
}
