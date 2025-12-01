package lavasession

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"github.com/lavanet/lava/v5/protocol/common"
	metrics "github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/protocol/qos"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	BlockedProviderSessionUsedStatus   = uint32(1)
	BlockedProviderSessionUnusedStatus = uint32(0)
)

var (
	retrySecondChanceAfter         = time.Minute * 3
	DebugProbes                    = false
	PeriodicProbeProviders         = false
	PeriodicProbeProvidersInterval = 5 * time.Second
)

// created with NewConsumerSessionManager
type ConsumerSessionManager struct {
	rpcEndpoint    *RPCEndpoint // used to filter out endpoints
	lock           sync.RWMutex
	pairing        map[string]*ConsumerSessionsWithProvider // key == provider address
	rawPairing     map[uint64]*ConsumerSessionsWithProvider // key == provider index in pairing. Used for periodic probing of providers
	stickySessions *StickySessionStore
	currentEpoch   uint64
	numberOfResets uint64

	// original pairingAddresses for current epoch
	// contains all addresses from the initial pairing. and the keys are the indexes of the pairing query (these indexes are used for data reliability)
	pairingAddresses       map[uint64]string
	pairingAddressesLength uint64

	// contains all provider addresses that are currently valid
	validAddresses []string
	// provider addresses that were given a second chance instead of reporting them immediately
	secondChanceGivenToAddresses map[string]struct{}

	// contains a sorted list of blocked addresses, sorted by their cu used this epoch for higher chance of response
	currentlyBlockedProviderAddresses []string

	// History of blocked providers from previous epoch to prevent known-bad providers
	// from getting a clean slate at epoch transitions
	previousEpochBlockedProviders map[string]struct{}

	// backup providers - emergency fallback providers when no regular providers are available
	backupProviders map[string]*ConsumerSessionsWithProvider // key == provider address

	addonAddresses    map[string][]string // key is RouterKey.String()
	reportedProviders *ReportedProviders
	// pairingPurge - contains all pairings that are unwanted this epoch, keeps them in memory in order to avoid release.
	// (if a consumer session still uses one of them or we want to report it.)
	pairingPurge                       map[string]*ConsumerSessionsWithProvider
	providerOptimizer                  ProviderOptimizer
	consumerMetricsManager             *metrics.ConsumerMetricsManager
	consumerPublicAddress              string
	activeSubscriptionProvidersStorage *ActiveSubscriptionProvidersStorage

	qosManager *qos.QoSManager

	// getLavaBlockHeight returns the current Lava blockchain block height
	// This is NOT used for RelaySession.Epoch (which must be the pairing epoch start block)
	getLavaBlockHeight func() int64
}

func (csm *ConsumerSessionManager) GetQoSManager() *qos.QoSManager {
	return csm.qosManager
}

func (csm *ConsumerSessionManager) GetNumberOfValidProviders() int {
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	return len(csm.validAddresses)
}

// this is being read in multiple locations and but never changes so no need to lock.
func (csm *ConsumerSessionManager) RPCEndpoint() RPCEndpoint {
	return *csm.rpcEndpoint
}

func (csm *ConsumerSessionManager) UpdateAllProviders(epoch uint64, pairingList map[uint64]*ConsumerSessionsWithProvider, backupProviderList map[uint64]*ConsumerSessionsWithProvider) error {
	utils.LavaFormatDebug("UpdateAllProviders", utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "pairingListLen", Value: len(pairingList)})
	pairingListLength := len(pairingList)
	// TODO: we can block updating until some of the probing is done, this can prevent failed attempts on epoch change when we have no information on the providers,
	// and all of them are new (less effective on big pairing lists or a process that runs for a few epochs)

	defer func() {
		// run this after done updating pairing
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond) // sleep up to 500ms in order to scatter different chains probe triggers
		go func() {
			ctx := context.Background()
			// Probe all providers to eliminate offline ones from affecting relays
			csm.probeProviders(ctx, pairingList, epoch) // pairingList is thread safe it's members are not (accessed through csm.pairing)

			// Check re-blocked providers from previous epoch and unblock if healthy
			csm.checkAndUnblockHealthyReBlockedProviders(ctx, epoch)
		}()
	}()
	previousEpoch := csm.atomicReadCurrentEpoch()
	// clean qos manager purged epochs.
	csm.qosManager.CleanPurgedEpochs(previousEpoch)

	csm.lock.Lock()         // start by locking the class lock.
	defer csm.lock.Unlock() // we defer here so in case we return an error it will unlock automatically.

	csm.rawPairing = pairingList

	if epoch < previousEpoch { // sentry shouldn't update an old epoch
		return utils.LavaFormatError("trying to update provider list for older epoch", nil, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "currentEpoch", Value: csm.atomicReadCurrentEpoch()})
	}

	// For same-epoch updates, we still need to proceed with the update
	// because each ConsumerSessionManager has its own state (validAddresses, reportedProviders, etc.)
	// that needs to be reset. We just skip the epoch write to avoid redundant atomic operations.
	skipEpochWrite := (epoch == previousEpoch)
	if skipEpochWrite {
		utils.LavaFormatDebug("UpdateAllProviders called with same epoch, updating state anyway",
			utils.Attribute{Key: "epoch", Value: epoch},
			utils.Attribute{Key: "spec", Value: csm.rpcEndpoint.Key()})
	}

	// Update Epoch (only if it's different)
	if !skipEpochWrite {
		csm.atomicWriteCurrentEpoch(epoch)
	}

	// Reset States - MUST run even for same-epoch updates because each CSM has its own state
	// csm.validAddresses length is reset in setValidAddressesToDefaultValue
	csm.pairingAddresses = make(map[uint64]string, pairingListLength)

	// Save blocking history from previous epoch before clearing
	// This prevents known-bad providers from getting a clean slate at epoch transition
	csm.previousEpochBlockedProviders = make(map[string]struct{})
	for _, blockedAddr := range csm.currentlyBlockedProviderAddresses {
		csm.previousEpochBlockedProviders[blockedAddr] = struct{}{}
		utils.LavaFormatDebug("UpdateAllProviders: Preserving blocked provider from previous epoch",
			utils.Attribute{Key: "provider", Value: blockedAddr},
			utils.Attribute{Key: "fromEpoch", Value: previousEpoch},
			utils.Attribute{Key: "toEpoch", Value: epoch},
		)
	}

	csm.secondChanceGivenToAddresses = make(map[string]struct{})

	csm.reportedProviders.Reset()
	csm.pairingAddressesLength = uint64(pairingListLength)
	csm.numberOfResets = 0

	providerAddressToEndpoint := map[string]string{}

	csm.RemoveAddonAddresses("", nil)
	// Reset the pairingPurge.
	// This happens only after an entire epoch. so its impossible to have session connected to the old purged list
	// Save reference to old pairingPurge BEFORE updating, so we can close connections outside the lock
	oldPairingPurge := csm.pairingPurge
	csm.pairingPurge = csm.pairing
	csm.pairing = make(map[string]*ConsumerSessionsWithProvider, pairingListLength)
	for idx, provider := range pairingList {
		csm.pairingAddresses[idx] = provider.PublicLavaAddress
		csm.pairing[provider.PublicLavaAddress] = provider
		providerAddressToEndpoint[provider.PublicLavaAddress] = provider.Endpoints[0].NetworkAddress
	}
	csm.setValidAddressesToDefaultValue("", nil, context.Background()) // the starting point is that valid addresses are equal to pairing addresses.

	// Re-block providers that were blocked in previous epoch and still exist in new pairing
	// This prevents users from hitting known-bad providers at epoch transition
	for blockedAddr := range csm.previousEpochBlockedProviders {
		if _, exists := csm.pairing[blockedAddr]; exists {
			utils.LavaFormatDebug("UpdateAllProviders: Re-blocking provider from previous epoch",
				utils.Attribute{Key: "provider", Value: blockedAddr},
				utils.Attribute{Key: "epoch", Value: epoch},
			)
			// Remove from valid addresses to keep it blocked
			csm.removeAddressFromValidAddresses(blockedAddr)
		}
	}

	// reset session related metrics
	go csm.consumerMetricsManager.ResetSessionRelatedMetrics()
	go csm.providerOptimizer.UpdateWeights(CalcWeightsByStake(pairingList), epoch)
	go csm.consumerMetricsManager.ResetBlockedProvidersMetrics(csm.rpcEndpoint.ChainID, csm.rpcEndpoint.ApiInterface, providerAddressToEndpoint)

	// Store backup providers separately from main pairing list for emergency fallback scenarios
	csm.backupProviders = make(map[string]*ConsumerSessionsWithProvider, len(backupProviderList))
	for _, provider := range backupProviderList {
		csm.backupProviders[provider.PublicLavaAddress] = provider
	}

	// Clean up expired sticky sessions
	csm.stickySessions.DeleteOldSessions(previousEpoch)

	utils.LavaFormatDebug("updated providers", utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "spec", Value: csm.rpcEndpoint.Key()})

	// Close old connections OUTSIDE the lock to prevent blocking other operations
	// This is safe because after an entire epoch, it's impossible to have sessions connected to the old purged list
	go csm.closePurgedUnusedPairingsConnections(oldPairingPurge)

	return nil
}

func (csm *ConsumerSessionManager) Initialized() bool {
	csm.lock.RLock()         // start by locking the class lock.
	defer csm.lock.RUnlock() // we defer here so in case we return an error it will unlock automatically.
	return len(csm.pairingAddresses) != 0
}

func (csm *ConsumerSessionManager) RemoveAddonAddresses(addon string, extensions []string) {
	if addon == "" && len(extensions) == 0 {
		// purge all
		csm.addonAddresses = make(map[string][]string)
	} else {
		routerKey := NewRouterKey(append(extensions, addon))
		if csm.addonAddresses == nil {
			csm.addonAddresses = make(map[string][]string)
		}
		csm.addonAddresses[routerKey.String()] = []string{}
	}
}

