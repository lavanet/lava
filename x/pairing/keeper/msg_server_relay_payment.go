package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	maxComplaintsPerEpoch                     = 3
	collectPaymentsFromNumberOfPreviousEpochs = 2
	providerPaymentMultiplier                 = 2 // multiplying the amount of payments to protect provider from unstaking
)

func (k msgServer) RelayPayment(goCtx context.Context, msg *types.MsgRelayPayment) (*types.MsgRelayPaymentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	lavaChainID := ctx.BlockHeader().ChainID
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	dataReliabilityStore, err := dataReliabilityByConsumer(msg.VRFs)
	if err != nil {
		return nil, utils.LavaFormatWarning("error creating dataReliabilityByConsumer", err)
	}

	for relayIdx, relay := range msg.Relays {
		if relay.LavaChainId != lavaChainID {
			return nil, utils.LavaFormatWarning("relay request for the wrong lava chain", fmt.Errorf("relay_payment_wrong_lava_chain_id"),
				utils.Attribute{Key: "relay.LavaChainId", Value: relay.LavaChainId},
				utils.Attribute{Key: "expected_ChainID", Value: lavaChainID},
			)
		}
		if relay.Epoch > ctx.BlockHeight() {
			return nil, utils.LavaFormatWarning("invalid block in relay msg", fmt.Errorf("relay request for a block in the future"),
				utils.Attribute{Key: "blockheight", Value: ctx.BlockHeight()},
				utils.Attribute{Key: "relayBlock", Value: relay.Epoch},
			)
		}

		clientAddr, err := sigs.ExtractSignerAddress(relay)
		if err != nil {
			return nil, utils.LavaFormatWarning("recover PubKey from relay failed", err,
				utils.Attribute{Key: "sig", Value: relay.Sig},
			)
		}
		providerAddr, err := sdk.AccAddressFromBech32(relay.Provider)
		if err != nil {
			return nil, utils.LavaFormatWarning("invalid provider address in relay msg", err,
				utils.Attribute{Key: "provider", Value: relay.Provider},
				utils.Attribute{Key: "creator", Value: msg.Creator},
			)
		}
		if !providerAddr.Equals(creator) {
			return nil, utils.LavaFormatWarning("invalid provider address in relay msg", fmt.Errorf("creator and signed provider mismatch"),
				utils.Attribute{Key: "provider", Value: relay.Provider},
				utils.Attribute{Key: "creator", Value: msg.Creator},
			)
		}

		// TODO: add support for spec changes
		spec, found := k.specKeeper.GetSpec(ctx, relay.SpecId)
		if !found || !spec.Enabled {
			return nil, utils.LavaFormatWarning("invalid spec ID in relay msg", fmt.Errorf("spec in proof is not found or disabled"),
				utils.Attribute{Key: "chainID", Value: relay.SpecId},
			)
		}

		isValidPairing, vrfk, thisProviderIndex, allowedCU, providersToPair, legacy, err := k.Keeper.ValidatePairingForClient(
			ctx,
			relay.SpecId,
			clientAddr,
			providerAddr,
			uint64(relay.Epoch),
		)
		if err != nil {
			return nil, utils.LavaFormatWarning("invalid pairing on proof of relay", err,
				utils.Attribute{Key: "client", Value: clientAddr.String()},
				utils.Attribute{Key: "provider", Value: providerAddr.String()},
			)
		}
		if !isValidPairing {
			return nil, utils.LavaFormatWarning("invalid pairing on proof of relay", fmt.Errorf("pairing result doesn't include provider"),
				utils.Attribute{Key: "client", Value: clientAddr.String()},
				utils.Attribute{Key: "provider", Value: providerAddr.String()},
			)
		}

		epochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, uint64(relay.Epoch))
		if err != nil {
			return nil, utils.LavaFormatWarning("problem getting epoch start", err,
				utils.Attribute{Key: "relayEpoch", Value: relay.Epoch},
				utils.Attribute{Key: "epochStart", Value: epochStart},
			)
		}

		payReliability := false
		// validate data reliability
		vrfStoreKey := VRFKey{ChainID: relay.SpecId, Epoch: epochStart, Consumer: clientAddr.String()}
		if vrfData, ok := dataReliabilityStore[vrfStoreKey]; ok {
			delete(dataReliabilityStore, vrfStoreKey)
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String()}
			if !spec.DataReliabilityEnabled {
				details["chainID"] = relay.SpecId
				return nil, utils.LavaFormatWarning("data reliability disabled", fmt.Errorf("compares_hashes false for spec and reliability was received"),
					utils.Attribute{Key: "chainID", Value: relay.SpecId},
				)
			}

			// verify user signed this data reliability
			valid, err := sigs.ValidateSignerOnVRFData(clientAddr, *vrfData)
			if err != nil || !valid {
				return nil, utils.LavaFormatWarning("invalid signature by consumer on data reliability message", err)
			}
			otherProviderAddress, err := sigs.RecoverProviderPubKeyFromVrfDataOnly(vrfData)
			if err != nil {
				return nil, utils.LavaFormatWarning("invalid signature by other provider on data reliability message", err)
			}
			if otherProviderAddress.Equals(providerAddr) {
				// provider signed his own stuff
				return nil, utils.LavaFormatWarning("invalid signature by other provider on data reliability message, provider signed his own message", fmt.Errorf("provider attempted to claim data reliability sent by himself"))
			}
			// check this other provider is indeed legitimate
			isValidPairing, _, _, _, _, _, err := k.Keeper.ValidatePairingForClient(
				ctx,
				relay.SpecId,
				clientAddr,
				otherProviderAddress,
				uint64(relay.Epoch),
			)
			if err != nil {
				return nil, utils.LavaFormatWarning("invalid signature by other provider on data reliability message, provider pairing error", err)
			}
			if !isValidPairing {
				return nil, utils.LavaFormatWarning("invalid signature by other provider on data reliability message, provider pairing mismatch", fmt.Errorf("pairing isn't valid"))
			}
			vrfPk := &utils.VrfPubKey{}
			vrfPk, err = vrfPk.DecodeFromBech32(vrfk)
			if err != nil {
				return nil, utils.LavaFormatWarning("invalid parsing of vrf pk form bech32", err,
					utils.Attribute{Key: "vrf_bech32", Value: vrfk},
				)
			}
			// signatures valid, validate VRF signing
			valid = utils.VerifyVrfProofFromVRFData(vrfData, *vrfPk, epochStart)
			if !valid {
				return nil, utils.LavaFormatWarning("invalid vrf proof by consumer, result doesn't correspond to proof", fmt.Errorf("vrf signing is invalid, proof result mismatch"))
			}

			index, vrfErr := utils.GetIndexForVrf(vrfData.VrfValue, uint32(providersToPair), spec.ReliabilityThreshold)
			if vrfErr != nil {
				return nil, utils.LavaFormatWarning("cannot get index for vrf", vrfErr,
					utils.Attribute{Key: "VRF_index", Value: index},
				)
			}
			if index != int64(thisProviderIndex) {
				return nil, utils.LavaFormatWarning("data reliability returned mismatch index", fmt.Errorf("data reliability returned mismatch index"),
					utils.Attribute{Key: "VRF_index", Value: index},
					utils.Attribute{Key: "thisProviderIndex", Value: thisProviderIndex},
				)
			}
			// all checks passed
			payReliability = true
		}

		// this prevents double spend attacks, and tracks the CU per session a client can use
		totalCUInEpochForUserProvider, err := k.Keeper.AddEpochPayment(ctx, relay.SpecId, epochStart, clientAddr, providerAddr, relay.CuSum, strconv.FormatUint(relay.SessionId, 16))
		if err != nil {
			// double spending on user detected!
			return nil, utils.LavaFormatWarning("double spending detected", err,
				utils.Attribute{Key: "epoch", Value: epochStart},
				utils.Attribute{Key: "client", Value: clientAddr.String()},
				utils.Attribute{Key: "provider", Value: providerAddr.String()},
				utils.Attribute{Key: "unique_ID", Value: relay.SessionId},
			)
		}

		err = k.Keeper.EnforceClientCUsUsageInEpoch(ctx, allowedCU, totalCUInEpochForUserProvider, clientAddr, relay.SpecId, uint64(relay.Epoch))
		if err != nil {
			// TODO: maybe give provider money but burn user, colluding?
			// TODO: display correct totalCU and usedCU for provider
			return nil, utils.LavaFormatWarning("user bypassed CU limit", err,
				utils.Attribute{Key: "epoch", Value: epochStart},
				utils.Attribute{Key: "client", Value: clientAddr.String()},
				utils.Attribute{Key: "provider", Value: providerAddr.String()},
				utils.Attribute{Key: "cuToPay", Value: relay.CuSum},
				utils.Attribute{Key: "totalCUInEpochForUserProvider", Value: totalCUInEpochForUserProvider},
			)
		}

		// pairing is valid, we can pay provider for work
		reward := k.Keeper.MintCoinsPerCU(ctx).MulInt64(int64(relay.CuSum))
		if reward.IsZero() {
			continue
		}

		rewardCoins := sdk.Coins{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: reward.TruncateInt()}}

		if len(msg.DescriptionString) > 20 {
			msg.DescriptionString = msg.DescriptionString[:20]
		}
		details := map[string]string{"chainID": fmt.Sprintf(relay.SpecId), "client": clientAddr.String(), "provider": providerAddr.String(), "CU": strconv.FormatUint(relay.CuSum, 10), "BasePay": rewardCoins.String(), "totalCUInEpoch": strconv.FormatUint(totalCUInEpochForUserProvider, 10), "uniqueIdentifier": strconv.FormatUint(relay.SessionId, 10), "descriptionString": msg.DescriptionString}

		if relay.QosReport != nil {
			QoS, err := relay.QosReport.ComputeQoS()
			if err != nil {
				return nil, utils.LavaFormatWarning("bad QoSReport", err)
			}
			// TODO: QoSReport is deprecated remove after version 0.12.0
			details["QoSReport"] = "Latency: " + relay.QosReport.Latency.String() + ", Availability: " + relay.QosReport.Availability.String() + ", Sync: " + relay.QosReport.Sync.String()
			// allow easier extraction of components
			details["QoSLatency"] = relay.QosReport.Latency.String()
			details["QoSAvailability"] = relay.QosReport.Availability.String()
			details["QoSSync"] = relay.QosReport.Sync.String()
			details["QoSScore"] = QoS.String()

			reward = reward.Mul(QoS.Mul(k.QoSWeight(ctx)).Add(sdk.OneDec().Sub(k.QoSWeight(ctx)))) // reward*QOSScore*QOSWeight + reward*(1-QOSWeight) = reward*(QOSScore*QOSWeight + (1-QOSWeight))
			rewardCoins = sdk.Coins{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: reward.TruncateInt()}}
		}

		// first check we can burn user before we give money to the provider
		amountToBurnClient := k.Keeper.BurnCoinsPerCU(ctx).MulInt64(int64(relay.CuSum))
		if legacy {
			burnAmount := sdk.Coin{Amount: amountToBurnClient.TruncateInt(), Denom: epochstoragetypes.TokenDenom}
			burnSucceeded, err2 := k.BurnClientStake(ctx, relay.SpecId, clientAddr, burnAmount, false)

			if err2 != nil {
				return nil, utils.LavaFormatWarning("BurnUserStake failed on user", err2,
					utils.Attribute{Key: "amountToBurn", Value: burnAmount},
				)
			}
			if !burnSucceeded {
				return nil, utils.LavaFormatWarning("BurnUserStake failed on user, did not find user, or insufficient funds", fmt.Errorf("insufficient funds or didn't find user"),
					utils.Attribute{Key: "amountToBurn", Value: burnAmount},
				)
			}

			details["clientFee"] = burnAmount.String()
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
		if !rewardCoins.AmountOf(epochstoragetypes.TokenDenom).IsZero() {
			err = k.Keeper.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoins)
			if err != nil {
				utils.LavaFormatError("MintCoins Failed", err)
				panic(fmt.Sprintf("module failed to mint coins to give to provider: %s", err))
			}
			//
			// Send to provider
			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, rewardCoins)
			if err != nil {
				utils.LavaFormatError("SendCoinsFromModuleToAccount Failed", err)
				panic(fmt.Sprintf("failed to transfer minted new coins to provider, %s account: %s", err, providerAddr))
			}
		}

		details["relayNumber"] = strconv.FormatUint(relay.RelayNum, 10)
		// differentiate between different relays by providing the index in the keys
		successDetails := appendRelayPaymentDetailsToEvent(details, uint64(relayIdx))
		// calling the same event repeatedly within a transaction just appends the new keys to the event
		utils.LogLavaEvent(ctx, logger, types.RelayPaymentEventName, successDetails, "New Proof Of Work Was Accepted")
		if !legacy {
			err = k.chargeComputeUnitsToProjectAndSubscription(ctx, clientAddr, relay)
			if err != nil {
				return nil, utils.LavaFormatError("Failed chaging CU to project and subscription", err)
			}
		}

		// Get servicersToPair param
		servicersToPair, err := k.ServicersToPairCount(ctx, epochStart)
		if err != nil {
			return nil, utils.LavaFormatError("couldn't get servicers to pair", err,
				utils.Attribute{Key: "epoch", Value: epochStart},
			)
		}

		// update provider payment storage with complainer's CU
		err = k.updateProviderPaymentStorageWithComplainerCU(ctx, relay.UnresponsiveProviders, logger, epochStart, relay.SpecId, relay.CuSum, servicersToPair, clientAddr)
		if err != nil {
			utils.LogLavaEvent(ctx, logger, types.UnresponsiveProviderUnstakeFailedEventName, map[string]string{"err:": err.Error()}, "Error Unresponsive Providers could not unstake")
		}
	}
	if len(dataReliabilityStore) > 0 {
		return nil, utils.LavaFormatWarning("invalid relay payment with unused data reliability proofs", fmt.Errorf("didn't find a usage match for each relay"),
			utils.Attribute{Key: "dataReliabilityProofs", Value: dataReliabilityStore},
		)
	}
	return &types.MsgRelayPaymentResponse{}, nil
}

