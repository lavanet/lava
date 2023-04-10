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
	errorLogAndFormat := func(name string, attrs map[string]string, details string) (*types.MsgRelayPaymentResponse, error) {
		return nil, utils.LavaError(ctx, logger, name, attrs, details)
	}

	dataReliabilityStore, err := dataReliabilityByConsumer(msg.VRFs)
	if err != nil {
		return errorLogAndFormat("data_reliability_claim", map[string]string{"error": err.Error()}, "error creating dataReliabilityByConsumer")
	}

	for _, relay := range msg.Relays {
		if relay.LavaChainId != lavaChainID {
			return errorLogAndFormat("relay_payment_lava_chain_id", map[string]string{"relay.LavaChainId": relay.LavaChainId, "expected_ChainID": lavaChainID}, "relay request for the wrong lava chain")
		}
		if relay.Epoch > ctx.BlockHeight() {
			return errorLogAndFormat("relay_future_block", map[string]string{"blockheight": string(relay.Sig)}, "relay request for a block in the future")
		}

		clientAddr, err := sigs.ExtractSignerAddress(relay)
		if err != nil {
			return errorLogAndFormat("relay_payment_sig", map[string]string{"sig": string(relay.Sig)}, "recover PubKey from relay failed")
		}
		providerAddr, err := sdk.AccAddressFromBech32(relay.Provider)
		if err != nil {
			return errorLogAndFormat("relay_payment_addr", map[string]string{"provider": relay.Provider, "creator": msg.Creator}, "invalid provider address in relay msg")
		}
		if !providerAddr.Equals(creator) {
			return errorLogAndFormat("relay_payment_addr", map[string]string{"provider": relay.Provider, "creator": msg.Creator}, "invalid provider address in relay msg, creator and signed provider mismatch")
		}

		// TODO: add support for spec changes
		spec, found := k.specKeeper.GetSpec(ctx, relay.SpecId)
		if !found || !spec.Enabled {
			return errorLogAndFormat("relay_payment_spec", map[string]string{"chainID": relay.SpecId}, "invalid spec ID specified in proof")
		}

		isValidPairing, vrfk, thisProviderIndex, allowedCU, providersToPair, legacy, err := k.Keeper.ValidatePairingForClient(
			ctx,
			relay.SpecId,
			clientAddr,
			providerAddr,
			uint64(relay.Epoch),
		)
		if err != nil {
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String(), "error": err.Error()}
			return errorLogAndFormat("relay_payment_pairing", details, "invalid pairing on proof of relay")
		}
		if !isValidPairing {
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String(), "error": "pairing result doesn't include provider"}
			return errorLogAndFormat("relay_payment_pairing", details, "invalid pairing claim on proof of relay")
		}

		epochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, uint64(relay.Epoch))
		if err != nil {
			details := map[string]string{"epoch": strconv.FormatUint(epochStart, 10), "block": strconv.FormatUint(uint64(relay.Epoch), 10), "error": err.Error()}
			return errorLogAndFormat("relay_payment_epoch_start", details, "problem getting epoch start")
		}

		payReliability := false
		// validate data reliability
		vrfStoreKey := VRFKey{ChainID: relay.SpecId, Epoch: epochStart, Consumer: clientAddr.String()}
		if vrfData, ok := dataReliabilityStore[vrfStoreKey]; ok {
			delete(dataReliabilityStore, vrfStoreKey)
			details := map[string]string{"client": clientAddr.String(), "provider": providerAddr.String()}
			if !spec.DataReliabilityEnabled {
				details["chainID"] = relay.SpecId
				return errorLogAndFormat("relay_payment_data_reliability_disabled", details, "compares_hashes false for spec and reliability was received")
			}

			// verify user signed this data reliability
			valid, err := sigs.ValidateSignerOnVRFData(clientAddr, *vrfData)
			if err != nil || !valid {
				details["error"] = err.Error()
				return errorLogAndFormat("relay_data_reliability_signer", details, "invalid signature by consumer on data reliability message")
			}
			otherProviderAddress, err := sigs.RecoverProviderPubKeyFromVrfDataOnly(vrfData)
			if err != nil {
				return errorLogAndFormat("relay_data_reliability_other_provider", details, "invalid signature by other provider on data reliability message")
			}
			if otherProviderAddress.Equals(providerAddr) {
				// provider signed his own stuff
				details["error"] = "provider attempted to claim data reliability sent by himself"
				return errorLogAndFormat("relay_data_reliability_other_provider", details, "invalid signature by other provider on data reliability message, provider signed his own message")
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
				details["error"] = err.Error()
				return errorLogAndFormat("relay_data_reliability_other_provider_pairing", details, "invalid signature by other provider on data reliability message, provider pairing error")
			}
			if !isValidPairing {
				details["error"] = "pairing isn't valid"
				return errorLogAndFormat("relay_data_reliability_other_provider_pairing", details, "invalid signature by other provider on data reliability message, provider pairing mismatch")
			}
			vrfPk := &utils.VrfPubKey{}
			vrfPk, err = vrfPk.DecodeFromBech32(vrfk)
			if err != nil {
				details["error"] = err.Error()
				details["vrf_bech32"] = vrfk
				return errorLogAndFormat("relay_data_reliability_client_vrf_pk", details, "invalid parsing of vrf pk form bech32")
			}
			// signatures valid, validate VRF signing
			valid = utils.VerifyVrfProofFromVRFData(vrfData, *vrfPk, epochStart)
			if !valid {
				details["error"] = "vrf signing is invalid, proof result mismatch"
				return errorLogAndFormat("relay_data_reliability_vrf_proof", details, "invalid vrf proof by consumer, result doesn't correspond to proof")
			}

			index, vrfErr := utils.GetIndexForVrf(vrfData.VrfValue, uint32(providersToPair), spec.ReliabilityThreshold)
			if vrfErr != nil {
				details["error"] = vrfErr.Error()
				details["VRF_index"] = strconv.FormatInt(index, 10)
				return errorLogAndFormat("relay_payment_reliability_vrf_data", details, details["error"])
			}
			if index != int64(thisProviderIndex) {
				details["error"] = "data reliability returned mismatch index"
				details["VRF_index"] = strconv.FormatInt(index, 10)
				details["thisProviderIndex"] = strconv.FormatInt(int64(thisProviderIndex), 10)
				return errorLogAndFormat("relay_payment_reliability_vrf_data", details, details["error"])
			}
			// all checks passed
			payReliability = true
		}

		// this prevents double spend attacks, and tracks the CU per session a client can use
		totalCUInEpochForUserProvider, err := k.Keeper.AddEpochPayment(ctx, relay.SpecId, epochStart, clientAddr, providerAddr, relay.CuSum, strconv.FormatUint(relay.SessionId, 16))
		if err != nil {
			// double spending on user detected!
			details := map[string]string{
				"epoch":     strconv.FormatUint(epochStart, 10),
				"client":    clientAddr.String(),
				"provider":  providerAddr.String(),
				"error":     err.Error(),
				"unique_ID": strconv.FormatUint(relay.SessionId, 16),
			}
			return errorLogAndFormat("relay_payment_claim", details, "double spending detected")
		}

		err = k.Keeper.EnforceClientCUsUsageInEpoch(ctx, relay.CuSum, allowedCU, totalCUInEpochForUserProvider, epochStart)
		if err != nil {
			// TODO: maybe give provider money but burn user, colluding?
			// TODO: display correct totalCU and usedCU for provider
			details := map[string]string{
				"epoch":                         strconv.FormatUint(epochStart, 10),
				"client":                        clientAddr.String(),
				"provider":                      providerAddr.String(),
				"error":                         err.Error(),
				"CU":                            strconv.FormatUint(relay.CuSum, 10),
				"cuToPay":                       strconv.FormatUint(relay.CuSum, 10),
				"totalCUInEpochForUserProvider": strconv.FormatUint(totalCUInEpochForUserProvider, 10),
			}
			return errorLogAndFormat("relay_payment_user_limit", details, "user bypassed CU limit")
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
				details["error"] = err.Error()
				return errorLogAndFormat("relay_payment_QoS", details, "bad QoSReport")
			}
			details["QoSReport"] = "Latency: " + relay.QosReport.Latency.String() + ", Availability: " + relay.QosReport.Availability.String() + ", Sync: " + relay.QosReport.Sync.String()
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
				details["amountToBurn"] = burnAmount.String()
				details["error"] = err2.Error()
				return errorLogAndFormat("relay_payment_burn", details, "BurnUserStake failed on user")
			}
			if !burnSucceeded {
				details["amountToBurn"] = burnAmount.String()
				details["error"] = "insufficient funds or didn't find user"
				return errorLogAndFormat("relay_payment_burn", details, "BurnUserStake failed on user, did not find user, or insufficient funds")
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
				details["error"] = err.Error()
				utils.LavaError(ctx, logger, "relay_payment", details, "MintCoins Failed,")
				panic(fmt.Sprintf("module failed to mint coins to give to provider: %s", err))
			}
			//
			// Send to provider
			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, rewardCoins)
			if err != nil {
				details["error"] = err.Error()
				utils.LavaError(ctx, logger, types.RelayPaymentEventName, details, "SendCoinsFromModuleToAccount Failed,")
				panic(fmt.Sprintf("failed to transfer minted new coins to provider, %s account: %s", err, providerAddr))
			}
		}

		details["relayNumber"] = strconv.FormatUint(relay.RelayNum, 10)
		utils.LogLavaEvent(ctx, logger, types.RelayPaymentEventName, details, "New Proof Of Work Was Accepted")

		// if this returns an error it means this is legacy consumer
		if !legacy {
			project, _, err := k.projectsKeeper.GetProjectForDeveloper(ctx, clientAddr.String(), uint64(relay.Epoch))
			if err != nil {
				details["error"] = err.Error()
				return errorLogAndFormat("relay_payment_failed_get_project_for_developer", details, "Failed to get project for client")
			}

			err = k.projectsKeeper.AddComputeUnitsToProject(ctx, project, relay.CuSum)
			if err != nil {
				details["error"] = err.Error()
				return errorLogAndFormat("relay_payment_failed_project_add_cu", details, "Failed to add CU to the project")
			}
		}

		// Get servicersToPair param
		servicersToPair, err := k.ServicersToPairCount(ctx, epochStart)
		if err != nil {
			return nil, utils.LavaError(ctx, k.Logger(ctx), "get_servicers_to_pair", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", epochStart)}, "couldn't get servicers to pair")
		}

		// update provider payment storage with complainer's CU
		err = k.updateProviderPaymentStorageWithComplainerCU(ctx, relay.UnresponsiveProviders, logger, epochStart, relay.SpecId, relay.CuSum, servicersToPair, clientAddr)
		if err != nil {
			utils.LogLavaEvent(ctx, logger, types.UnresponsiveProviderUnstakeFailedEventName, map[string]string{"err:": err.Error()}, "Error Unresponsive Providers could not unstake")
		}
	}
	if len(dataReliabilityStore) > 0 {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "invalid relay payment with unused data reliability proofs", map[string]string{"dataReliabilityProofs": fmt.Sprintf("%+v", dataReliabilityStore)}, "didn't find a usage match for each relay")
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