// csm is Rlocked
func (csm *ConsumerSessionManager) CalculateAddonValidAddresses(addon string, extensions []string, ctx context.Context) (supportingProviderAddresses []string) {
	utils.LavaFormatInfo("🔎 CALCULATING VALID ADDRESSES", utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("totalValidAddresses", len(csm.validAddresses)), utils.LogAttr("currentlyBlockedCount", len(csm.currentlyBlockedProviderAddresses)), utils.LogAttr("GUID", ctx))
	for _, providerAdress := range csm.validAddresses {
		providerEntry := csm.pairing[providerAdress]
		supportsAddon := providerEntry.IsSupportingAddon(addon)
		supportsExtensions := providerEntry.IsSupportingExtensions(extensions, ctx)
		utils.LavaFormatTrace("[Archive Debug] Provider extension check",
			utils.LogAttr("providerAddress", providerAdress),
			utils.LogAttr("supportsAddon", supportsAddon),
			utils.LogAttr("supportsExtensions", supportsExtensions),
			utils.LogAttr("GUID", ctx))
		if supportsAddon && supportsExtensions {
			supportingProviderAddresses = append(supportingProviderAddresses, providerAdress)
			utils.LavaFormatTrace("[Archive Debug] Provider added to supporting list",
				utils.LogAttr("providerAddress", providerAdress),
				utils.LogAttr("GUID", ctx))
		} else {
			utils.LavaFormatTrace("[Archive Debug] Provider filtered out",
				utils.LogAttr("providerAddress", providerAdress),
				utils.LogAttr("reason", "does not support addon or extensions"),
				utils.LogAttr("GUID", ctx))
		}
	}
	utils.LavaFormatInfo("✅ CALCULATION RESULT", utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("supportingProviderCount", len(supportingProviderAddresses)), utils.LogAttr("supportingProviders", supportingProviderAddresses), utils.LogAttr("GUID", ctx))
	return supportingProviderAddresses
}

// assuming csm is Rlocked
func (csm *ConsumerSessionManager) getValidAddresses(addon string, extensions []string, ctx context.Context) (addresses []string) {
	utils.LavaFormatTrace("[Archive Debug] getValidAddresses called",
		utils.LogAttr("addon", addon),
		utils.LogAttr("extensions", extensions),
		utils.LogAttr("GUID", ctx))
	routerKey := NewRouterKey(append(extensions, addon))
	routerKeyString := routerKey.String()
	if csm.addonAddresses == nil || csm.addonAddresses[routerKeyString] == nil {
		utils.LavaFormatTrace("[Archive Debug] Calling CalculateAddonValidAddresses",
			utils.LogAttr("addon", addon),
			utils.LogAttr("extensions", extensions),
			utils.LogAttr("GUID", ctx))
		return csm.CalculateAddonValidAddresses(addon, extensions, ctx)
	}
	utils.LavaFormatTrace("[Archive Debug] Using cached addonAddresses",
		utils.LogAttr("routerKeyString", routerKeyString),
		utils.LogAttr("cachedAddresses", csm.addonAddresses[routerKeyString]),
		utils.LogAttr("GUID", ctx))
	return csm.addonAddresses[routerKeyString]
}

// After 2 epochs we need to close all open connections.
// otherwise golang garbage collector is not closing network connections and they
// will remain open forever.
// This function is now called asynchronously to avoid blocking UpdateAllProviders while closing connections.
func (csm *ConsumerSessionManager) closePurgedUnusedPairingsConnections(pairingPurge map[string]*ConsumerSessionsWithProvider) {
	if pairingPurge == nil {
		return
	}

	for providerAddr, purgedPairing := range pairingPurge {
		callbackPurge := func() {
			for _, endpoint := range purgedPairing.Endpoints {
				for _, endpointConnection := range endpoint.Connections {
					if endpointConnection.connection != nil {
						utils.LavaFormatTrace("purging connection",
							utils.LogAttr("providerAddr", providerAddr),
							utils.LogAttr("endpoint", endpoint.NetworkAddress),
						)
						endpointConnection.connection.Close()
					}
				}
			}
		}
		// on cases where there is still an active subscription over the epoch handover, we purge the connection when subscription ends.
		if csm.activeSubscriptionProvidersStorage.IsProviderCurrentlyUsed(providerAddr) {
			utils.LavaFormatTrace("skipping purge for provider, as its currently used in a subscription",
				utils.LogAttr("providerAddr", providerAddr),
			)
			csm.activeSubscriptionProvidersStorage.addToPurgeWhenDone(providerAddr, callbackPurge)
			continue
		}
		callbackPurge()
	}
}

func (csm *ConsumerSessionManager) PeriodicProbeProviders(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			if csm.rawPairing != nil {
				csm.probeProviders(ctx, csm.rawPairing, csm.atomicReadCurrentEpoch())
			}
		case <-ctx.Done():
			return
		}
	}
}

func (csm *ConsumerSessionManager) probeProviders(ctx context.Context, pairingList map[uint64]*ConsumerSessionsWithProvider, epoch uint64) error {
	guid := utils.GenerateUniqueIdentifier()
	ctx = utils.AppendUniqueIdentifier(ctx, guid)
	if DebugProbes {
		utils.LavaFormatInfo("providers probe initiated", utils.Attribute{Key: "endpoint", Value: csm.rpcEndpoint}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "epoch", Value: epoch})
	}
	// Create a wait group to synchronize the goroutines
	wg := sync.WaitGroup{}
	wg.Add(len(pairingList)) // increment by this and not by 1 for each go routine because we don;t want a race finishing the go routine before the next invocation
	for _, consumerSessionWithProvider := range pairingList {
		// Start a new goroutine for each provider
		go func(consumerSessionsWithProvider *ConsumerSessionsWithProvider) {
			// Call the probeProvider function and defer the WaitGroup Done call
			defer wg.Done()
			latency, providerAddress, err := csm.probeProvider(ctx, consumerSessionsWithProvider, epoch, false)
			success := err == nil // if failure then regard it in availability
			csm.consumerMetricsManager.SetProviderLiveness(csm.rpcEndpoint.ChainID, providerAddress, consumerSessionWithProvider.Endpoints[0].NetworkAddress, success)
			csm.providerOptimizer.AppendProbeRelayData(providerAddress, latency, success)
		}(consumerSessionWithProvider)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
		// all probes finished in time
		if DebugProbes {
			utils.LavaFormatDebug("providers probe done", utils.Attribute{Key: "endpoint", Value: csm.rpcEndpoint}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "epoch", Value: epoch})
		}
		return nil
	case <-ctx.Done():
		utils.LavaFormatWarning("providers probe ran out of time", nil, utils.Attribute{Key: "endpoint", Value: csm.rpcEndpoint}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "epoch", Value: epoch})
		// ran out of time
		return ctx.Err()
	}
}

// this code needs to be thread safe
func (csm *ConsumerSessionManager) probeProvider(ctx context.Context, consumerSessionsWithProvider *ConsumerSessionsWithProvider, epoch uint64, tryReconnectToDisabledEndpoints bool) (latency time.Duration, providerAddress string, err error) {
	connected, endpoints, providerAddress, err := consumerSessionsWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider(ctx, tryReconnectToDisabledEndpoints, true, "", nil)
	if err != nil || !connected {
		if AllProviderEndpointsDisabledError.Is(err) {
			csm.blockProvider(ctx, providerAddress, true, epoch, MaxConsecutiveConnectionAttempts, 0, false, csm.GenerateReconnectCallback(consumerSessionsWithProvider), []error{err}) // reporting and blocking provider this epoch
		}
		return 0, providerAddress, err
	}

	var endpointInfos []EndpointInfo
	lastError := fmt.Errorf("endpoints list is empty") // this error will happen if we had 0 endpoints
	for _, endpointAndConnection := range endpoints {
		err := func() error {
			connectCtx, cancel := context.WithTimeout(ctx, common.AverageWorldLatency)
			defer cancel()
			guid, found := utils.GetUniqueIdentifier(connectCtx)
			if !found {
				return utils.LavaFormatError("probeProvider failed fetching unique identifier from context when it's set", nil)
			}
			if endpointAndConnection == nil ||
				endpointAndConnection.chosenEndpointConnection == nil ||
				endpointAndConnection.chosenEndpointConnection.Client == nil {
				// returned nil client in endpoint, this should never happen, but checking just in case.
				consumerSessionsWithProvider.Lock.Lock()
				defer consumerSessionsWithProvider.Lock.Unlock()
				return utils.LavaFormatError("returned nil client in endpoint", nil, utils.Attribute{Key: "consumerSessionWithProvider", Value: consumerSessionsWithProvider})
			}
			client := endpointAndConnection.chosenEndpointConnection.Client
			probeReq := &pairingtypes.ProbeRequest{
				Guid:         guid,
				SpecId:       csm.rpcEndpoint.ChainID,
				ApiInterface: csm.rpcEndpoint.ApiInterface,
			}
			var trailer metadata.MD
			relaySentTime := time.Now()
			metadataAdd := metadata.New(map[string]string{common.LAVA_LB_UNIQUE_ID_HEADER: endpointAndConnection.chosenEndpointConnection.GetLbUniqueId()})
			connectCtx = metadata.NewOutgoingContext(connectCtx, metadataAdd)

			probeResp, err := client.Probe(connectCtx, probeReq, grpc.Trailer(&trailer))

			relayLatency := time.Since(relaySentTime)
			versions := trailer.Get(common.VersionMetadataKey)
			if err != nil {
				return utils.LavaFormatError("probe call error", err, utils.Attribute{Key: "provider", Value: providerAddress})
			}
			providerGuid := probeResp.GetGuid()
			if providerGuid != guid {
				return utils.LavaFormatWarning("mismatch probe response", nil, utils.Attribute{Key: "provider", Value: providerAddress}, utils.Attribute{Key: "provider Guid", Value: providerGuid}, utils.Attribute{Key: "sent guid", Value: guid})
			}
			if probeResp.LatestBlock == 0 {
				return utils.LavaFormatWarning("provider returned 0 latest block", nil, utils.Attribute{Key: "provider", Value: providerAddress}, utils.Attribute{Key: "sent guid", Value: guid})
			}

			endpointInfos = append(endpointInfos, EndpointInfo{
				Latency:  relayLatency,
				Endpoint: endpointAndConnection.endpoint,
			})
			// public lava address is a value that is not changing, so it's thread safe
			if DebugProbes {
				utils.LavaFormatDebug("Probed provider successfully", utils.Attribute{Key: "latency", Value: relayLatency}, utils.Attribute{Key: "provider", Value: consumerSessionsWithProvider.PublicLavaAddress}, utils.LogAttr("version", strings.Join(versions, ",")))
			}
			return nil
		}()
		if err != nil {
			lastError = err
		}
	}

	if len(endpointInfos) == 0 {
		// no endpoints.
		return 0, providerAddress, lastError
	}
	sort.Sort(EndpointInfoList(endpointInfos))
	consumerSessionsWithProvider.sortEndpointsByLatency(endpointInfos)
	return endpointInfos[0].Latency, providerAddress, nil
}

