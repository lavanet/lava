package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type BadgeData struct {
	Badge       types.Badge
	BadgeSigner sdk.AccAddress
}

func (k msgServer) RelayPayment(goCtx context.Context, msg *types.MsgRelayPayment) (*types.MsgRelayPaymentResponse, error) {
	if len(msg.LatestBlockReports) > len(msg.Relays) {
		return nil, utils.LavaFormatError("RelayPayment_invalid_latest_block_reports", fmt.Errorf("invalid latest block reports"),
			utils.LogAttr("latestBlockReports", msg.LatestBlockReports),
			utils.LogAttr("len(latestBlockReports)", len(msg.LatestBlockReports)),
			utils.LogAttr("relays", msg.Relays),
			utils.LogAttr("len(relays)", len(msg.Relays)),
		)
	}

	if !commontypes.ValidateString(msg.GetDescriptionString(), commontypes.DESCRIPTION_RESTRICTIONS, nil) &&
		len(msg.GetDescriptionString()) != 0 {
		return nil, utils.LavaFormatWarning("RelayPayment_invalid_description", fmt.Errorf("invalid string"),
			utils.LogAttr("reason", msg.GetDescriptionString()),
		)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	lavaChainID := ctx.BlockHeader().ChainID
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	addressEpochBadgeMap := map[string]BadgeData{}
	for _, relay := range msg.Relays {
		if relay.Badge != nil {
			mapKey := types.CreateAddressEpochBadgeMapKey(relay.Badge.Address, relay.Badge.Epoch)
			_, ok := addressEpochBadgeMap[mapKey]
			if !ok {
				badgeSigner, err := sigs.ExtractSignerAddress(*relay.Badge)
				if err != nil {
					utils.LavaFormatError("can't extract badge's signer from badge's project signature", err,
						utils.Attribute{Key: "badgeUserAddress", Value: relay.Badge.Address},
						utils.Attribute{Key: "epoch", Value: relay.Badge.Epoch},
					)
					continue
				}
				badgeData := BadgeData{
					Badge:       *relay.Badge,
					BadgeSigner: badgeSigner,
				}
				addressEpochBadgeMap[mapKey] = badgeData
			}
		}
	}

	var rejectedCu uint64 // aggregated rejected CU (due to badge CU overuse or provider double spending)
	rejected_relays_num := len(msg.Relays)
	for relayIdx, relay := range msg.Relays {
		rejectedCu += relay.CuSum
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

		var newBadgeTimerExpiry uint64 // if the badge is new and need to setup a timer, this will be a non-zero value
		if relay.LavaChainId != lavaChainID {
			utils.LavaFormatWarning("relay request for the wrong lava chain", fmt.Errorf("relay_payment_wrong_lava_chain_id"),
				utils.Attribute{Key: "relay.LavaChainId", Value: relay.LavaChainId},
				utils.Attribute{Key: "expected_ChainID", Value: lavaChainID},
			)
			continue
		}
		if relay.Epoch > ctx.BlockHeight() || relay.Epoch < 0 {
			utils.LavaFormatWarning("invalid block in relay msg", fmt.Errorf("relay request for a block in the future"),
				utils.Attribute{Key: "blockheight", Value: ctx.BlockHeight()},
				utils.Attribute{Key: "relayBlock", Value: relay.Epoch},
			)
			continue
		}

		clientAddr, err := sigs.ExtractSignerAddress(relay)
		if err != nil {
			utils.LavaFormatWarning("recover PubKey from relay failed", err,
				utils.Attribute{Key: "sig", Value: relay.Sig},
			)
			continue
		}

		addressEpochBadgeMapKey := types.CreateAddressEpochBadgeMapKey(clientAddr.String(), uint64(relay.Epoch))
		badgeData, badgeFound := addressEpochBadgeMap[addressEpochBadgeMapKey]
		badgeSig := []byte{}
		// if badge is found in the map, clientAddr will change (assuming the badge is valid) since the badge user is not a valid consumer (the badge signer is)
		if badgeFound {
			newBadgeTimerExpiry, err = k.checkBadge(ctx, badgeData, clientAddr.String(), relay)
			if err != nil {
				utils.LavaFormatWarning("badge check failed", err)
				continue
			}

			// badge is valid & CU enforced -> switch address to badge signer (developer key) and continue with payment
			clientAddr = badgeData.BadgeSigner
			badgeSig = badgeData.Badge.ProjectSig
		}

		project, err := k.GetProjectData(ctx, clientAddr, relay.SpecId, uint64(relay.Epoch))
		if err != nil {
			utils.LavaFormatWarning("invalid project data", err)
			continue
		}

		epochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, uint64(relay.Epoch))
		if err != nil {
			utils.LavaFormatWarning("problem getting epoch start", err,
				utils.Attribute{Key: "relayEpoch", Value: relay.Epoch},
				utils.Attribute{Key: "epochStart", Value: epochStart},
			)
			continue
		}

		if k.IsDoubleSpend(ctx, relay.SpecId, epochStart, project.Index, providerAddr, strconv.FormatUint(relay.SessionId, 16)) {
			utils.LavaFormatWarning("double spending detected", err,
				utils.Attribute{Key: "epoch", Value: epochStart},
				utils.Attribute{Key: "client", Value: clientAddr.String()},
				utils.Attribute{Key: "provider", Value: providerAddr.String()},
				utils.Attribute{Key: "unique_ID", Value: relay.SessionId},
			)
			continue
		}

		// *** up until here we checked non-critical traits of the relay and didn't fail the TX
		// if they failed (one relay should affect all of them). From here on, every check will
		// fail the TX ***

		totalCUInEpochForUserProvider := k.Keeper.AddEpochPayment(ctx, relay.SpecId, epochStart, project.Index, providerAddr, relay.CuSum, strconv.FormatUint(relay.SessionId, 16))

		if badgeFound {
			k.handleBadgeCu(ctx, badgeData, relay.Provider, relay.CuSum, newBadgeTimerExpiry)
		}

		// TODO: add support for spec changes
		spec, found := k.specKeeper.GetSpec(ctx, relay.SpecId)
		if !found || !spec.Enabled {
			return nil, utils.LavaFormatWarning("invalid spec ID in relay msg", fmt.Errorf("spec in proof is not found or disabled"),
				utils.Attribute{Key: "chainID", Value: relay.SpecId},
			)
		}

		isValidPairing, allowedCU, providers, err := k.Keeper.ValidatePairingForClient(
			ctx,
			relay.SpecId,
			providerAddr,
			uint64(relay.Epoch),
			project,
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

		rewardedCU, err := k.Keeper.EnforceClientCUsUsageInEpoch(ctx, relay.CuSum, allowedCU, totalCUInEpochForUserProvider, clientAddr, relay.SpecId, uint64(relay.Epoch))
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
		rewardedCUDec := sdk.NewDecFromInt(sdk.NewIntFromUint64(rewardedCU))

		if len(msg.DescriptionString) > 20 {
			msg.DescriptionString = msg.DescriptionString[:20]
		}
		details := map[string]string{"chainID": fmt.Sprintf(relay.SpecId), "epoch": strconv.FormatInt(relay.Epoch, 10), "client": clientAddr.String(), "provider": providerAddr.String(), "CU": strconv.FormatUint(relay.CuSum, 10), "totalCUInEpoch": strconv.FormatUint(totalCUInEpochForUserProvider, 10), "uniqueIdentifier": strconv.FormatUint(relay.SessionId, 10), "descriptionString": msg.DescriptionString}
		details["rewardedCU"] = strconv.FormatUint(relay.CuSum, 10)

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

			rewardedCUDec = rewardedCUDec.Mul(QoS.Mul(k.QoSWeight(ctx)).Add(sdk.OneDec().Sub(k.QoSWeight(ctx)))) // reward*QOSScore*QOSWeight + reward*(1-QOSWeight) = reward*(QOSScore*QOSWeight + (1-QOSWeight))
		}

		if relay.QosExcellenceReport != nil {
			details["ExcellenceQoSLatency"] = relay.QosExcellenceReport.Latency.String()
			details["ExcellenceQoSAvailability"] = relay.QosExcellenceReport.Availability.String()
			details["ExcellenceQoSSync"] = relay.QosExcellenceReport.Sync.String()
		}

		details["projectID"] = project.Index
		details["badge"] = fmt.Sprint(badgeSig)
		details["clientFee"] = "0"
		details["reliabilityPay"] = "false"
		details["Mint"] = "0ulava"
		details["relayNumber"] = strconv.FormatUint(relay.RelayNum, 10)
		details["rewardedCU"] = strconv.FormatUint(rewardedCU, 10)
		// differentiate between different relays by providing the index in the keys
		successDetails := appendRelayPaymentDetailsToEvent(details, uint64(relayIdx))
		// calling the same event repeatedly within a transaction just appends the new keys to the event
		utils.LogLavaEvent(ctx, logger, types.RelayPaymentEventName, successDetails, "New Proof Of Work Was Accepted")

		cuAfterQos := rewardedCUDec.TruncateInt().Uint64()
		err = k.chargeCuToSubscriptionAndCreditProvider(ctx, clientAddr, relay, cuAfterQos)
		if err != nil {
			return nil, utils.LavaFormatError("Failed charging CU to project and subscription", err)
		}

		// update provider payment storage with complainer's CU
		err = k.updateProviderPaymentStorageWithComplainerCU(ctx, relay.UnresponsiveProviders, logger, epochStart, relay.SpecId, cuAfterQos, providers, project.Index)
		if err != nil {
			var reportedProviders []string
			for _, p := range relay.UnresponsiveProviders {
				reportedProviders = append(reportedProviders, p.String())
			}
			reportedProvidersStr := strings.Join(reportedProviders, ",")
			utils.LavaFormatError("failed to update complainers CU for providers", err,
				utils.Attribute{Key: "reported_providers", Value: reportedProvidersStr},
				utils.Attribute{Key: "epoch", Value: strconv.FormatUint(epochStart, 10)},
				utils.Attribute{Key: "chain_id", Value: relay.SpecId},
				utils.Attribute{Key: "cu", Value: strconv.FormatUint(relay.CuSum, 10)},
				utils.Attribute{Key: "project_index", Value: project.Index},
			)
		}
		rejectedCu -= relay.CuSum
		rejected_relays_num--
	}

	// if all relays failed, fail the TX
	if rejected_relays_num != 0 {
		return nil, utils.LavaFormatWarning("relay payment failed", fmt.Errorf("all relays rejected"),
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "description", Value: msg.DescriptionString},
		)
	}

	// only some of the relays were rejected - event for rejected CU
	var rejected_relays bool
	if rejectedCu != 0 {
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.RejectedCuEventName, map[string]string{"rejected_cu": strconv.FormatUint(rejectedCu, 10)}, "Total rejected CU (not paid) in current RelayPayment TX")
		rejected_relays = true
	}

	latestBlockReports := map[string]string{
		"provider": msg.GetCreator(),
	}
	for _, report := range msg.LatestBlockReports {
		latestBlockReports[report.GetSpecId()] = strconv.FormatUint(report.GetLatestBlock(), 10)
		k.setStakeEntryBlockReport(ctx, creator, report.GetSpecId(), report.GetLatestBlock())
	}
	utils.LogLavaEvent(ctx, logger, types.LatestBlocksReportEventName, latestBlockReports, "New LatestBlocks Report for provider")

	return &types.MsgRelayPaymentResponse{RejectedRelays: rejected_relays}, nil
}

