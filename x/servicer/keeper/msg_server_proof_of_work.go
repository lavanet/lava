package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
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
	errorLogAndFormat := func(name string, attrs map[string]string, details string) (*types.MsgProofOfWorkResponse, error) {
		return nil, utils.LavaError(ctx, logger, name, attrs, details)
	}
	for _, relay := range msg.Relays {

		pubKey, err := sigs.RecoverPubKeyFromRelay(relay)
		if err != nil {
			return errorLogAndFormat("relay_proof_sig", map[string]string{"sig": string(relay.Sig)}, "recover PubKey from relay failed")
		}
		clientAddr, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return errorLogAndFormat("relay_proof_user_addr", map[string]string{"user": pubKey.Address().String()}, "invalid user address in relay msg")
		}
		servicerAddr, err := sdk.AccAddressFromBech32(relay.Servicer)
		if err != nil {
			return errorLogAndFormat("relay_proof_addr", map[string]string{"servicer": relay.Servicer, "creator": msg.Creator}, "invalid servicer address in relay msg")
		}
		if !servicerAddr.Equals(creator) {
			return errorLogAndFormat("relay_proof_addr", map[string]string{"servicer": relay.Servicer, "creator": msg.Creator}, "invalid servicer address in relay msg, creator and signed servicer mismatch")
		}

		//
		// TODO: add support for spec changes
		ok, _ := k.Keeper.specKeeper.IsSpecIDFoundAndActive(ctx, uint64(relay.SpecId))
		if !ok {
			return errorLogAndFormat("relay_proof_spec", map[string]string{"chainID": fmt.Sprintf("%d", relay.SpecId)}, "invalid spec ID specified in proof")
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
			details := map[string]string{"client": clientAddr.String(), "servicer": servicerAddr.String(), "error": err.Error()}
			return errorLogAndFormat("relay_proof_pairing", details, "invalid pairing on proof of relay")
		}

		sessionStart, overlapSessionStart, err := k.GetSessionStartForBlock(ctx, types.BlockNum{Num: uint64(relay.BlockHeight)})
		if err != nil {
			details := map[string]string{"height": strconv.FormatInt(relay.BlockHeight, 10), "error": err.Error()}
			return errorLogAndFormat("relay_proof_session", details, "invalid session start for block")
		}
		if isOverlap {
			sessionStart = overlapSessionStart
		}
		//this prevents double spend attacks, and tracks the CU per session a client can use
		totalCUInSessionForUser, err := k.Keeper.AddSessionPayment(ctx, sessionStart.Num, clientAddr, servicerAddr, relay.CuSum, strconv.FormatUint(relay.SessionId, 16))
		if err != nil {
			//double spending on user detected!
			details := map[string]string{"session": strconv.FormatUint(sessionStart.Num, 10), "client": clientAddr.String(), "servicer": servicerAddr.String(), "error": err.Error(), "unique_ID": strconv.FormatUint(relay.SessionId, 16)}
			return errorLogAndFormat("relay_proof_claim", details, "double spending detected")
		}
		err = k.userKeeper.EnforceUserCUsUsageInSession(ctx, userStake, totalCUInSessionForUser)
		if err != nil {
			//TODO: maybe give servicer money but burn user, colluding?
			details := map[string]string{"session": strconv.FormatUint(sessionStart.Num, 10), "client": clientAddr.String(), "servicer": servicerAddr.String(), "error": err.Error(), "CU": strconv.FormatUint(relay.CuSum, 10), "totalCUInSession": strconv.FormatUint(totalCUInSessionForUser, 10)}
			return errorLogAndFormat("relay_proof_user_limit", details, "user bypassed CU limit")
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

			details := map[string]string{"chainID": fmt.Sprintf("%d", relay.SpecId), "client": clientAddr.String(), "servicer": servicerAddr.String(), "CU": strconv.FormatUint(relay.CuSum, 10), "Mint": rewardCoins.String(), "totalCUInSession": strconv.FormatUint(totalCUInSessionForUser, 10), "isOverlap": fmt.Sprintf("%t", isOverlap)}
			//first check we can burn user before we give money to the servicer
			clientBurn := k.Keeper.userKeeper.GetCoinsPerCU(ctx)
			amountToBurnClient := sdk.NewIntFromUint64(uint64(float64(relay.CuSum) * clientBurn))
			spec, found := k.specKeeper.GetSpec(ctx, uint64(relay.SpecId))
			if !found {
				details["chainID"] = strconv.FormatUint(uint64(relay.SpecId), 10)
				errorLogAndFormat("relay_proof_spec", details, "failed to get spec for chain ID")
				panic(fmt.Sprintf("failed to get spec for index: %d", relay.SpecId))
			}
			burnAmount := sdk.Coin{Amount: amountToBurnClient, Denom: "stake"}
			burnSucceeded, err2 := k.userKeeper.BurnUserStake(ctx, usertypes.SpecName{Name: spec.Name}, clientAddr, burnAmount, false)
			if err2 != nil {
				details["amountToBurn"] = burnAmount.String()
				details["error"] = err2.Error()
				return errorLogAndFormat("relay_proof_burn", details, "BurnUserStake failed on user")
			}
			if !burnSucceeded {
				details["amountToBurn"] = burnAmount.String()
				details["error"] = "insufficient funds or didn't find user"
				return errorLogAndFormat("relay_proof_burn", details, "BurnUserStake failed on user, did not find user, or insufficient funds")
			}

			//
			// Mint to module
			err := k.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoins)
			if err != nil {
				details["error"] = err.Error()
				utils.LavaError(ctx, logger, "relay_payment", details, "MintCoins Failed,")
				panic(fmt.Sprintf("module failed to mint coins to give to servicer: %s", err))
			}
			//
			// Send to servicer
			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, servicerAddr, rewardCoins)
			if err != nil {
				details["error"] = err.Error()
				utils.LavaError(ctx, logger, "relay_payment", details, "SendCoinsFromModuleToAccount Failed,")
				panic(fmt.Sprintf("failed to transfer minted new coins to servicer, %s account: %s", err, servicerAddr))
			}
			details["clientFee"] = burnAmount.String()
			utils.LogLavaEvent(ctx, logger, "relay_payment", details, "New Proof Of Work Was Accepted")
		} else {
			details := map[string]string{"client": clientAddr.String(), "servicer": servicerAddr.String(), "error": "pairing result doesn't include servicer"}
			return errorLogAndFormat("relay_proof_pairing", details, "invalid pairing claim on proof of relay")
		}
	}

	return &types.MsgProofOfWorkResponse{}, nil
}