// csm needs to be locked here
func (csm *ConsumerSessionManager) setValidAddressesToDefaultValue(addon string, extensions []string, ctx context.Context) {
	utils.LavaFormatInfo("Resetting blocked provider list", utils.LogAttr("blockedProvidersBeforeReset", csm.currentlyBlockedProviderAddresses), utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("GUID", ctx))
	csm.currentlyBlockedProviderAddresses = make([]string, 0) // reset currently blocked provider addresses
	utils.LavaFormatInfo("Blocked provider list reset complete", utils.LogAttr("blockedProvidersAfterReset", csm.currentlyBlockedProviderAddresses), utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("GUID", ctx))
	if addon == "" && len(extensions) == 0 {
		csm.validAddresses = make([]string, len(csm.pairingAddresses))
		index := 0
		for _, provider := range csm.pairingAddresses {
			csm.validAddresses[index] = provider
			index++
		}
	} else {
		// check if one of the pairing addresses supports the addon
	addingToValidAddresses:
		for _, provider := range csm.pairingAddresses {
			supportsAddon := csm.pairing[provider].IsSupportingAddon(addon)
			supportsExtensions := csm.pairing[provider].IsSupportingExtensions(extensions, ctx)

			utils.LavaFormatTrace("[Archive Debug] Provider filtering check",
				utils.LogAttr("providerAddress", provider),
				utils.LogAttr("addon", addon),
				utils.LogAttr("extensions", extensions),
				utils.LogAttr("supportsAddon", supportsAddon),
				utils.LogAttr("supportsExtensions", supportsExtensions),
				utils.LogAttr("GUID", ctx))

			if supportsAddon && supportsExtensions {
				for _, validAddress := range csm.validAddresses {
					if validAddress == provider {
						// it exists, no need to add it again
						continue addingToValidAddresses
					}
				}
				// get here only it found a supporting provider that is not valid
				utils.LavaFormatTrace("[Archive Debug] Adding provider to valid addresses",
					utils.LogAttr("providerAddress", provider),
					utils.LogAttr("GUID", ctx))
				csm.validAddresses = append(csm.validAddresses, provider)
			} else {
				utils.LavaFormatTrace("[Archive Debug] Provider filtered out",
					utils.LogAttr("providerAddress", provider),
					utils.LogAttr("reason", "does not support addon or extensions"),
					utils.LogAttr("GUID", ctx))
			}
		}
		csm.RemoveAddonAddresses(addon, extensions) // refresh the list
		csm.addonAddresses[NewRouterKey(append(extensions, addon)).String()] = csm.CalculateAddonValidAddresses(addon, extensions, ctx)
	}
}

// reads cs.currentEpoch atomically
func (csm *ConsumerSessionManager) atomicWriteCurrentEpoch(epoch uint64) {
	atomic.StoreUint64(&csm.currentEpoch, epoch)
}

// reads cs.currentEpoch atomically
func (csm *ConsumerSessionManager) atomicReadCurrentEpoch() (epoch uint64) {
	return atomic.LoadUint64(&csm.currentEpoch)
}

func (csm *ConsumerSessionManager) atomicReadNumberOfResets() (resets uint64) {
	return atomic.LoadUint64(&csm.numberOfResets)
}

// reset the valid addresses list and increase numberOfResets
func (csm *ConsumerSessionManager) resetValidAddresses(addon string, extensions []string) uint64 {
	csm.lock.Lock() // lock write
	defer csm.lock.Unlock()
	if len(csm.getValidAddresses(addon, extensions, context.Background())) == 0 { // re verify it didn't change while waiting for lock.
		csm.setValidAddressesToDefaultValue(addon, extensions, context.Background())
		// only if length is larger than 0 after reset we actually reset. otherwise we don't have any providers for addon or extension
		if len(csm.getValidAddresses(addon, extensions, context.Background())) != 0 {
			utils.LavaFormatWarning("Provider pairing list is empty, resetting state.", nil, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "extensions", Value: extensions})
		} else {
			utils.LavaFormatWarning("No providers for asked addon or extension, list is empty after trying to reset", nil, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "extensions", Value: extensions})
			if addon == "" && len(extensions) == 0 {
				utils.LavaFormatError("User subscription might have expired or not purchased properly, pairing list is empty after reset.", nil, utils.LogAttr("consumer_address", csm.consumerPublicAddress))
			}
		}
		csm.numberOfResets += 1
	}
	// if len(csm.validAddresses) != 0 meaning we had a reset (or an epoch change), so we need to return the numberOfResets which is currently in csm
	return csm.numberOfResets
}

func (csm *ConsumerSessionManager) cacheAddonAddresses(addon string, extensions []string, ctx context.Context) []string {
	csm.lock.Lock() // lock to set validAddresses[addon] if it's not cached
	defer csm.lock.Unlock()
	routerKey := NewRouterKey(append(extensions, addon))
	routerKeyString := routerKey.String()
	if csm.addonAddresses == nil || csm.addonAddresses[routerKeyString] == nil {
		csm.RemoveAddonAddresses(addon, extensions)
		csm.addonAddresses[routerKeyString] = csm.CalculateAddonValidAddresses(addon, extensions, ctx)
	}
	return csm.addonAddresses[routerKeyString]
}

// validating we still have providers, otherwise reset valid addresses list
// also caches validAddresses for an addon to save on compute
func (csm *ConsumerSessionManager) validatePairingListNotEmpty(addon string, extensions []string, ctx context.Context) uint64 {
	numberOfResets := csm.atomicReadNumberOfResets()
	validAddresses := csm.cacheAddonAddresses(addon, extensions, ctx)
	utils.LavaFormatInfo("🔍 VALIDATING PROVIDERS", utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("validAddressesCount", len(validAddresses)), utils.LogAttr("validAddresses", validAddresses), utils.LogAttr("currentlyBlockedCount", len(csm.currentlyBlockedProviderAddresses)), utils.LogAttr("GUID", ctx))
	if len(validAddresses) == 0 {
		utils.LavaFormatWarning("🚨 NO VALID PROVIDERS - TRIGGERING RESET", nil, utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("GUID", ctx))
		numberOfResets = csm.resetValidAddresses(addon, extensions)
		utils.LavaFormatInfo("🔄 RESET COMPLETED", utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("newValidAddressesCount", len(csm.cacheAddonAddresses(addon, extensions, ctx))), utils.LogAttr("GUID", ctx))
	}
	return numberOfResets
}

func (csm *ConsumerSessionManager) getSessionWithProviderOrError(ctx context.Context, wantedProviderNumber int, usedProviders UsedProvidersInf, tempIgnoredProviders *ignoredProviders, cuNeededForSession uint64, requestedBlock int64, addon string, extensionNames []string, stateful uint32, virtualEpoch uint64, stickiness string) (sessionWithProviderMap SessionWithProviderMap, err error) {
	sessionWithProviderMap, err = csm.getValidConsumerSessionsWithProvider(ctx, wantedProviderNumber, tempIgnoredProviders, cuNeededForSession, requestedBlock, addon, extensionNames, stateful, virtualEpoch, stickiness)
	if err != nil {
		if PairingListEmptyError.Is(err) {
			// Emergency fallback chain: backup providers first, then blocked providers for maximum availability
			if len(csm.backupProviders) > 0 {
				utils.LavaFormatDebug("No regular providers available, trying backup providers", utils.LogAttr("GUID", ctx))
				// try to get a session from the backup providers
				sessionWithProviderMap, err = csm.getValidConsumerSessionsWithProviderFromBackupProviderList(ctx, tempIgnoredProviders, cuNeededForSession, requestedBlock, addon, extensionNames, stateful, virtualEpoch, usedProviders)
				if err == nil {
					// backup providers succeeded, return the session
					utils.LavaFormatDebug("Successfully got session from backup providers", utils.LogAttr("GUID", ctx))
					return sessionWithProviderMap, nil
				}
				// backup providers failed, continue to blocked providers
				utils.LavaFormatDebug("Backup providers failed, trying blocked providers", utils.LogAttr("error", err.Error()), utils.LogAttr("GUID", ctx))
			}

			// try to recover a session from the currently blocked providers
			var errOnRetry error
			sessionWithProviderMap, errOnRetry = csm.tryGetConsumerSessionWithProviderFromBlockedProviderList(ctx, wantedProviderNumber, tempIgnoredProviders, cuNeededForSession, requestedBlock, addon, extensionNames, stateful, virtualEpoch, usedProviders)
			if errOnRetry != nil {
				utils.LavaFormatDebug("All providers failed (regular, backup, and blocked)", utils.LogAttr("GUID", ctx))
				return nil, errOnRetry
			}
			utils.LavaFormatDebug("Successfully got session from blocked providers", utils.LogAttr("GUID", ctx))
		} else {
			return nil, err
		}
		// if we got here we managed to get a sessionWithProviderMap
	}
	return sessionWithProviderMap, nil
}

