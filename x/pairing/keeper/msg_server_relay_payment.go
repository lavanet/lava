package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) RelayPayment(goCtx context.Context, msg *types.MsgRelayPayment) (*types.MsgRelayPaymentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)

	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	errorLogAndFormat := func(name string, attrs map[string]string, details string) (*types.MsgRelayPaymentResponse, error) {
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
		providerAddr, err := sdk.AccAddressFromBech32(relay.Provider)
		if err != nil {
			return errorLogAndFormat("relay_proof_addr", map[string]string{"provider": relay.Provider, "creator": msg.Creator}, "invalid provider address in relay msg")
		}
		if !providerAddr.Equals(creator) {
			return errorLogAndFormat("relay_proof_addr", map[string]string{"provider": relay.Provider, "creator": msg.Creator}, "invalid provider address in relay msg, creator and signed provider mismatch")
		}

		// TODO: add support for spec changes
		ok, _ := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, relay.ChainID)
		if !ok {
			return errorLogAndFormat("relay_proof_spec", map[string]string{"chainID": relay.ChainID}, "invalid spec ID specified in proof")
		}

		isValidPairing, isOverlap, userStake, err := k.Keeper.ValidatePairingForClient(
			ctx,
			relay.ChainID,
			clientAddr,
			providerAddr,
			uint64(relay.BlockHeight),
		)
		if err != nil {
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String(), "error": err.Error()}
			return errorLogAndFormat("relay_proof_pairing", details, "invalid pairing on proof of relay")
		}
		if !isValidPairing {
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String(), "error": "pairing result doesn't include provider"}
			return errorLogAndFormat("relay_proof_pairing", details, "invalid pairing claim on proof of relay")
		}
		epochStart, _ := k.epochStorageKeeper.GetEpochStartForBlock(ctx, uint64(relay.BlockHeight))
		if isOverlap {
			epochStart = k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, uint64(relay.BlockHeight))
		}
		//this prevents double spend attacks, and tracks the CU per session a client can use
		totalCUInEpochForUserProvider, err := k.Keeper.AddEpochPayment(ctx, relay.ChainID, epochStart, clientAddr, providerAddr, relay.CuSum, strconv.FormatUint(relay.SessionId, 16))
		if err != nil {
			//double spending on user detected!
			details := map[string]string{"session": strconv.FormatUint(epochStart, 10), "client": clientAddr.String(), "provider": providerAddr.String(),
				"error": err.Error(), "unique_ID": strconv.FormatUint(relay.SessionId, 16)}
			return errorLogAndFormat("relay_proof_claim", details, "double spending detected")
		}
		allowedCU, err := k.GetAllowedCU(ctx, userStake)
		if err != nil {
			panic(fmt.Sprintf("user %s, allowedCU was not found for stake of: %d", clientAddr, userStake.Stake.Amount.Int64()))
		}
		cuToPay, err := k.Keeper.EnforceClientCUsUsageInEpoch(ctx, relay.ChainID, relay.CuSum, relay.BlockHeight, allowedCU, clientAddr, totalCUInEpochForUserProvider, providerAddr, epochStart)
		if err != nil {
			//TODO: maybe give provider money but burn user, colluding?
			//TODO: display correct totalCU and usedCU for provider
			details := map[string]string{
				"session":                       strconv.FormatUint(epochStart, 10),
				"client":                        clientAddr.String(),
				"provider":                      providerAddr.String(),
				"error":                         err.Error(),
				"CU":                            strconv.FormatUint(relay.CuSum, 10),
				"cuToPay":                       strconv.FormatUint(cuToPay, 10),
				"totalCUInEpochForUserProvider": strconv.FormatUint(totalCUInEpochForUserProvider, 10)}
			return errorLogAndFormat("relay_proof_user_limit", details, "user bypassed CU limit")
		}
		if cuToPay > relay.CuSum {
			panic("cuToPay should never be higher than relay.CuSum")
		}
		//

		//pairing is valid, we can pay provider for work
		reward := k.Keeper.MintCoinsPerCU(ctx).MulInt64(int64(cuToPay))
		rewardCoins := sdk.Coins{sdk.Coin{Denom: "stake", Amount: reward.TruncateInt()}}
		if reward.IsZero() {
			continue
		}

		details := map[string]string{"chainID": fmt.Sprintf(relay.ChainID), "client": clientAddr.String(), "provider": providerAddr.String(), "CU": strconv.FormatUint(cuToPay, 10), "Mint": rewardCoins.String(), "totalCUInEpoch": strconv.FormatUint(totalCUInEpochForUserProvider, 10), "isOverlap": fmt.Sprintf("%t", isOverlap)}
		//first check we can burn user before we give money to the provider
		amountToBurnClient := k.Keeper.BurnCoinsPerCU(ctx).MulInt64(int64(cuToPay))
		spec, found := k.specKeeper.GetSpec(ctx, relay.ChainID)
		if !found {
			details["chainID"] = relay.ChainID
			errorLogAndFormat("relay_proof_spec", details, "failed to get spec for chain ID")
			panic(fmt.Sprintf("failed to get spec for index: %s", relay.ChainID))
		}
		burnAmount := sdk.Coin{Amount: amountToBurnClient.TruncateInt(), Denom: "stake"}
		burnSucceeded, err2 := k.BurnClientStake(ctx, spec.Index, clientAddr, burnAmount, false)
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
		err = k.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoins)
		if err != nil {
			details["error"] = err.Error()
			utils.LavaError(ctx, logger, "relay_payment", details, "MintCoins Failed,")
			panic(fmt.Sprintf("module failed to mint coins to give to provider: %s", err))
		}
		//
		// Send to provider
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, rewardCoins)
		if err != nil {
			details["error"] = err.Error()
			utils.LavaError(ctx, logger, "relay_payment", details, "SendCoinsFromModuleToAccount Failed,")
			panic(fmt.Sprintf("failed to transfer minted new coins to provider, %s account: %s", err, providerAddr))
		}
		details["clientFee"] = burnAmount.String()
		utils.LogLavaEvent(ctx, logger, "relay_payment", details, "New Proof Of Work Was Accepted")

	}
	return &types.MsgRelayPaymentResponse{}, nil
}