func (k msgServer) updateProviderPaymentStorageWithComplainerCU(ctx sdk.Context, unresponsiveData []byte, logger log.Logger, epoch uint64, chainID string, cuSum uint64, servicersToPair uint64, clientAddr sdk.AccAddress) error {
	var unresponsiveProviders []string

	// check that unresponsiveData exists
	if len(unresponsiveData) == 0 {
		return nil
	}

	// check that servicersToPair is bigger than 1
	if servicersToPair <= 1 {
		servicersToPair = 2
	}

	// unmarshal the byte array unresponsiveData to get a list of unresponsive providers Bech32 addresses
	err := json.Unmarshal(unresponsiveData, &unresponsiveProviders)
	if err != nil {
		return utils.LavaFormatError("unable to unmarshal unresponsive providers", err, []utils.Attribute{{Key: "UnresponsiveProviders", Value: unresponsiveData}, {Key: "dataLength", Value: len(unresponsiveData)}}...)
	}

	// check there are unresponsive providers
	if len(unresponsiveProviders) == 0 {
		// nothing to do.
		return nil
	}

	// the added complainer CU takes into account the number of providers the client complained on and the number
	complainerCuToAdd := cuSum / (uint64(len(unresponsiveProviders)) * (servicersToPair - 1))

	// iterate over the unresponsive providers list and update their complainers_total_cu
	for _, unresponsiveProvider := range unresponsiveProviders {
		// get provider address
		sdkUnresponsiveProviderAddress, err := sdk.AccAddressFromBech32(unresponsiveProvider)
		if err != nil { // if bad data was given, we cant parse it so we ignote it and continue this protects from spamming wrong information.
			utils.LavaFormatError("unable to sdk.AccAddressFromBech32(unresponsive_provider)", err, utils.Attribute{Key: "unresponsive_provider_address", Value: unresponsiveProvider})
			continue
		}

		// get this epoch's epochPayments object
		epochPayments, found, key := k.GetEpochPaymentsFromBlock(ctx, epoch)
		if !found {
			// the epochPayments object should exist since we already paid. if not found, print an error and continue
			utils.LavaFormatError("did not find epochPayments object", err, utils.Attribute{Key: "epochPaymentsKey", Value: key})
			continue
		}

		// get the providerPaymentStorage object using the providerStorageKey
		providerStorageKey := k.GetProviderPaymentStorageKey(ctx, chainID, epoch, sdkUnresponsiveProviderAddress)
		providerPaymentStorage, found := k.GetProviderPaymentStorage(ctx, providerStorageKey)

		if !found {
			// providerPaymentStorage not found (this provider has no payments in this epoch and also no complaints) -> we need to add one complaint
			emptyProviderPaymentStorageWithComplaint := types.ProviderPaymentStorage{
				Index:                                  providerStorageKey,
				UniquePaymentStorageClientProviderKeys: []string{},
				Epoch:                                  epoch,
				ComplainersTotalCu:                     uint64(0),
			}

			// append the emptyProviderPaymentStorageWithComplaint to the epochPayments object's providerPaymentStorages
			epochPayments.ProviderPaymentStorageKeys = append(epochPayments.GetProviderPaymentStorageKeys(), emptyProviderPaymentStorageWithComplaint.GetIndex())
			k.SetEpochPayments(ctx, epochPayments)

			// assign providerPaymentStorage with the new empty providerPaymentStorage
			providerPaymentStorage = emptyProviderPaymentStorageWithComplaint
		}

		// add complainer's used CU to providerPaymentStorage
		providerPaymentStorage.ComplainersTotalCu += complainerCuToAdd

		// set the final provider payment storage state including the complaints
		k.SetProviderPaymentStorage(ctx, providerPaymentStorage)
	}

	return nil
}