func (k msgServer) setStakeEntryBlockReport(ctx sdk.Context, providerAddr sdk.AccAddress, chainID string, latestBlock uint64) {
	stakeEntry, found, ind := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, providerAddr)
	if found {
		stakeEntry.BlockReport = &epochstoragetypes.BlockReport{
			Epoch:       k.epochStorageKeeper.GetEpochStart(ctx),
			LatestBlock: latestBlock,
		}
		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, stakeEntry, ind)
	}
}

func (k msgServer) updateProviderPaymentStorageWithComplainerCU(ctx sdk.Context, unresponsiveProviders []*types.ReportedProvider, logger log.Logger, epoch uint64, chainID string, cuSum uint64, providersToPair []epochstoragetypes.StakeEntry, projectID string) error {
	// check that unresponsiveData exists
	if len(unresponsiveProviders) == 0 {
		return nil
	}

	// check that servicersToPair is bigger than 1
	if len(providersToPair) <= 1 {
		return nil
	}

	// the added complainer CU takes into account the number of providers the client complained on and the number
	complainerCuToAdd := cuSum / (uint64(len(unresponsiveProviders)) * uint64(len(providersToPair)-1))

	// iterate over the unresponsive providers list and update their complainers_total_cu
	for _, unresponsiveProvider := range unresponsiveProviders {
		// get provider address
		sdkUnresponsiveProviderAddress, err := sdk.AccAddressFromBech32(unresponsiveProvider.GetAddress())
		if err != nil { // if bad data was given, we cant parse it so we ignote it and continue this protects from spamming wrong information.
			utils.LavaFormatError("unable to sdk.AccAddressFromBech32(unresponsive_provider)", err, utils.Attribute{Key: "unresponsive_provider_address", Value: unresponsiveProvider})
			continue
		}

		found := false
		for _, provider := range providersToPair {
			if provider.Address == unresponsiveProvider.Address {
				found = true
				break
			}
		}
		if !found {
			utils.LavaFormatError("reported provider that is not in the pairing list of the client", err, utils.Attribute{Key: "unresponsive_provider_address", Value: unresponsiveProvider})
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
		timestamp := time.Unix(unresponsiveProvider.TimestampS, 0)
		utils.LogLavaEvent(ctx, logger, types.ProviderReportedEventName, map[string]string{"provider": unresponsiveProvider.GetAddress(), "timestamp": timestamp.Format(time.DateTime), "disconnections": strconv.FormatUint(unresponsiveProvider.GetDisconnections(), 10), "errors": strconv.FormatUint(unresponsiveProvider.GetErrors(), 10), "project": projectID, "cu": strconv.FormatUint(complainerCuToAdd, 10), "epoch": strconv.FormatUint(epoch, 10), "total_complaint_this_epoch": strconv.FormatUint(providerPaymentStorage.ComplainersTotalCu, 10)}, "provider got reported by consumer")
		// set the final provider payment storage state including the complaints
		k.SetProviderPaymentStorage(ctx, providerPaymentStorage)
	}

	return nil
}

func (k Keeper) chargeCuToSubscriptionAndCreditProvider(ctx sdk.Context, clientAddr sdk.AccAddress, relay *types.RelaySession, cuAfterQos uint64) error {
	epoch := uint64(relay.Epoch)

	project, err := k.projectsKeeper.GetProjectForDeveloper(ctx, clientAddr.String(), epoch)
	if err != nil {
		return fmt.Errorf("failed to get project for client")
	}

	err = k.projectsKeeper.ChargeComputeUnitsToProject(ctx, project, epoch, relay.CuSum)
	if err != nil {
		return fmt.Errorf("failed to add CU to the project")
	}

	sub, err := k.subscriptionKeeper.ChargeComputeUnitsToSubscription(ctx, project.GetSubscription(), epoch, relay.CuSum)
	if err != nil {
		return fmt.Errorf("failed to add CU to the subscription")
	}

	err = k.subscriptionKeeper.AddTrackedCu(ctx, sub.Consumer, relay.Provider, relay.SpecId, cuAfterQos, sub.Block)
	if err != nil {
		return err
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

func (k Keeper) checkBadge(ctx sdk.Context, badgeData BadgeData, client string, relay *types.RelaySession) (newTimerExpiry uint64, err error) {
	if !badgeData.Badge.IsBadgeValid(client, relay.LavaChainId, uint64(relay.Epoch)) {
		return 0, utils.LavaFormatWarning("badge must match traits in relay request", fmt.Errorf("invalid badge"),
			utils.Attribute{Key: "badgeAddress", Value: badgeData.Badge.Address},
			utils.Attribute{Key: "badgeLavaChainId", Value: badgeData.Badge.LavaChainId},
			utils.Attribute{Key: "badgeEpoch", Value: badgeData.Badge.Epoch},
			utils.Attribute{Key: "relayAddress", Value: client},
			utils.Attribute{Key: "relayLavaChainId", Value: relay.LavaChainId},
			utils.Attribute{Key: "relayEpoch", Value: relay.Epoch},
		)
	}

	badgeUsedCuKey := types.BadgeUsedCuKey(badgeData.Badge.ProjectSig, relay.Provider)
	badgeUsedCuMapEntry, found := k.GetBadgeUsedCu(ctx, badgeUsedCuKey)
	if !found {
		// calculate the expiry timestamp for a new timer that will be created later
		// (the timer with a callback to delete the badgeUsedCuEntry after badge.Epoch+blocksToSave (see keeper.go))
		badgeUsedCuTimerExpiryBlock := k.BadgeUsedCuExpiry(ctx, badgeData.Badge)
		if badgeUsedCuTimerExpiryBlock <= uint64(ctx.BlockHeight()) {
			return 0, utils.LavaFormatWarning("badge rejected", fmt.Errorf("badge used CU entry validity expired"),
				utils.Attribute{Key: "badgeUsedCuTimerExpiryBlock", Value: badgeUsedCuTimerExpiryBlock},
				utils.Attribute{Key: "currentBlock", Value: uint64(ctx.BlockHeight())},
			)
		}
		newTimerExpiry = badgeUsedCuTimerExpiryBlock
		badgeUsedCuMapEntry = types.BadgeUsedCu{
			UsedCu: 0,
		}
	}

	// enforce badge CU overuse
	if relay.CuSum+badgeUsedCuMapEntry.UsedCu > badgeData.Badge.CuAllocation {
		return newTimerExpiry, utils.LavaFormatWarning("badge CU allocation exceeded", fmt.Errorf("could not update badge's used CU"),
			utils.Attribute{Key: "relayCuSum", Value: relay.CuSum},
			utils.Attribute{Key: "badgeCuLeft", Value: badgeData.Badge.CuAllocation - badgeUsedCuMapEntry.UsedCu},
		)
	}

	return newTimerExpiry, nil
}

func (k Keeper) handleBadgeCu(ctx sdk.Context, badgeData BadgeData, provider string, relayCuSum uint64, newTimerExpiry uint64) {
	badgeUsedCuKey := types.BadgeUsedCuKey(badgeData.Badge.ProjectSig, provider)
	badgeUsedCuMapEntry, found := k.GetBadgeUsedCu(ctx, badgeUsedCuKey)
	if newTimerExpiry != 0 && !found {
		// setting a timer with a callback to delete the badgeUsedCuEntry after badge.Epoch+blocksToSave (see keeper.go)
		// timerKey = badgeUsedCuMapKey since all badgeUsedCuMapKey keys are unique - can be used to differentiate the timers
		k.badgeTimerStore.AddTimerByBlockHeight(ctx, newTimerExpiry, badgeUsedCuKey, []byte{})

		badgeUsedCuMapEntry = types.BadgeUsedCu{
			BadgeUsedCuKey: badgeUsedCuKey,
			UsedCu:         0,
		}
	}

	badgeUsedCuMapEntry.UsedCu += relayCuSum
	k.SetBadgeUsedCu(ctx, badgeUsedCuMapEntry)
}