// GetSessions will return a ConsumerSession, given cu needed for that session.
// The user can also request specific providers to not be included in the search for a session.
func (csm *ConsumerSessionManager) GetSessions(ctx context.Context, wantedProviderNumber int, cuNeededForSession uint64, usedProviders UsedProvidersInf, requestedBlock int64, addon string, extensions []*spectypes.Extension, stateful uint32, virtualEpoch uint64, stickiness string) (
	consumerSessionMap ConsumerSessionsMap, errRet error,
) {
	// set usedProviders if they were chosen for this relay
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	cantSelectError := usedProviders.TryLockSelection(timeoutCtx)
	if cantSelectError != nil {
		if ContextDoneNoNeedToLockSelectionError.Is(cantSelectError) {
			return nil, utils.LavaFormatDebug("Context deadline exceeded when trying to lock selection", utils.LogAttr("GUID", ctx))
		}
		return nil, utils.LavaFormatError("failed getting sessions from used Providers", nil, utils.LogAttr("usedProviders", usedProviders), utils.LogAttr("endpoint", csm.rpcEndpoint), utils.LogAttr("GUID", ctx))
	}
	defer func() { usedProviders.AddUsed(consumerSessionMap, errRet) }()
	routerKey := NewRouterKeyFromExtensions(extensions)
	initUnwantedProviders := usedProviders.GetUnwantedProvidersToSend(routerKey)

	extensionNames := common.GetExtensionNames(extensions)
	utils.LavaFormatTrace("[Archive Debug] GetSessions extension conversion",
		utils.LogAttr("originalExtensions", extensions),
		utils.LogAttr("extensionNames", extensionNames),
		utils.LogAttr("GUID", ctx))
	// if pairing list is empty we reset the state.
	numberOfResets := csm.validatePairingListNotEmpty(addon, extensionNames, ctx)

	// providers that we don't try to connect this iteration.
	tempIgnoredProviders := &ignoredProviders{
		providers:    initUnwantedProviders,
		currentEpoch: csm.atomicReadCurrentEpoch(),
	}
	utils.LavaFormatTrace("GetSessions tempIgnoredProviders", utils.LogAttr("tempIgnoredProviders", tempIgnoredProviders), utils.LogAttr("GUID", ctx))

	// Get a valid consumerSessionsWithProvider
	sessionWithProviderMap, err := csm.getSessionWithProviderOrError(ctx, wantedProviderNumber, usedProviders, tempIgnoredProviders, cuNeededForSession, requestedBlock, addon, extensionNames, stateful, virtualEpoch, stickiness)
	if err != nil {
		utils.LavaFormatTrace("GetSessions error", utils.LogAttr("error", err.Error()), utils.LogAttr("GUID", ctx))
		return nil, err
	}

	// Save how many sessions we are aiming to have
	wantedSession := len(sessionWithProviderMap)
	// Save sessions to return
	sessions := make(ConsumerSessionsMap, wantedSession)
	for {
		for providerAddress, sessionWithProvider := range sessionWithProviderMap {
			// Extract values from session with provider
			consumerSessionsWithProvider := sessionWithProvider.SessionsWithProvider
			sessionEpoch := sessionWithProvider.CurrentEpoch

			// Get a valid Endpoint from the provider chosen
			connected, endpoints, _, err := consumerSessionsWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider(ctx, false, false, addon, extensionNames)
			if err != nil {
				// verify err is AllProviderEndpointsDisabled and report.
				if AllProviderEndpointsDisabledError.Is(err) {
					tempIgnoredProviders.providers[providerAddress] = struct{}{}
					err = csm.blockProvider(ctx, providerAddress, true, sessionEpoch, MaxConsecutiveConnectionAttempts, 0, false, csm.GenerateReconnectCallback(consumerSessionsWithProvider), []error{err}) // reporting and blocking provider this epoch
					if err != nil {
						if !EpochMismatchError.Is(err) {
							// only acceptable error is EpochMismatchError so if different, throw fatal
							utils.LavaFormatFatal("Unsupported Error", err, utils.LogAttr("GUID", ctx))
						}
					}
					continue
				} else {
					utils.LavaFormatFatal("Unsupported Error", err, utils.LogAttr("GUID", ctx))
				}
			} else if !connected {
				// If failed to connect we ignore this provider for this get session request only
				// and try again getting a random provider to pick from
				tempIgnoredProviders.providers[providerAddress] = struct{}{}
				continue
			}

			// get the endpoint we got, as its the only one returned when asking fetchEndpointConnectionFromConsumerSessionWithProvider with false value
			endpoint := endpoints[0]

			// we get the reported providers here after we try to connect, so if any provider didn't respond he will already be added to the list.
			reportedProviders := csm.GetReportedProviders(sessionEpoch)

			// Get session from endpoint or create new or continue. if more than 10 connections are open.
			consumerSession, pairingEpoch, err := consumerSessionsWithProvider.GetConsumerSessionInstanceFromEndpoint(endpoint.chosenEndpointConnection, numberOfResets, csm.qosManager)
			if err != nil {
				utils.LavaFormatError("Error on consumerSessionWithProvider.getConsumerSessionInstanceFromEndpoint", err,
					utils.LogAttr("providerAddress", providerAddress),
					utils.LogAttr("validAddresses", csm.validAddresses),
					utils.LogAttr("Error", err.Error()),
					utils.LogAttr("GUID", ctx),
				)
				if MaximumNumberOfSessionsExceededError.Is(err) {
					// we can get a different provider, adding this provider to the list of providers to skip on.
					tempIgnoredProviders.providers[providerAddress] = struct{}{}
				} else if MaximumNumberOfBlockListedSessionsError.Is(err) {
					// provider has too many block listed sessions. we block it until the next epoch and ignore it so it won't pop up again when resetting the provider list.
					tempIgnoredProviders.providers[providerAddress] = struct{}{}
					err = csm.blockProvider(ctx, providerAddress, false, sessionEpoch, 0, 0, false, nil, []error{})
					if err != nil {
						utils.LavaFormatError("Failed to block provider: ", err, utils.LogAttr("GUID", ctx))
					}
				} else {
					utils.LavaFormatFatal("Unsupported Error", err, utils.LogAttr("GUID", ctx))
				}

				continue
			}

			if pairingEpoch != sessionEpoch {
				// pairingEpoch and SessionEpoch must be the same, we validate them here if they are different we raise an error and continue with pairingEpoch
				utils.LavaFormatError("sessionEpoch and pairingEpoch mismatch", nil, utils.Attribute{Key: "sessionEpoch", Value: sessionEpoch}, utils.Attribute{Key: "pairingEpoch", Value: pairingEpoch})
				sessionEpoch = pairingEpoch
			}

			// If we successfully got a consumerSession we can apply the current CU to the consumerSessionWithProvider.UsedComputeUnits
			err = consumerSessionsWithProvider.addUsedComputeUnits(cuNeededForSession, virtualEpoch)
			if err != nil {
				utils.LavaFormatDebug("consumerSessionWithProvider.addUsedComputeUnit", utils.Attribute{Key: "Error", Value: err.Error()}, utils.LogAttr("GUID", ctx))
				if MaxComputeUnitsExceededError.Is(err) {
					tempIgnoredProviders.providers[providerAddress] = struct{}{}
					// We must unlock the consumer session before continuing.
					consumerSession.Free(nil)
					continue
				} else {
					utils.LavaFormatFatal("Unsupported Error", err, utils.LogAttr("GUID", ctx))
				}
			} else {
				// consumer session is locked and valid, we need to set the relayNumber and the relay cu. before returning.
				// Successfully created/got a consumerSession.

				utils.LavaFormatTrace("Consumer get session",
					utils.LogAttr("provider", providerAddress),
					utils.LogAttr("sessionEpoch", sessionEpoch),
					utils.LogAttr("consumerSession.CUSum", consumerSession.CuSum),
					utils.LogAttr("consumerSession.RelayNum", consumerSession.RelayNum),
					utils.LogAttr("consumerSession.SessionId", consumerSession.SessionId),
					utils.LogAttr("GUID", ctx),
				)

				// If no error, add provider session map
				sessionInfo := &SessionInfo{
					StakeSize:         consumerSessionsWithProvider.getProviderStakeSize(),
					Session:           consumerSession,
					Epoch:             sessionEpoch, // Must use pairing epoch (epoch start block) for provider validation
					ReportedProviders: reportedProviders,
				}

				// adding qos summary for error parsing.
				// consumer session is locked here so its ok to read the qos report.
				sessionInfo.QoSSummaryResult = consumerSession.getQosComputedResultOrZero()
				sessions[providerAddress] = sessionInfo

				qosReport, _ := csm.providerOptimizer.GetReputationReportForProvider(providerAddress)
				if csm.rpcEndpoint.Geolocation != uint64(endpoint.endpoint.Geolocation) && !consumerSessionsWithProvider.StaticProvider {
					// rawQosReport is used only when building the relay payment message to be used to update
					// the provider's reputation on-chain. If the consumer and provider don't share geolocation
					// (consumer geo: csm.rpcEndpoint.Geolocation, provider geo: endpoint.endpoint.Geolocation)
					// we don't want to update the reputation by it, so we null the rawQosReport
					qosReport = nil
				}
				consumerSession.SetUsageForSession(cuNeededForSession, qosReport, usedProviders, routerKey)
				// We successfully added provider, we should ignore it if we need to fetch new
				tempIgnoredProviders.providers[providerAddress] = struct{}{}
				if len(sessions) == wantedSession {
					return sessions, nil
				}
				continue
			}
		}

		// If we do not have enough fetch more
		sessionWithProviderMap, err = csm.getSessionWithProviderOrError(ctx, 1, usedProviders, tempIgnoredProviders, cuNeededForSession, requestedBlock, addon, extensionNames, stateful, virtualEpoch, stickiness)
		// If error exists but we have sessions, return them
		if err != nil && len(sessions) != 0 {
			return sessions, nil
		}
		// If error happens, and we do not have any sessions return an error
		if err != nil {
			return nil, err
		}
		// if we got here we managed to get more sessions so we will try to connect and return a session to the user.
	}
}

// csm must be rlocked here
func (csm *ConsumerSessionManager) getTopTenProvidersForStatefulCalls(validAddresses []string, ignoredProvidersList map[string]struct{}) []string {
	// sort by cu used, easiest to sort by that factor as it probably means highest QOS and easily read by atomic
	customSort := func(i, j int) bool {
		return csm.pairing[validAddresses[i]].atomicReadUsedComputeUnits() > csm.pairing[validAddresses[j]].atomicReadUsedComputeUnits()
	}
	// Sort the slice using the custom sorting rule
	sort.Slice(validAddresses, customSort)
	addresses := []string{}
	wantedLength := 10
	for _, sortedAddress := range validAddresses {
		// skip ignored providers
		if _, foundInIgnoredProviderList := ignoredProvidersList[sortedAddress]; foundInIgnoredProviderList {
			continue
		}
		// fill the slice until we have 10 providers who are not ignored
		addresses = append(addresses, sortedAddress)
		if len(addresses) >= wantedLength {
			break
		}
	}
	return addresses
}

