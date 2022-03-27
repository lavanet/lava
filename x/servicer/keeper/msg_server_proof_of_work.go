package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer"
	"github.com/lavanet/lava/x/servicer/types"
	usertypes "github.com/lavanet/lava/x/user/types"
)

func (k msgServer) ProofOfWork(goCtx context.Context, msg *types.MsgProofOfWork) (*types.MsgProofOfWorkResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)

	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	for _, relay := range msg.Relays {

		pubKey, err := relayer.RecoverPubKeyFromRelay(relay)
		if err != nil {
			logger.Error("error on proof of work, bad sig")
			return nil, fmt.Errorf("error on proof of work, bad sig")
		}
		clientAddr, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			logger.Error("error on proof of work, bad user address")
			return nil, fmt.Errorf("error on proof of work, bad user address")
		}
		servicerAddr, err := sdk.AccAddressFromBech32(relay.Servicer)
		if err != nil {
			logger.Error("servicerAddr err", err)
			return nil, err
		}
		if !servicerAddr.Equals(creator) {
			logger.Error("error on proof of work, servicerAddr != creator")
			return nil, fmt.Errorf("error on proof of work, servicerAddr != creator")
		}

		//
		// TODO: add support for spec changes
		ok, _ := k.Keeper.specKeeper.IsSpecIDFoundAndActive(ctx, uint64(relay.SpecId))
		if !ok {
			return nil, fmt.Errorf("error on proof of work, spec specified: %d is inactive", relay.SpecId)
		}

		isValidPairing, isOverlap, userStake, err := k.Keeper.ValidatePairingForClient(
			ctx,
			uint64(relay.SpecId),
			clientAddr,
			servicerAddr,
			types.BlockNum{
				Num: uint64(relay.BlockHeight),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error on pairing for addresses : %s and %s, block %d, err: %s", clientAddr, servicerAddr, relay.BlockHeight, err)
		}

		sessionStart, overlapSessionStart, err := k.GetSessionStartForBlock(ctx, types.BlockNum{Num: uint64(relay.BlockHeight)})
		if err != nil {
			return nil, fmt.Errorf("error on proof of work, could not get session start for: %d err: %s", relay.BlockHeight, err)
		}
		if isOverlap {
			sessionStart = overlapSessionStart
		}
		//this prevents double spend attacks, and tracks the CU per session a client can use
		totalCUInSessionForUser, err := k.Keeper.AddSessionPayment(ctx, *sessionStart, clientAddr, servicerAddr, relay.CuSum, strconv.FormatUint(relay.SessionId, 16))
		if err != nil {
			//double spending on user detected!
			return nil, err
		}
		err = k.userKeeper.EnforceUserCUsUsageInSession(ctx, userStake, totalCUInSessionForUser)
		if err != nil {
			return nil, err
		}
		//
		if isValidPairing {
			//pairing is valid, we can pay servicer for work
			uintReward := uint64(float64(relay.CuSum) * k.Keeper.GetCoinsPerCU(ctx))
			if uintReward == 0 {
				continue
			}

			reward := sdk.NewIntFromUint64(uintReward)
			rewardCoins := sdk.Coins{sdk.Coin{Denom: "stake", Amount: reward}}

			//first check we can burn user before we give money to the servicer
			clientBurn := k.Keeper.userKeeper.GetCoinsPerCU(ctx)
			amountToBurnClient := sdk.NewIntFromUint64(uint64(float64(relay.CuSum) * clientBurn))
			spec, found := k.specKeeper.GetSpec(ctx, uint64(relay.SpecId))
			if !found {
				panic(fmt.Sprintf("failed to get spec for index: %d", relay.SpecId))
			}
			burnAmount := sdk.Coin{Amount: amountToBurnClient, Denom: "stake"}
			burnSucceeded, err2 := k.userKeeper.BurnUserStake(ctx, usertypes.SpecName{Name: spec.Name}, clientAddr, burnAmount, false)
			if err2 != nil {
				return nil, fmt.Errorf("BurnUserStake failed on user %s, amount to burn: %s, error: ", clientAddr, burnAmount, err2)
			}
			if !burnSucceeded {
				return nil, fmt.Errorf("BurnUserStake failed on user %s, did not find user, or insufficient funds: %s ", clientAddr, burnAmount)
			}

			//
			// Mint to module
			err := k.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoins)
			if err != nil {
				logger.Error("MintCoins", "err", err)
				panic(fmt.Sprintf("module failed to mint coins to give to servicer: %s", err))
			}
			//
			// Send to servicer
			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, servicerAddr, rewardCoins)
			if err != nil {
				logger.Error("SendCoinsFromModuleToAccount", "err", err, "servicerAddr", servicerAddr)
				panic(fmt.Sprintf("failed to transfer minted new coins to servicer, %s account: %s", err, servicerAddr))
			}

			//
			// TODO: save session information to prevent replay attack
			//

		}

	}

	return &types.MsgProofOfWorkResponse{}, nil
}
