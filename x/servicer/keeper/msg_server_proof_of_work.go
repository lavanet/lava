package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer"
	"github.com/lavanet/lava/x/servicer/types"
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
		// TODO: is this correct? spec could be disabled after the fact
		ok, _ := k.Keeper.specKeeper.IsSpecIDFoundAndActive(ctx, uint64(relay.SpecId))
		if !ok {
			return nil, fmt.Errorf("error on proof of work, spec specified: %d is inactive", relay.SpecId)
		}

		//
		//TODO: validate CU requested is valid for the user and not too big, this requires the user module
		//

		//
		//
		isValidPairing, isOverlap, err := k.Keeper.ValidatePairingForClient(
			ctx,
			uint64(relay.SpecId),
			clientAddr,
			servicerAddr,
			types.BlockNum{
				Num: uint64(relay.BlockHeight),
			},
		)
		//TODO: use isOverlap to calculate limiting the CU of previous session
		_ = isOverlap
		if err != nil {
			return nil, fmt.Errorf("error on pairing for addresses : %s and %s, block %d, err: %s", clientAddr, servicerAddr, relay.BlockHeight, err)
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

			clientBurn := k.Keeper.userKeeper.GetCoinsPerCU(ctx)
			amountToBurnClient := sdk.NewIntFromUint64(uint64(float64(relay.CuSum) * clientBurn))
			//TODO: burn stake for client
			_ = amountToBurnClient
		}

	}

	return &types.MsgProofOfWorkResponse{}, nil
}