// Get a valid provider address.
func (csm *ConsumerSessionManager) getValidProviderAddresses(ctx context.Context, wantedProviders int, ignoredProvidersList map[string]struct{}, cu uint64, requestedBlock int64, addon string, extensions []string, stateful uint32, stickiness string) (addresses []string, err error) {
	// cs.Lock must be Rlocked here.
	ignoredProvidersListLength := len(ignoredProvidersList)
	validAddresses := csm.getValidAddresses(addon, extensions, ctx)
	validAddressesLength := len(validAddresses)
	totalValidLength := validAddressesLength - ignoredProvidersListLength

	if stickysession, ok := csm.stickySessions.Get(stickiness); ok {
		// Check if sticky session provider is still valid
		providerValid := slices.Contains(validAddresses, stickysession.Provider)
		if providerValid {
			addresses = []string{stickysession.Provider}
			utils.LavaFormatTrace("returning sticky session", utils.LogAttr("provider", stickysession.Provider), utils.LogAttr("id", stickiness), utils.LogAttr("GUID", ctx))
			return addresses, nil
		} else {
			utils.LavaFormatTrace("sticky session provider is no longer valid, deleting", utils.LogAttr("provider", stickysession.Provider), utils.LogAttr("id", stickiness), utils.LogAttr("GUID", ctx))
			csm.stickySessions.Delete(stickiness)
		}
	}

	if totalValidLength <= 0 {
		// check all ignored are actually valid addresses
		ignoredProvidersListLength = 0
		for _, address := range validAddresses {
			if _, ok := ignoredProvidersList[address]; ok {
				ignoredProvidersListLength++
			}
		}
		if validAddressesLength-ignoredProvidersListLength <= 0 {
			utils.LavaFormatDebug("Pairing list empty", utils.Attribute{Key: "Provider list", Value: validAddresses}, utils.Attribute{Key: "IgnoredProviderList", Value: ignoredProvidersList}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "extensions", Value: extensions}, utils.LogAttr("GUID", ctx))
			err = PairingListEmptyError
			return addresses, err
		}
	}
	var providers []string
	if stateful == common.CONSISTENCY_SELECT_ALL_PROVIDERS && csm.providerOptimizer.Strategy() != provideroptimizer.StrategyCost {
		providers = csm.getTopTenProvidersForStatefulCalls(validAddresses, ignoredProvidersList)
	} else if stickiness != "" {
		providers = csm.providerOptimizer.ChooseProviderFromTopTier(validAddresses, ignoredProvidersList, cu, requestedBlock)
	} else {
		// Make a copy of ignoredProvidersList to avoid modifying the original
		ignoredProvidersListCopy := make(map[string]struct{}, len(ignoredProvidersList))
		for k, v := range ignoredProvidersList {
			ignoredProvidersListCopy[k] = v
		}
		for i := 0; i < wantedProviders; i++ {
			provider, _ := csm.providerOptimizer.ChooseProvider(validAddresses, ignoredProvidersListCopy, cu, requestedBlock)
			if len(provider) == 0 {
				break
			}
			for _, provider := range provider {
				ignoredProvidersListCopy[provider] = struct{}{}
			}
			providers = append(providers, provider...)
		}
	}

	utils.LavaFormatInfo("Choosing providers",
		utils.LogAttr("validAddresses", validAddresses),
		utils.LogAttr("ignoredProvidersList", ignoredProvidersList),
		utils.LogAttr("chosenProviders", providers),
		utils.LogAttr("addon", addon),
		utils.LogAttr("extensions", extensions),
		utils.LogAttr("stateful", stateful),
		utils.LogAttr("GUID", ctx),
	)

	// Archive-specific debug logging
	if len(extensions) > 0 {
		utils.LavaFormatTrace("[Archive Debug] Final provider selection",
			utils.LogAttr("validAddresses", validAddresses),
			utils.LogAttr("extensions", extensions),
			utils.LogAttr("chosenProviders", providers),
			utils.LogAttr("GUID", ctx))
	}

	// make sure we have at least 1 valid provider
	if len(providers) == 0 || providers[0] == "" {
		utils.LavaFormatDebug("No providers returned by the optimizer", utils.Attribute{Key: "Provider list", Value: validAddresses}, utils.Attribute{Key: "IgnoredProviderList", Value: ignoredProvidersList}, utils.LogAttr("GUID", ctx))
		err = PairingListEmptyError
		return addresses, err
	}

	// If stickiness is requested, store the first provider for future use
	if stickiness != "" {
		utils.LavaFormatTrace("setting sticky session", utils.LogAttr("provider", providers[0]), utils.LogAttr("id", stickiness), utils.LogAttr("GUID", ctx))
		csm.stickySessions.Set(stickiness, &StickySession{
			Provider: providers[0],
			Epoch:    csm.atomicReadCurrentEpoch(),
		})
		return []string{providers[0]}, nil
	}
	return providers, nil
}

