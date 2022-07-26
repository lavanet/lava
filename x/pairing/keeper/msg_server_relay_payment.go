package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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
		if relay.BlockHeight > ctx.BlockHeight() {
			return errorLogAndFormat("relay_future_block", map[string]string{"blockheight": string(relay.Sig)}, "relay request for a block in the future")
		}

		pubKey, err := sigs.RecoverPubKeyFromRelay(*relay)
		if err != nil {
			return errorLogAndFormat("relay_payment_sig", map[string]string{"sig": string(relay.Sig)}, "recover PubKey from relay failed")
		}
		clientAddr, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return errorLogAndFormat("relay_payment_user_addr", map[string]string{"user": pubKey.Address().String()}, "invalid user address in relay msg")
		}
		providerAddr, err := sdk.AccAddressFromBech32(relay.Provider)
		if err != nil {
			return errorLogAndFormat("relay_payment_addr", map[string]string{"provider": relay.Provider, "creator": msg.Creator}, "invalid provider address in relay msg")
		}
		if !providerAddr.Equals(creator) {
			return errorLogAndFormat("relay_payment_addr", map[string]string{"provider": relay.Provider, "creator": msg.Creator}, "invalid provider address in relay msg, creator and signed provider mismatch")
		}

		// TODO: add support for spec changes
		ok, _ := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, relay.ChainID)
		if !ok {
			return errorLogAndFormat("relay_payment_spec", map[string]string{"chainID": relay.ChainID}, "invalid spec ID specified in proof")
		}

		isValidPairing, isOverlap, userStake, thisProviderIndex, err := k.Keeper.ValidatePairingForClient(
			ctx,
			relay.ChainID,
			clientAddr,
			providerAddr,
			uint64(relay.BlockHeight),
		)
		if err != nil {
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String(), "error": err.Error()}
			return errorLogAndFormat("relay_payment_pairing", details, "invalid pairing on proof of relay")
		}
		if !isValidPairing {
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String(), "error": "pairing result doesn't include provider"}
			return errorLogAndFormat("relay_payment_pairing", details, "invalid pairing claim on proof of relay")
		}

		payReliability := false
		//validate data reliability
		if relay.DataReliability != nil {
			//verify user signed this data reliability
			valid, err := sigs.ValidateSignerOnVRFData(clientAddr, *relay.DataReliability)
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String()}
			if err != nil || !valid {
				details["error"] = err.Error()
				return errorLogAndFormat("relay_data_reliability_signer", details, "invalid signature by consumer on data reliability message")
			}
			otherProviderAddress, err := sigs.RecoverProviderPubKeyFromVrfDataOnly(relay.DataReliability)
			if err != nil {

				return errorLogAndFormat("relay_data_reliability_other_provider", details, "invalid signature by other provider on data reliability message")
			}
			if otherProviderAddress.Equals(providerAddr) {
				//provider signed his own stuff
				details["error"] = "provider attempted to claim data reliability sent by himself"
				return errorLogAndFormat("relay_data_reliability_other_provider", details, "invalid signature by other provider on data reliability message, provider signed his own message")
			}
			//check this other provider is indeed legitimate
			isValidPairing, _, _, _, err := k.Keeper.ValidatePairingForClient(
				ctx,
				relay.ChainID,
				clientAddr,
				otherProviderAddress,
				uint64(relay.BlockHeight),
			)
			if err != nil {
				details["error"] = err.Error()
				return errorLogAndFormat("relay_data_reliability_other_provider_pairing", details, "invalid signature by other provider on data reliability message, provider pairing error")
			}
			if !isValidPairing {
				details["error"] = "pairing isn't valid"
				return errorLogAndFormat("relay_data_reliability_other_provider_pairing", details, "invalid signature by other provider on data reliability message, provider pairing mismatch")
			}
			vrfPk := &utils.VrfPubKey{}
			vrfPk, err = vrfPk.DecodeFromBech32(userStake.Vrfpk)
			if err != nil {
				details["error"] = err.Error()
				details["vrf_bech32"] = userStake.Vrfpk
				return errorLogAndFormat("relay_data_reliability_client_vrf_pk", details, "invalid parsing of vrf pk form bech32")
			}
			//signatures valid, validate VRF signing
			valid = utils.VerifyVrfProofFromVRFData(relay.DataReliability, *vrfPk)
			if !valid {
				details["error"] = "vrf signing is invalid, proof result mismatch"
				return errorLogAndFormat("relay_data_reliability_vrf_proof", details, "invalid vrf proof by consumer, result doesn't correspond to proof")
			}
			spec, found := k.specKeeper.GetSpec(ctx, relay.ChainID)
			if !found {
				details["chainID"] = relay.ChainID
				errorLogAndFormat("relay_payment_spec", details, "failed to get spec for chain ID")
				panic(fmt.Sprintf("failed to get spec for index: %s", relay.ChainID))
			}
			index := utils.GetIndexForVrf(relay.DataReliability.VrfValue, uint32(k.ServicersToPairCount(ctx)), spec.ReliabilityThreshold)
			if index != int64(thisProviderIndex) {
				details["error"] = "data reliability data did not pass the threshold or returned mismatch index"
				details["VRF_index"] = strconv.FormatInt(index, 10)
				errorLogAndFormat("relay_payment_reliability_vrf_data", details, details["error"])
			}
			//all checks passed
			payReliability = true
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
			return errorLogAndFormat("relay_payment_claim", details, "double spending detected")
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
			return errorLogAndFormat("relay_payment_user_limit", details, "user bypassed CU limit")
		}
		if cuToPay > relay.CuSum {
			panic("cuToPay should never be higher than relay.CuSum")
		}
		//

		//pairing is valid, we can pay provider for work
		reward := k.Keeper.MintCoinsPerCU(ctx).MulInt64(int64(cuToPay))
		if reward.IsZero() {
			continue
		}

		rewardCoins := sdk.Coins{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: reward.TruncateInt()}}

		if len(msg.DescriptionString) > 20 {
			msg.DescriptionString = msg.DescriptionString[:20]
		}
		details := map[string]string{"chainID": fmt.Sprintf(relay.ChainID), "client": clientAddr.String(), "provider": providerAddr.String(), "CU": strconv.FormatUint(cuToPay, 10), "BasePay": rewardCoins.String(), "totalCUInEpoch": strconv.FormatUint(totalCUInEpochForUserProvider, 10), "isOverlap": fmt.Sprintf("%t", isOverlap), "uniqueIdentifier": strconv.FormatUint(relay.SessionId, 10), "descriptionString": msg.DescriptionString}

		if relay.QoSReport != nil {
			QoS, err := relay.QoSReport.ComputeQoS()
			if err != nil {
				details["error"] = err.Error()
				return errorLogAndFormat("relay_payment_QoS", details, "bad QoSReport")
			}
			details["QoSReport"] = "Latency: " + relay.QoSReport.Latency.String() + ", Availability: " + relay.QoSReport.Availability.String() + ", Sync: " + relay.QoSReport.Sync.String()
			details["QoSScore"] = QoS.String()

			reward = reward.Mul(QoS.Mul(k.QoSWeight(ctx)).Add(sdk.OneDec().Sub(k.QoSWeight(ctx)))) // reward*QOSScore*QOSWeight + reward*(1-QOSWeight) = reward*(QOSScore*QOSWeight + (1-QOSWeight))
			rewardCoins = sdk.Coins{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: reward.TruncateInt()}}
		}

		//first check we can burn user before we give money to the provider
		amountToBurnClient := k.Keeper.BurnCoinsPerCU(ctx).MulInt64(int64(cuToPay))

		burnAmount := sdk.Coin{Amount: amountToBurnClient.TruncateInt(), Denom: epochstoragetypes.TokenDenom}
		burnSucceeded, err2 := k.BurnClientStake(ctx, relay.ChainID, clientAddr, burnAmount, false)

		if err2 != nil {
			details["amountToBurn"] = burnAmount.String()
			details["error"] = err2.Error()
			return errorLogAndFormat("relay_payment_burn", details, "BurnUserStake failed on user")
		}
		if !burnSucceeded {
			details["amountToBurn"] = burnAmount.String()
			details["error"] = "insufficient funds or didn't find user"
			return errorLogAndFormat("relay_payment_burn", details, "BurnUserStake failed on user, did not find user, or insufficient funds")
		}

		if payReliability {
			details["reliabilityPay"] = "true"
			rewardAddition := reward.Mul(k.Keeper.DataReliabilityReward(ctx))
			reward = reward.Add(rewardAddition)
			rewardCoins = sdk.Coins{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: reward.TruncateInt()}}
			details["Mint"] = rewardCoins.String()
		} else {
			details["reliabilityPay"] = "false"
			details["Mint"] = details["BasePay"]
		}

		// Mint to module
		if !reward.IsZero() {
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
		}
		details["clientFee"] = burnAmount.String()
		details["relayNumber"] = strconv.FormatUint(relay.RelayNum, 10)
		utils.LogLavaEvent(ctx, logger, "relay_payment", details, "New Proof Of Work Was Accepted")

	}
	return &types.MsgRelayPaymentResponse{}, nil
}
