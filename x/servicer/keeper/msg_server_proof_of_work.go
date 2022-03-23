package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k msgServer) ProofOfWork(goCtx context.Context, msg *types.MsgProofOfWork) (*types.MsgProofOfWorkResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	clientRequestRaw := msg.ClientRequest
	clientRequest, err := clientRequestRaw.ParseData(ctx)
	if err != nil {
		return nil, fmt.Errorf("error on proof of work, can't verify client message: %s", err)
	}
	foundAndActive, _ := k.Keeper.specKeeper.IsSpecIDFoundAndActive(ctx, uint64(clientRequest.Spec_id))
	if !foundAndActive {
		return nil, fmt.Errorf("error on proof of work, spec specified: %d is inactive", clientRequest.Spec_id)
	}
	clientAddr, err := sdk.AccAddressFromBech32(clientRequest.ClientSig)
	if err != nil {
		return nil, fmt.Errorf("error on proof of work, invalid client address: %s", err)
	}
	//TODO: validate CU requested is valid for the user and not too big, this requires the user module
	//TODO: validate the user request only holds supported apis
	possibleServicerAddresses, err := k.Keeper.ValidatePairingForClient(ctx, uint64(clientRequest.Spec_id), clientAddr, *msg.BlockOfWork)
	if err != nil {
		return nil, fmt.Errorf("error on pairing for addresses : %s and %s, err: %s", clientAddr, msg.Creator, err)
	}
	for _, possibleAddr := range possibleServicerAddresses {
		servicerAddr, err := sdk.AccAddressFromBech32(msg.Creator)
		if err != nil {
			return nil, fmt.Errorf("error on proof of work, invalid servicer address: %s", err)
		}
		if possibleAddr.Equals(servicerAddr) {
			//pairing is possible, we can pay servicer for work
			amountToMintForServicerWork := sdk.NewIntFromUint64(uint64(float64(clientRequest.CU_sum) * k.Keeper.GetCoinsPerCU(ctx)))
			amountToMintForServicerWorkCoins := sdk.Coin{Denom: "stake", Amount: amountToMintForServicerWork}
			err := k.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, []sdk.Coin{amountToMintForServicerWorkCoins})
			if err != nil {
				panic(fmt.Sprintf("module failed to mint coins to give to servicer: %s", err))
			}
			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, servicerAddr, []sdk.Coin{amountToMintForServicerWorkCoins})
			if err != nil {
				panic(fmt.Sprintf("failed to transfer minted new coins to servicer, %s account: %s", err, servicerAddr))
			}
			// TODO: add param to burn client, and add code for that
			//k.Keeper.GetBurnClientFactor(ctx)
			amountToBurnClient := sdk.NewIntFromUint64(uint64(float64(clientRequest.CU_sum) * 0))
			//need client Keeper and add a function BurnFromStake
			_ = amountToBurnClient
			break
		}
	}
	return &types.MsgProofOfWorkResponse{}, nil
}