// On cases where the valid provider list is empty, by being already used in this attempt, and we got to a point
// where we need another session (for retry or a timeout happened) we want to try fetching a blocked provider for the list.
// the list will be sorted by most cu served giving the best provider that was blocked a second chance to get back to valid addresses.
func (csm *ConsumerSessionManager) tryGetConsumerSessionWithProviderFromBlockedProviderList(ctx context.Context, wantedProviderNumber int, ignoredProviders *ignoredProviders, cuNeededForSession uint64, requestedBlock int64, addon string, extensions []string, stateful uint32, virtualEpoch uint64, usedProviders UsedProvidersInf) (sessionWithProviderMap SessionWithProviderMap, err error) {
	csm.lock.RLock()
	// we do not defer yet as we might need to unlock due to an epoch change

	// reading the epoch here while locked, to get the epoch of the pairing.
	currentEpoch := csm.atomicReadCurrentEpoch()

	// if len(csm.currentlyBlockedProviderAddresses) == 0 we probably reset the state so we can fetch it normally OR ||
	// on a very rare case epoch change can happen. in this case we should just fetch a provider from the new pairing list.
	// we also enter this case if all validAddresses are inside ignoredProviders
	if len(csm.currentlyBlockedProviderAddresses) == 0 || ignoredProviders.currentEpoch < currentEpoch {
		// epoch changed just now (between the getValidConsumerSessionsWithProvider to tryGetConsumerSessionWithProviderFromBlockedProviderList)
		if ignoredProviders.currentEpoch < currentEpoch {
			utils.LavaFormatDebug("Epoch changed between getValidConsumerSessionsWithProvider to tryGetConsumerSessionWithProviderFromBlockedProviderList getting pairing from new epoch list", utils.LogAttr("GUID", ctx))
		}
		csm.lock.RUnlock() // unlock because getValidConsumerSessionsWithProvider is locking.
		return csm.getValidConsumerSessionsWithProvider(ctx, wantedProviderNumber, ignoredProviders, cuNeededForSession, requestedBlock, addon, extensions, stateful, virtualEpoch, "")
	}

	// if we got here we validated the epoch is still the same epoch as we expected and we need to fetch a session from the blocked provider list.
	defer csm.lock.RUnlock()

	routerKey := NewRouterKey(extensions)
	// csm.currentlyBlockedProviderAddresses is sorted by the provider with the highest cu used this epoch to the lowest
	// meaning if we fetch the first successful index this is probably the highest success ratio to get a response.
	for _, providerAddress := range csm.currentlyBlockedProviderAddresses {
		// check if we have this provider already.
		if _, providerExistInIgnoredProviders := ignoredProviders.providers[providerAddress]; providerExistInIgnoredProviders {
			utils.LavaFormatTrace("[continue] provider already in ignored providers", utils.LogAttr("providerAddress", providerAddress), utils.LogAttr("GUID", ctx))
			continue
		}
		consumerSessionsWithProvider := csm.pairing[providerAddress]
		// Add to ignored (no matter what)
		ignoredProviders.providers[providerAddress] = struct{}{}
		usedProviders.AddUnwantedAddresses(providerAddress, routerKey) // add the address to our unwanted providers to avoid infinite recursion

		// validate this provider has enough cu to be used
		if err := consumerSessionsWithProvider.validateComputeUnits(cuNeededForSession, virtualEpoch); err != nil {
			// we already added to ignored we can just continue to the next provider
			utils.LavaFormatTrace("[continue] no compute units", utils.LogAttr("providerAddress", providerAddress), utils.LogAttr("GUID", ctx))
			continue
		}

		// validate this provider supports the required extension or addon
		if !consumerSessionsWithProvider.IsSupportingAddon(addon) || !consumerSessionsWithProvider.IsSupportingExtensions(extensions, ctx) {
			utils.LavaFormatTrace("[continue] no addon or extensions", utils.LogAttr("providerAddress", providerAddress), utils.LogAttr("GUID", ctx))
			continue
		}

		consumerSessionsWithProvider.atomicWriteBlockedStatus(BlockedProviderSessionUsedStatus) // will add to valid addresses if successful
		// If no error, return session map
		return SessionWithProviderMap{
			providerAddress: &SessionWithProvider{
				SessionsWithProvider: consumerSessionsWithProvider,
				CurrentEpoch:         currentEpoch,
				retryConnecting:      true,
			},
		}, nil
	}

	// if we got here we failed to fetch a valid provider meaning no pairing available.
	return nil, utils.LavaFormatError(csm.rpcEndpoint.ChainID+" could not get a provider address from blocked provider list", PairingListEmptyError, utils.LogAttr("csm.currentlyBlockedProviderAddresses", csm.currentlyBlockedProviderAddresses), utils.LogAttr("addons", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("ignoredProviders", ignoredProviders.providers), utils.LogAttr("GUID", ctx))
}

// getValidConsumerSessionsWithProviderFromBackupProviderList retrieves valid backup provider sessions for emergency fallback when no regular providers are available.
func (csm *ConsumerSessionManager) getValidConsumerSessionsWithProviderFromBackupProviderList(ctx context.Context, ignoredProviders *ignoredProviders, cuNeededForSession uint64, requestedBlock int64, addon string, extensions []string, stateful uint32, virtualEpoch uint64, usedProviders UsedProvidersInf) (sessionWithProviderMap SessionWithProviderMap, err error) {
	csm.lock.RLock()
	defer csm.lock.RUnlock()

	utils.LavaFormatTrace("Called getValidConsumerSessionsWithProviderFromBackupProviderList", utils.LogAttr("ignoredProviders", ignoredProviders), utils.LogAttr("GUID", ctx))

	currentEpoch := csm.atomicReadCurrentEpoch() // reading the epoch here while locked, to get the epoch of the pairing.
	if ignoredProviders.currentEpoch < currentEpoch {
		utils.LavaFormatDebug("ignoredProviders epoch is not the current epoch, resetting ignoredProviders", utils.Attribute{Key: "ignoredProvidersEpoch", Value: ignoredProviders.currentEpoch}, utils.Attribute{Key: "currentEpoch", Value: currentEpoch}, utils.LogAttr("GUID", ctx))
		ignoredProviders.providers = make(map[string]struct{}) // reset the old providers as epochs changed so we have a new pairing list.
		ignoredProviders.currentEpoch = currentEpoch
	}

	// Check if backup providers exist
	if len(csm.backupProviders) == 0 {
		utils.LavaFormatDebug("No backup providers configured", utils.LogAttr("GUID", ctx))
		return nil, utils.LavaFormatError("no backup providers configured", nil, utils.LogAttr("GUID", ctx))
	}

	// Get valid backup provider addresses that support the required addon and extensions
	backupProviderAddresses := []string{}
	for providerAddress, consumerSessionsWithProvider := range csm.backupProviders {
		// Skip if provider is in ignored list (already tried or failed)
		if _, exists := ignoredProviders.providers[providerAddress]; exists {
			continue
		}

		// Validate backup provider supports required addons and extensions (simplified validation for emergency scenarios)
		if !consumerSessionsWithProvider.IsSupportingAddon(addon) || !consumerSessionsWithProvider.IsSupportingExtensions(extensions, ctx) {
			continue
		}

		backupProviderAddresses = append(backupProviderAddresses, providerAddress)
	}

	if len(backupProviderAddresses) == 0 {
		utils.LavaFormatDebug("No valid backup providers found", utils.LogAttr("GUID", ctx))
		return nil, utils.LavaFormatError("no valid backup providers available", nil, utils.LogAttr("GUID", ctx))
	}

	utils.LavaFormatTrace("getValidConsumerSessionsWithProviderFromBackupProviderList", utils.LogAttr("backupProviderAddresses", backupProviderAddresses), utils.LogAttr("GUID", ctx))

	// Create map to save sessions with providers
	sessionWithProviderMap = make(SessionWithProviderMap, len(backupProviderAddresses))

	// Iterate over backup providers and create sessions (all-or-nothing approach for emergency scenarios)
	for _, providerAddress := range backupProviderAddresses {
		consumerSessionsWithProvider := csm.backupProviders[providerAddress]
		if consumerSessionsWithProvider == nil {
			utils.LavaFormatFatal("invalid backup provider address", nil,
				utils.Attribute{Key: "providerAddress", Value: providerAddress},
				utils.Attribute{Key: "all_backupProviderAddresses", Value: backupProviderAddresses},
				utils.Attribute{Key: "backupProviders", Value: csm.backupProviders},
				utils.Attribute{Key: "epochAtStart", Value: currentEpoch},
				utils.Attribute{Key: "currentEpoch", Value: csm.atomicReadCurrentEpoch()},
				utils.LogAttr("GUID", ctx),
			)
		}

		// Add provider session map with current epoch for consistency
		sessionWithProviderMap[providerAddress] = &SessionWithProvider{
			SessionsWithProvider: consumerSessionsWithProvider,
			CurrentEpoch:         currentEpoch,
		}

		// Add to ignored to avoid reusing the same provider in subsequent calls
		ignoredProviders.providers[providerAddress] = struct{}{}
	}

	utils.LavaFormatTrace("getValidConsumerSessionsWithProviderFromBackupProviderList", utils.LogAttr("sessionWithProviderMap", sessionWithProviderMap), utils.LogAttr("GUID", ctx))

	return sessionWithProviderMap, nil
}

func (csm *ConsumerSessionManager) getValidConsumerSessionsWithProvider(ctx context.Context, wantedProviderNumber int, ignoredProviders *ignoredProviders, cuNeededForSession uint64, requestedBlock int64, addon string, extensions []string, stateful uint32, virtualEpoch uint64, stickiness string) (sessionWithProviderMap SessionWithProviderMap, err error) {
	csm.lock.RLock()
	defer csm.lock.RUnlock()

	utils.LavaFormatTrace("Called getValidConsumerSessionsWithProvider", utils.LogAttr("wantedProviderNumber", wantedProviderNumber), utils.LogAttr("ignoredProviders", ignoredProviders), utils.LogAttr("GUID", ctx))

	currentEpoch := csm.atomicReadCurrentEpoch() // reading the epoch here while locked, to get the epoch of the pairing.
	if ignoredProviders.currentEpoch < currentEpoch {
		utils.LavaFormatDebug("ignoredProviders epoch is not the current epoch, resetting ignoredProviders", utils.Attribute{Key: "ignoredProvidersEpoch", Value: ignoredProviders.currentEpoch}, utils.Attribute{Key: "currentEpoch", Value: currentEpoch}, utils.LogAttr("GUID", ctx))
		ignoredProviders.providers = make(map[string]struct{}) // reset the old providers as epochs changed so we have a new pairing list.
		ignoredProviders.currentEpoch = currentEpoch
	}

	// Fetch provider addresses
	providerAddresses, err := csm.getValidProviderAddresses(ctx, wantedProviderNumber, ignoredProviders.providers, cuNeededForSession, requestedBlock, addon, extensions, stateful, stickiness)
	if err != nil {
		utils.LavaFormatDebug(csm.rpcEndpoint.ChainID+" could not get a provider addresses", utils.LogAttr("error", err), utils.LogAttr("GUID", ctx))
		return nil, err
	}

	// save how many providers we are aiming to return
	wantedProviderNumber = len(providerAddresses)

	// Create map to save sessions with providers
	sessionWithProviderMap = make(SessionWithProviderMap, wantedProviderNumber)

	// Iterate till we fill map or do not have more
	for {
		// Iterate over providers
		for _, providerAddress := range providerAddresses {
			consumerSessionsWithProvider := csm.pairing[providerAddress]
			if consumerSessionsWithProvider == nil {
				utils.LavaFormatFatal("invalid provider address returned from csm.getValidProviderAddresses", nil,
					utils.Attribute{Key: "providerAddress", Value: providerAddress},
					utils.Attribute{Key: "all_providerAddresses", Value: providerAddresses},
					utils.Attribute{Key: "pairing", Value: csm.pairing},
					utils.Attribute{Key: "epochAtStart", Value: currentEpoch},
					utils.Attribute{Key: "currentEpoch", Value: csm.atomicReadCurrentEpoch()},
					utils.Attribute{Key: "validAddresses", Value: csm.getValidAddresses(addon, extensions, ctx)},
					utils.Attribute{Key: "wantedProviderNumber", Value: wantedProviderNumber},
					utils.LogAttr("GUID", ctx),
				)
			}
			if err := consumerSessionsWithProvider.validateComputeUnits(cuNeededForSession, virtualEpoch); err != nil {
				// Add to ignored
				ignoredProviders.providers[providerAddress] = struct{}{}
				continue
			}

			// If no error, add provider session map
			sessionWithProviderMap[providerAddress] = &SessionWithProvider{
				SessionsWithProvider: consumerSessionsWithProvider,
				CurrentEpoch:         currentEpoch,
			}
			// Add to ignored
			ignoredProviders.providers[providerAddress] = struct{}{}

			// If we have enough providers return
			if len(sessionWithProviderMap) == wantedProviderNumber {
				return sessionWithProviderMap, nil
			}
		}

		// If we do not have enough fetch more
		providerAddresses, err = csm.getValidProviderAddresses(ctx, 1, ignoredProviders.providers, cuNeededForSession, requestedBlock, addon, extensions, stateful, stickiness)

		// If error exists but we have providers, return them
		if err != nil && len(sessionWithProviderMap) != 0 {
			return sessionWithProviderMap, nil
		}

		// If error happens, and we do not have any provider return error
		if err != nil {
			utils.LavaFormatError("could not get a provider addresses", err, utils.LogAttr("GUID", ctx))
			return nil, err
		}
	}
}

// must be locked before use
func (csm *ConsumerSessionManager) sortBlockedProviderListByCuServed() {
	// Defining the custom sorting rule (used cu per provider)
	// descending order of cu used (highest to lowest)
	customSort := func(i, j int) bool {
		return csm.pairing[csm.currentlyBlockedProviderAddresses[i]].atomicReadUsedComputeUnits() > csm.pairing[csm.currentlyBlockedProviderAddresses[j]].atomicReadUsedComputeUnits()
	}
	// Sort the slice using the custom sorting rule
	sort.Slice(csm.currentlyBlockedProviderAddresses, customSort)
}

// removes a given address from the valid addresses list.
func (csm *ConsumerSessionManager) removeAddressFromValidAddresses(address string) error {
	// cs Must be Locked here.
	for idx, addr := range csm.validAddresses {
		if addr == address {
			// remove the index from the valid list.
			csm.validAddresses = append(csm.validAddresses[:idx], csm.validAddresses[idx+1:]...)
			csm.RemoveAddonAddresses("", nil)
			// add the address to our block provider list.
			csm.currentlyBlockedProviderAddresses = append(csm.currentlyBlockedProviderAddresses, address)
			utils.LavaFormatInfo("➕ ADDED TO BLOCKED LIST", utils.LogAttr("address", address), utils.LogAttr("newBlockedCount", len(csm.currentlyBlockedProviderAddresses)))
			// sort the blocked provider list by cu served
			csm.sortBlockedProviderListByCuServed()
			provider, ok := csm.pairing[addr]
			if ok {
				info := csm.RPCEndpoint()
				go func(networkAddress string, chainId string, apiInterface string, providerAddress string) {
					csm.consumerMetricsManager.SetBlockedProvider(chainId, apiInterface, providerAddress, networkAddress, true)
				}(provider.Endpoints[0].NetworkAddress, info.ChainID, info.ApiInterface, addr)
			}
			return nil
		}
	}
	return AddressIndexWasNotFoundError
}

// Blocks a provider making him unavailable for pick this epoch, will also report him as unavailable if reportProvider is set to true.
// Validates that the sessionEpoch is equal to cs.currentEpoch otherwise doesn't take effect.
func (csm *ConsumerSessionManager) blockProvider(ctx context.Context, address string, reportProvider bool, sessionEpoch uint64, disconnections uint64, errors uint64, allowSecondChance bool, reconnectCallback func() error, errorsForReport []error) error {
	utils.LavaFormatInfo("🔒 BLOCKING PROVIDER", utils.LogAttr("address", address), utils.LogAttr("currentBlockedCount", len(csm.currentlyBlockedProviderAddresses)), utils.LogAttr("errorsForReport", errorsForReport), utils.LogAttr("GUID", ctx))

	// find Index of the address
	if sessionEpoch != csm.atomicReadCurrentEpoch() { // we read here atomically so cs.currentEpoch cant change in the middle, so we can save time if epochs mismatch
		return EpochMismatchError
	}

	var runSecondChance bool
	csm.lock.Lock() // we lock RW here because we need to make sure nothing changes while we verify validAddresses/addedToPurgeAndReport
	// on unlock we also want to trigger a routine that will remove blocked providers from block list if they exist and we allow them a second chance
	defer func() {
		csm.lock.Unlock()
		if runSecondChance {
			// if we decide to allow a second chance, this provider will return to our list of valid providers (if it exists)
			go func() {
				<-time.After(retrySecondChanceAfter)
				// check epoch is still relevant, if not just return
				if sessionEpoch != csm.atomicReadCurrentEpoch() {
					return
				}
				utils.LavaFormatDebug("Running second chance for provider", utils.LogAttr("address", address), utils.LogAttr("GUID", ctx))
				csm.validateAndReturnBlockedProviderToValidAddressesList(address)
			}()
		}
	}()

	if sessionEpoch != csm.atomicReadCurrentEpoch() { // After we lock we need to verify again that the epoch didn't change while we waited for the lock.
		return EpochMismatchError
	}

	err := csm.removeAddressFromValidAddresses(address)
	if err != nil {
		if AddressIndexWasNotFoundError.Is(err) {
			// in case index wasn't  found just continue with the method
			utils.LavaFormatDebug("address was not found in valid addresses list", utils.Attribute{Key: "address", Value: address}, utils.Attribute{Key: "error", Value: err}, utils.Attribute{Key: "validAddresses", Value: csm.validAddresses}, utils.LogAttr("GUID", ctx))
		} else {
			return err
		}
	}

	if reportProvider { // Report provider flow
		if allowSecondChance { // on epoch change, we don't report providers immediately we allow them a recovery phase.
			if _, ok := csm.secondChanceGivenToAddresses[address]; ok {
				// already exists in second chance, need to block.
				csm.reportedProviders.ReportProvider(address, errors, disconnections, reconnectCallback, errorsForReport)
			} else {
				// first time reported, allowing a second chance.
				csm.secondChanceGivenToAddresses[address] = struct{}{}
				// address was removed from valid addresses, we can still return it after a duration for second chance.
				runSecondChance = true
			}
		} else {
			csm.reportedProviders.ReportProvider(address, errors, disconnections, reconnectCallback, errorsForReport)
		}
	}

	return nil
}

// Report session failure, mark it as blocked from future usages, report if timeout happened.
func (csm *ConsumerSessionManager) OnSessionFailure(consumerSession *SingleConsumerSession, errorReceived error) error {
	// consumerSession must be locked when getting here.
	if err := consumerSession.VerifyLock(); err != nil {
		return sdkerrors.Wrapf(err, "OnSessionFailure, consumerSession.lock must be locked before accessing this method, additional info:")
	}
	// redemptionSession = true, if we got this provider from the blocked provider list.
	// if so, it means we already reported this provider and blocked it we do not need to do it again.
	// due to session failure we also don't need to remove it from the blocked provider list.
	// we will just update the QOS info, and return
	redemptionSession := consumerSession.Parent.atomicReadBlockedStatus() == BlockedProviderSessionUsedStatus

	// consumer Session should be locked here. so we can just apply the session failure here.
	if consumerSession.BlockListed {
		// if consumer session is already blocklisted return an error.
		return sdkerrors.Wrapf(SessionIsAlreadyBlockListedError, "trying to report a session failure of a blocklisted consumer session")
	}

	// check if need to block & report
	var blockProvider, reportProvider bool
	if sdkerrors.IsOf(errorReceived, ReportAndBlockProviderError) {
		blockProvider = true
		reportProvider = true
	} else if sdkerrors.IsOf(errorReceived, BlockProviderError) {
		blockProvider = true
	}

	if sdkerrors.IsOf(errorReceived, BlockEndpointError) {
		utils.LavaFormatTrace("Got BlockEndpointError, blocking endpoint and session",
			utils.LogAttr("error", errorReceived),
			utils.LogAttr("sessionID", consumerSession.SessionId),
		)

		// Block the endpoint and the consumer session from future usages
		consumerSession.EndpointConnection.blockListed.Store(true)
		consumerSession.BlockListed = true
	}

	csm.qosManager.AddFailedRelay(consumerSession.epoch, consumerSession.SessionId)
	consumerSession.ConsecutiveErrors = append(consumerSession.ConsecutiveErrors, errorReceived)
	// copy consecutive errors for report.
	errorsForConsumerSession := consumerSession.ConsecutiveErrors
	consumerSession.errorsCount += 1
	// set allow second change if we want to allow the provider to return the pool without being reported if the downtime was temporary.
	allowSecondChance := false
	// if this session failed more than MaximumNumberOfFailuresAllowedPerConsumerSession times or session went out of sync we block it.
	if len(consumerSession.ConsecutiveErrors) > MaximumNumberOfFailuresAllowedPerConsumerSession || IsSessionSyncLoss(errorReceived) {
		utils.LavaFormatDebug("Blocking consumer session",
			utils.LogAttr("ConsecutiveErrors", consumerSession.ConsecutiveErrors),
			utils.LogAttr("errorsCount", consumerSession.errorsCount),
			utils.LogAttr("id", consumerSession.SessionId),
		)
		consumerSession.BlockListed = true // block this session from future usages

		// check if this session is a redemption session meaning we already blocked and reported the provider if it was necessary.
		if !redemptionSession {
			// we will check the total number of cu for this provider and decide if we need to report it.
			if consumerSession.Parent.atomicReadUsedComputeUnits() <= consumerSession.LatestRelayCu { // if we had 0 successful relays and we reached block session we need to report this provider
				blockProvider = true
				reportProvider = true
				allowSecondChance = true
			}
		}
	}
	cuToDecrease := consumerSession.LatestRelayCu
	// latency, isHangingApi, syncScore aren't updated when there is a failure
	go csm.providerOptimizer.AppendRelayFailure(consumerSession.Parent.PublicLavaAddress)
	consumerSession.LatestRelayCu = 0 // making sure no one uses it in a wrong way
	consecutiveErrors := uint64(len(consumerSession.ConsecutiveErrors))
	parentConsumerSessionsWithProvider := consumerSession.Parent // must read this pointer before unlocking
	csm.updateMetricsManager(consumerSession, time.Duration(0), false)
	// finished with consumerSession here can unlock.
	consumerSession.Free(errorReceived) // we unlock before we change anything in the parent ConsumerSessionsWithProvider

	err := parentConsumerSessionsWithProvider.decreaseUsedComputeUnits(cuToDecrease) // change the cu in parent
	if err != nil {
		return err
	}

	if !redemptionSession && blockProvider {
		publicProviderAddress, pairingEpoch := parentConsumerSessionsWithProvider.getPublicLavaAddressAndPairingEpoch()
		err = csm.blockProvider(context.Background(), publicProviderAddress, reportProvider, pairingEpoch, 0, consecutiveErrors, allowSecondChance, nil, errorsForConsumerSession)
		if err != nil {
			if EpochMismatchError.Is(err) {
				return nil // no effects this epoch has been changed
			}
			return err
		}
	}
	return nil
}

// validating if the provider is currently not in valid addresses list. if the session was successful we can return the provider
// to our valid addresses list and resume its usage
func (csm *ConsumerSessionManager) validateAndReturnBlockedProviderToValidAddressesList(providerAddress string) {
	csm.lock.Lock()
	defer csm.lock.Unlock()
	csm.validateAndReturnBlockedProviderToValidAddressesListLocked(providerAddress)
}

// internal version that assumes csm.lock is already held
func (csm *ConsumerSessionManager) validateAndReturnBlockedProviderToValidAddressesListLocked(providerAddress string) {
	for idx, addr := range csm.currentlyBlockedProviderAddresses {
		if addr == providerAddress {
			// Remove it from the csm.currentlyBlockedProviderAddresses
			csm.currentlyBlockedProviderAddresses = append(csm.currentlyBlockedProviderAddresses[:idx], csm.currentlyBlockedProviderAddresses[idx+1:]...)
			// Reapply it to the valid addresses.
			csm.validAddresses = append(csm.validAddresses, addr)
			// Purge the current addon addresses so it will be created again next time get session is called.
			csm.RemoveAddonAddresses("", nil)
			// Reset redemption status
			if provider, ok := csm.pairing[providerAddress]; ok {
				info := csm.RPCEndpoint()
				provider.atomicWriteBlockedStatus(BlockedProviderSessionUnusedStatus)
				go func(networkAddress string, chainId string, apiInterface string, providerAddress string) {
					csm.consumerMetricsManager.SetBlockedProvider(chainId, apiInterface, providerAddress, networkAddress, false)
				}(provider.Endpoints[0].NetworkAddress, info.ChainID, info.ApiInterface, providerAddress)
			}
			return
		}
	}
	// if we didn't find it, we might had two sessions in parallel and thats ok. the first one dealt with it we can just return
}

// On a successful session this function will update all necessary fields in the consumerSession. and unlock it when it finishes
func (csm *ConsumerSessionManager) OnSessionDone(
	consumerSession *SingleConsumerSession,
	latestServicedBlock int64,
	specComputeUnits uint64,
	currentLatency time.Duration,
	expectedLatency time.Duration,
	expectedBH int64,
	numOfProviders int,
	providersCount uint64,
	isHangingApi bool,
	extensions []*spectypes.Extension,
) error {
	// release locks, update CU, relaynum etc..
	if err := consumerSession.VerifyLock(); err != nil {
		return sdkerrors.Wrapf(err, "OnSessionDone, consumerSession.lock must be locked before accessing this method")
	}

	if consumerSession.Parent.atomicReadBlockedStatus() == BlockedProviderSessionUsedStatus {
		// we will deal with the removal of this provider from the blocked list so we can for now set it as default
		consumerSession.Parent.atomicWriteBlockedStatus(BlockedProviderSessionUnusedStatus)
		// this provider is probably in the ignored provider list. we need to validate and return it to valid addresses
		providerAddress := consumerSession.Parent.PublicLavaAddress
		// we want this method to run last after we unlock the consumer session
		// golang defer operates in a Last-In-First-Out (LIFO) order, meaning this defer will run last.
		defer func() { go csm.validateAndReturnBlockedProviderToValidAddressesList(providerAddress) }()
	}

	defer consumerSession.Free(nil)                        // we need to be locked here, if we didn't get it locked we try lock anyway
	consumerSession.CuSum += consumerSession.LatestRelayCu // add CuSum to current cu usage.
	consumerSession.LatestRelayCu = 0                      // reset cu just in case
	consumerSession.ConsecutiveErrors = []error{}
	consumerSession.LatestBlock = latestServicedBlock // update latest serviced block
	// calculate QoS
	csm.qosManager.CalculateQoS(csm.atomicReadCurrentEpoch(), consumerSession.SessionId, consumerSession.Parent.PublicLavaAddress, currentLatency, expectedLatency, expectedBH-latestServicedBlock, numOfProviders, int64(providersCount))
	if !isHangingApi {
		// append relay data only for non hanging apis
		go csm.providerOptimizer.AppendRelayData(consumerSession.Parent.PublicLavaAddress, currentLatency, specComputeUnits, uint64(latestServicedBlock))
	}

	csm.updateMetricsManager(consumerSession, currentLatency, !isHangingApi) // apply latency only for non hanging apis
	return nil
}

// updates QoS metrics for a provider
// consumerSession should still be locked when accessing this method as it fetches information from the session it self
func (csm *ConsumerSessionManager) updateMetricsManager(consumerSession *SingleConsumerSession, relayLatency time.Duration, sessionSuccessful bool) {
	if csm.consumerMetricsManager == nil {
		return
	}
	info := csm.RPCEndpoint()
	apiInterface := info.ApiInterface
	chainId := info.ChainID

	var lastQos *pairingtypes.QualityOfServiceReport
	lastQoSReport := csm.qosManager.GetLastQoSReport(csm.atomicReadCurrentEpoch(), consumerSession.SessionId)
	if lastQoSReport != nil {
		qos := *lastQoSReport
		lastQos = &qos
	}

	var lastReputation *pairingtypes.QualityOfServiceReport
	lastReputationReport := csm.qosManager.GetLastReputationQoSReport(csm.atomicReadCurrentEpoch(), consumerSession.SessionId)
	if lastReputationReport != nil {
		qosRep := *lastReputationReport
		lastReputation = &qosRep
	}
	publicProviderAddress := consumerSession.Parent.PublicLavaAddress
	publicProviderEndpoint := consumerSession.Parent.Endpoints[0].NetworkAddress

	go func() {
		csm.consumerMetricsManager.SetQOSMetrics(chainId, apiInterface, publicProviderAddress, publicProviderEndpoint, lastQos, lastReputation, consumerSession.LatestBlock, consumerSession.RelayNum, relayLatency, sessionSuccessful)
	}()
}

// Get the reported providers currently stored in the session manager.
func (csm *ConsumerSessionManager) GetReportedProviders(epoch uint64) []*pairingtypes.ReportedProvider {
	if epoch != csm.atomicReadCurrentEpoch() {
		return nil // if epochs are not equal, we will return an empty list.
	}
	reportedProviders := csm.reportedProviders.GetReportedProviders()
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	filteredReportedProviders := []*pairingtypes.ReportedProvider{}
	for _, reportedProvider := range reportedProviders {
		provider, ok := csm.pairing[reportedProvider.Address]
		if !ok {
			// that shouldn't happen
			utils.LavaFormatError("Failed to find a reported provider in pairing list", nil, utils.LogAttr("provider_address", reportedProvider.Address), utils.LogAttr("epoch", csm.currentEpoch))
			continue
		}
		if provider.doesProviderEndpointsContainGeolocation(csm.RPCEndpoint().Geolocation) {
			filteredReportedProviders = append(filteredReportedProviders, reportedProvider)
		}
	}
	return filteredReportedProviders
}

// Atomically read csm.pairingAddressesLength for data reliability.
func (csm *ConsumerSessionManager) GetAtomicPairingAddressesLength() uint64 {
	return atomic.LoadUint64(&csm.pairingAddressesLength)
}

// On a successful Subscribe relay
func (csm *ConsumerSessionManager) OnSessionDoneIncreaseCUOnly(consumerSession *SingleConsumerSession, latestServicedBlock int64) error {
	if err := consumerSession.VerifyLock(); err != nil {
		return sdkerrors.Wrapf(err, "OnSessionDoneIncreaseRelayAndCu consumerSession.lock must be locked before accessing this method")
	}

	defer consumerSession.Free(nil) // we need to be locked here, if we didn't get it locked we try lock anyway
	consumerSession.LatestBlock = latestServicedBlock
	consumerSession.CuSum += consumerSession.LatestRelayCu // add CuSum to current cu usage.
	consumerSession.LatestRelayCu = 0                      // reset cu just in case
	consumerSession.ConsecutiveErrors = []error{}
	return nil
}

func (csm *ConsumerSessionManager) GenerateReconnectCallback(consumerSessionsWithProvider *ConsumerSessionsWithProvider) func() error {
	return func() error {
		ctx := utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier()) // unique identifier for retries
		_, providerAddress, err := csm.probeProvider(ctx, consumerSessionsWithProvider, csm.atomicReadCurrentEpoch(), true)
		if err == nil {
			utils.LavaFormatDebug("Reconnecting provider succeeded returning provider to valid addresses list", utils.LogAttr("provider", providerAddress))
			csm.validateAndReturnBlockedProviderToValidAddressesList(providerAddress)
		}
		return err
	}
}