type VRFKey struct {
	Consumer string
	Epoch    uint64
	ChainID  string
}

func dataReliabilityByConsumer(vrfs []*types.VRFData) (dataReliabilityByConsumer map[VRFKey]*types.VRFData, err error) {
	dataReliabilityByConsumer = map[VRFKey]*types.VRFData{}
	if len(vrfs) == 0 {
		return
	}
	for _, vrf := range vrfs {
		signer, err := sigs.GetSignerForVRF(*vrf)
		if err != nil {
			return nil, err
		}
		dataReliabilityByConsumer[VRFKey{
			Consumer: signer.String(),
			Epoch:    uint64(vrf.Epoch),
			ChainID:  vrf.ChainId,
		}] = vrf
	}
	return dataReliabilityByConsumer, nil
}

func (k Keeper) chargeComputeUnitsToProjectAndSubscription(ctx sdk.Context, clientAddr sdk.AccAddress, relay *types.RelaySession) error {
	project, _, err := k.projectsKeeper.GetProjectForDeveloper(ctx, clientAddr.String(), uint64(relay.Epoch))
	if err != nil {
		return fmt.Errorf("failed to get project for client")
	}

	err = k.projectsKeeper.ChargeComputeUnitsToProject(ctx, project, uint64(ctx.BlockHeight()), relay.CuSum)
	if err != nil {
		return fmt.Errorf("failed to add CU to the project")
	}

	err = k.subscriptionKeeper.ChargeComputeUnitsToSubscription(ctx, project.GetSubscription(), relay.CuSum)
	if err != nil {
		return fmt.Errorf("failed to add CU to the subscription")
	}

	return nil
}

func appendRelayPaymentDetailsToEvent(from map[string]string, uniqueIdentifier uint64) (to map[string]string) {
	to = map[string]string{}
	sessionIDStr := strconv.FormatUint(uniqueIdentifier, 10)
	for key, value := range from {
		to[key+"."+sessionIDStr] = value
	}
	return to
}
