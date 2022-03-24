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

	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	for _, relay := range msg.Relays {

		pubKey, err := relayer.RecoverPubKeyFromRelay(relay)
		if err != nil {
			return nil, fmt.Errorf("error on proof of work, bad sig")
		}
		clientAddr, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return nil, fmt.Errorf("error on proof of work, bad user address")
		}
		servicerAddr := (sdk.AccAddress)(relay.Servicer)
		if !servicerAddr.Equals(creator) {
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
		//TODO: validate the user request only holds supported apis
		// impossible atm; we do not know what apis user used

		//
		//TODO: get the pairing for the block of work and not the current one
		_, servicers, err := k.Keeper.GetPairingForClient(ctx, uint64(relay.SpecId), clientAddr)
		if err != nil {
			return nil, fmt.Errorf("error on pairing for addresses : %s and %s, err: %s", clientAddr, msg.Creator, err)
		}

		//
		// Find pairing, reward servicer, burn user tokens
		for _, servicer := range servicers {

			if servicer.Equals(servicerAddr) {
				reward := sdk.NewIntFromUint64(uint64(float64(relay.CuSum) * k.Keeper.GetCoinsPerCU(ctx)))
				rewardCoins := sdk.Coins{sdk.Coin{Denom: "stake", Amount: reward}}

				//
				// Mint to module
				err := k.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoins)
				if err != nil {
					panic(fmt.Sprintf("module failed to mint coins to give to servicer: %s", err))
				}
				//
				// Send to servicer
				err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, servicerAddr, rewardCoins)
				if err != nil {
					panic(fmt.Sprintf("failed to transfer minted new coins to servicer, %s account: %s", err, servicerAddr))
				}

				// TODO: add param to burn client, and add code for that
				//k.Keeper.GetBurnClientFactor(ctx)
				amountToBurnClient := sdk.NewIntFromUint64(uint64(float64(relay.CuSum) * 0))
				//need client Keeper and add a function BurnFromStake
				_ = amountToBurnClient

				break
			}
		}
	}

	return &types.MsgProofOfWorkResponse{}, nil
}