// checkAndUnblockHealthyReBlockedProviders checks providers that were re-blocked from previous epoch
// and immediately unblocks them if their probe was successful. This only happens at epoch transitions.
// Other providers and normal blocking behavior during an epoch remain unchanged.
func (csm *ConsumerSessionManager) checkAndUnblockHealthyReBlockedProviders(ctx context.Context, epoch uint64) {
	// Ensure context has unique identifier for probing
	if _, found := utils.GetUniqueIdentifier(ctx); !found {
		ctx = utils.AppendUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
	}

	csm.lock.Lock()

	// First pass: Identify which re-blocked providers had successful probes
	providersNeedingComprehensiveProbe := make(map[string]*ConsumerSessionsWithProvider)

	for blockedAddr := range csm.previousEpochBlockedProviders {
		cswp, exists := csm.pairing[blockedAddr]
		if !exists {
			continue // Provider not in current pairing
		}

		// Check if provider is in reported providers
		// If probe FAILED in this epoch, provider would be in reportedProviders (line 1208)
		// If probe SUCCEEDED in this epoch, provider would NOT be in reportedProviders
		if !csm.reportedProviders.IsReported(blockedAddr) {
			// Probe succeeded! Provider is healthy, immediately unblock
			utils.LavaFormatInfo("Re-blocked provider's probe succeeded, immediately unblocking",
				utils.Attribute{Key: "provider", Value: blockedAddr},
				utils.Attribute{Key: "epoch", Value: epoch},
				utils.LogAttr("GUID", ctx),
			)
			csm.validateAndReturnBlockedProviderToValidAddressesListLocked(blockedAddr)

			// Clean up: Remove from reported providers if it was there from previous epoch
			// This prevents periodic reconnection attempts from trying this provider again
			csm.reportedProviders.RemoveReport(blockedAddr)
		} else {
			// Probe failed with tryReconnect=false
			// Mark for comprehensive probe with tryReconnect=true to retry disabled endpoints
			providersNeedingComprehensiveProbe[blockedAddr] = cswp
			utils.LavaFormatDebug("Re-blocked provider's initial probe failed, will try comprehensive probe",
				utils.Attribute{Key: "provider", Value: blockedAddr},
				utils.Attribute{Key: "epoch", Value: epoch},
				utils.LogAttr("GUID", ctx),
			)
		}
	}
	csm.lock.Unlock()

	// Second pass: For providers that failed initial probe, try comprehensive probe with reconnection
	// This gives disabled endpoints a chance to be retried and re-enabled
	for blockedAddr, cswp := range providersNeedingComprehensiveProbe {
		utils.LavaFormatDebug("Attempting comprehensive probe with endpoint reconnection",
			utils.Attribute{Key: "provider", Value: blockedAddr},
			utils.Attribute{Key: "epoch", Value: epoch},
			utils.LogAttr("GUID", ctx),
		)

		// Probe with tryReconnect=TRUE - retry disabled endpoints
		_, providerAddress, err := csm.probeProvider(
			ctx,
			cswp,
			epoch,
			true, // tryReconnect=TRUE: Comprehensive probe, retries disabled endpoints
		)

		if err == nil {
			// Comprehensive probe succeeded! Provider recovered, unblock immediately
			utils.LavaFormatInfo("Re-blocked provider's comprehensive probe succeeded, immediately unblocking",
				utils.Attribute{Key: "provider", Value: providerAddress},
				utils.Attribute{Key: "epoch", Value: epoch},
				utils.LogAttr("GUID", ctx),
			)
			csm.validateAndReturnBlockedProviderToValidAddressesList(providerAddress)
			csm.reportedProviders.RemoveReport(providerAddress)
		} else {
			// Still failing even with comprehensive probe, keep blocked
			utils.LavaFormatDebug("Re-blocked provider still unhealthy after comprehensive probe, keeping blocked",
				utils.Attribute{Key: "provider", Value: providerAddress},
				utils.Attribute{Key: "error", Value: err.Error()},
				utils.Attribute{Key: "epoch", Value: epoch},
				utils.LogAttr("GUID", ctx),
			)
			// Provider stays in reportedProviders and will be retried by periodic reconnection (30s)
		}
	}
}

