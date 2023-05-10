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

	for relayIdx, relay := range msg.Relays {
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

		isValidPairing, allowedCU, servicersToPair, legacy, err := k.Keeper.ValidatePairingForClient(
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

		err = k.Keeper.EnforceClientCUsUsageInEpoch(ctx, allowedCU, totalCUInEpochForUserProvider, clientAddr, relay.SpecId, uint64(relay.Epoch))
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

		details["reliabilityPay"] = "false"
		details["Mint"] = details["BasePay"]

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
		// differentiate between different relays by providing the index in the keys
		successDetails := appendRelayPaymentDetailsToEvent(details, uint64(relayIdx))
		// calling the same event repeatedly within a transaction just appends the new keys to the event
		utils.LogLavaEvent(ctx, logger, types.RelayPaymentEventName, successDetails, "New Proof Of Work Was Accepted")
		if !legacy {
			err = k.chargeComputeUnitsToProjectAndSubscription(ctx, clientAddr, relay)
			if err != nil {
				details["error"] = err.Error()
				return errorLogAndFormat("relay_payment_failed", details, "")
			}
		}

		// update provider payment storage with complainer's CU
		err = k.updateProviderPaymentStorageWithComplainerCU(ctx, relay.UnresponsiveProviders, logger, epochStart, relay.SpecId, relay.CuSum, servicersToPair, clientAddr)
		if err != nil {
			utils.LogLavaEvent(ctx, logger, types.UnresponsiveProviderUnstakeFailedEventName, map[string]string{"err:": err.Error()}, "Error Unresponsive Providers could not unstake")
		}
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

func (k Keeper) chargeComputeUnitsToProjectAndSubscription(ctx sdk.Context, clientAddr sdk.AccAddress, relay *types.RelaySession) error {
	project, err := k.projectsKeeper.GetProjectForDeveloper(ctx, clientAddr.String(), uint64(relay.Epoch))
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