func NewConsumerSessionManager(
	rpcEndpoint *RPCEndpoint,
	providerOptimizer ProviderOptimizer,
	consumerMetricsManager *metrics.ConsumerMetricsManager,
	reporter metrics.Reporter,
	consumerPublicAddress string,
	activeSubscriptionProvidersStorage *ActiveSubscriptionProvidersStorage,
) *ConsumerSessionManager {
	csm := &ConsumerSessionManager{
		reportedProviders:      NewReportedProviders(reporter, rpcEndpoint.ChainID),
		consumerMetricsManager: consumerMetricsManager,
		consumerPublicAddress:  consumerPublicAddress,
		qosManager:             qos.NewQoSManager(),
		getLavaBlockHeight:     func() int64 { return 0 }, // default to 0, should be set by caller
	}
	csm.rpcEndpoint = rpcEndpoint
	csm.providerOptimizer = providerOptimizer
	csm.activeSubscriptionProvidersStorage = activeSubscriptionProvidersStorage
	csm.stickySessions = NewStickySessionStore()
	return csm
}

// SetLavaBlockHeightCallback sets the callback function to get current Lava blockchain block height
// This must be called after creating the ConsumerSessionManager
func (csm *ConsumerSessionManager) SetLavaBlockHeightCallback(getLavaBlockHeight func() int64) {
	csm.getLavaBlockHeight = getLavaBlockHeight
}
