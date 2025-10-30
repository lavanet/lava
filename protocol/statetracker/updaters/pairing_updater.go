package updaters

import (
	"math"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v5/x/plans/types"
	"golang.org/x/net/context"
)

const (
	CallbackKeyForPairingUpdate = "pairing-update"
)

type PairingUpdatable interface {
	UpdateEpoch(epoch uint64)
}

type ConsumerStateQueryInf interface {
	GetPairing(ctx context.Context, chainID string, blockHeight int64) ([]epochstoragetypes.StakeEntry, uint64, uint64, error)
	GetMaxCUForUser(ctx context.Context, chainID string, epoch uint64) (uint64, error)
}

type ConsumerSessionManagerInf interface {
	RPCEndpoint() lavasession.RPCEndpoint
	UpdateAllProviders(epoch uint64, pairingList map[uint64]*lavasession.ConsumerSessionsWithProvider, backupProviderList map[uint64]*lavasession.ConsumerSessionsWithProvider) error
}

type PairingUpdater struct {
	lock                       sync.RWMutex
	consumerSessionManagersMap map[string][]ConsumerSessionManagerInf // key is chainID so we don;t run getPairing more than once per chain
	nextBlockForUpdate         uint64
	stateQuery                 ConsumerStateQueryInf
	pairingUpdatables          []*PairingUpdatable
	specId                     string
	staticProviders            []*lavasession.RPCStaticProviderEndpoint
	backupProviders            []*lavasession.RPCStaticProviderEndpoint
}

func NewPairingUpdater(stateQuery ConsumerStateQueryInf, specId string) *PairingUpdater {
	return &PairingUpdater{consumerSessionManagersMap: map[string][]ConsumerSessionManagerInf{}, stateQuery: stateQuery, specId: specId, staticProviders: []*lavasession.RPCStaticProviderEndpoint{}}
}

func (pu *PairingUpdater) updateStaticProviders(staticProviders []*lavasession.RPCStaticProviderEndpoint) int {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	if len(staticProviders) > 0 && len(pu.staticProviders) == 0 {
		for _, staticProvider := range staticProviders {
			if staticProvider.ChainID == pu.specId {
				pu.staticProviders = append(pu.staticProviders, staticProvider)
			}
		}
	}
	// return length of relevant static providers
	return len(pu.staticProviders)
}

func (pu *PairingUpdater) updateBackupProviders(backupProviders []*lavasession.RPCStaticProviderEndpoint) {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	utils.LavaFormatInfo("Updating backup providers", utils.Attribute{Key: "backupProviders", Value: len(backupProviders)})
	if len(backupProviders) > 0 && len(pu.backupProviders) == 0 {
		for _, backupProvider := range backupProviders {
			if backupProvider.ChainID == pu.specId {
				pu.backupProviders = append(pu.backupProviders, backupProvider)
			}
		}
	}
	utils.LavaFormatInfo("Updating backup providers", utils.Attribute{Key: "backupProviders", Value: len(pu.backupProviders)})
}

func (pu *PairingUpdater) getNumberOfStaticProviders() int {
	pu.lock.RLock()
	defer pu.lock.RUnlock()
	return len(pu.staticProviders)
}

func (pu *PairingUpdater) RegisterPairing(ctx context.Context, consumerSessionManager ConsumerSessionManagerInf, staticProviders []*lavasession.RPCStaticProviderEndpoint, backupProviders []*lavasession.RPCStaticProviderEndpoint) error {
	chainID := consumerSessionManager.RPCEndpoint().ChainID
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	pairingList, epoch, nextBlockForUpdate, err := pu.stateQuery.GetPairing(timeoutCtx, chainID, -1) // fetching pairing list from blockchain
	numberOfRelevantProviders := pu.updateStaticProviders(staticProviders)                           // updating static providers, updateStaticProviders function returns the number of relevant static providers
	if err != nil && (epoch == 0 || numberOfRelevantProviders == 0) {                                // if there is an error from blockchain AND no static providers || epoch = 0, return error
		return err
	}
	utils.LavaFormatInfo("calling update backup providers", utils.Attribute{Key: "backupProviders", Value: len(backupProviders)})
	pu.updateBackupProviders(backupProviders) // updating backup providers
	pu.updateConsumerSessionManager(ctx, pairingList, consumerSessionManager, epoch)
	if nextBlockForUpdate > pu.nextBlockForUpdate {
		// make sure we don't update twice, this updates pu.nextBlockForUpdate
		pu.Update(int64(nextBlockForUpdate))
	}
	pu.lock.Lock()
	defer pu.lock.Unlock()
	consumerSessionsManagersList, ok := pu.consumerSessionManagersMap[chainID]
	if !ok {
		pu.consumerSessionManagersMap[chainID] = []ConsumerSessionManagerInf{consumerSessionManager}
		return nil
	}
	pu.consumerSessionManagersMap[chainID] = append(consumerSessionsManagersList, consumerSessionManager)
	return nil
}

func (pu *PairingUpdater) RegisterPairingUpdatable(ctx context.Context, pairingUpdatable *PairingUpdatable) error {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	_, epoch, _, err := pu.stateQuery.GetPairing(ctx, pu.specId, -1)
	if err != nil && (epoch == 0 || len(pu.staticProviders) == 0) {
		return err
	}

	(*pairingUpdatable).UpdateEpoch(epoch)
	pu.pairingUpdatables = append(pu.pairingUpdatables, pairingUpdatable)
	return nil
}

func (pu *PairingUpdater) UpdaterKey() string {
	return CallbackKeyForPairingUpdate
}

func (pu *PairingUpdater) updateInner(latestBlock int64) {
	pu.lock.RLock()
	defer pu.lock.RUnlock()
	ctx := context.Background()
	if int64(pu.nextBlockForUpdate) > latestBlock {
		return
	}
	nextBlockForUpdateList := []uint64{}
	for chainID, consumerSessionManagerList := range pu.consumerSessionManagersMap {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		pairingList, epoch, nextBlockForUpdate, err := pu.stateQuery.GetPairing(timeoutCtx, chainID, latestBlock)
		nextBlockForUpdateList = append(nextBlockForUpdateList, nextBlockForUpdate)
		if err != nil && (epoch == 0 || len(pu.staticProviders) == 0) {
			// it's ok that we don't have pairing, only if there are static providers and epoch is not 0
			utils.LavaFormatError("could not update pairing for chain, trying again next block", err, utils.Attribute{Key: "chain", Value: chainID})
			continue
		}
		for _, consumerSessionManager := range consumerSessionManagerList {
			// same pairing for all apiInterfaces, they pick the right endpoints from inside using our filter function
			err = pu.updateConsumerSessionManager(ctx, pairingList, consumerSessionManager, epoch)
			if err != nil {
				utils.LavaFormatError("failed updating consumer session manager", err, utils.Attribute{Key: "chainID", Value: chainID}, utils.Attribute{Key: "apiInterface", Value: consumerSessionManager.RPCEndpoint().ApiInterface}, utils.Attribute{Key: "pairingListLen", Value: len(pairingList)})
				continue
			}
		}
	}

	// get latest epoch from cache
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, epoch, nextPairingUpdateBlock, err := pu.stateQuery.GetPairing(timeoutCtx, pu.specId, latestBlock)
	if err != nil && (epoch == 0 || len(pu.staticProviders) == 0) {
		utils.LavaFormatError("could not update pairing for updatables, trying again next block", err)
	}

	if epoch == 0 {
		nextBlockForUpdateList = append(nextBlockForUpdateList, pu.nextBlockForUpdate+1)
	} else {
		nextBlockForUpdateList = append(nextBlockForUpdateList, nextPairingUpdateBlock)
		for _, updatable := range pu.pairingUpdatables {
			(*updatable).UpdateEpoch(epoch)
		}
	}

	nextBlockForUpdateMin := uint64(latestBlock) // in case the list is empty
	for idx, blockToUpdate := range nextBlockForUpdateList {
		if idx == 0 || blockToUpdate < nextBlockForUpdateMin {
			nextBlockForUpdateMin = blockToUpdate
		}
	}
	pu.nextBlockForUpdate = nextBlockForUpdateMin
}

func (pu *PairingUpdater) Reset(latestBlock int64) {
	utils.LavaFormatDebug("Reset Triggered for PairingUpdater", utils.LogAttr("block", latestBlock))
	pu.updateInner(latestBlock)
}

func (pu *PairingUpdater) Update(latestBlock int64) {
	pu.updateInner(latestBlock)
}

func (pu *PairingUpdater) updateConsumerSessionManager(ctx context.Context, pairingList []epochstoragetypes.StakeEntry, consumerSessionManager ConsumerSessionManagerInf, epoch uint64) (err error) {
	pairingListForThisCSM, err := pu.filterPairingListByEndpoint(ctx, planstypes.Geolocation(consumerSessionManager.RPCEndpoint().Geolocation), pairingList, consumerSessionManager.RPCEndpoint(), epoch)
	if err != nil {
		return err
	}
	if len(pu.staticProviders) > 0 {
		pairingListForThisCSM = pu.addStaticProvidersToPairingList(pairingListForThisCSM, consumerSessionManager.RPCEndpoint(), epoch)
	}
	backupProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	if len(pu.backupProviders) > 0 {
		utils.LavaFormatInfo("Adding backup providers to pairing list", utils.Attribute{Key: "backupProviders", Value: len(pu.backupProviders)})
		backupProviders = pu.parseBackupProviders(pairingListForThisCSM, consumerSessionManager.RPCEndpoint(), epoch)
	}
	err = consumerSessionManager.UpdateAllProviders(epoch, pairingListForThisCSM, backupProviders)
	return err
}

func (pu *PairingUpdater) addStaticProvidersToPairingList(pairingList map[uint64]*lavasession.ConsumerSessionsWithProvider, rpcEndpoint lavasession.RPCEndpoint, epoch uint64) map[uint64]*lavasession.ConsumerSessionsWithProvider {
	staticProviderSessions := pu.createProviderSessionsFromConfig(pu.staticProviders, rpcEndpoint, epoch, pairingList, "StaticProvider_")
	for idx, providerSession := range staticProviderSessions {
		pairingList[idx] = providerSession
	}
	return pairingList
}

// parseBackupProviders transforms raw backup provider data into ConsumerSessionsWithProvider objects for emergency fallback scenarios.
func (pu *PairingUpdater) parseBackupProviders(pairingList map[uint64]*lavasession.ConsumerSessionsWithProvider, rpcEndpoint lavasession.RPCEndpoint, epoch uint64) map[uint64]*lavasession.ConsumerSessionsWithProvider {
	return pu.createProviderSessionsFromConfig(pu.backupProviders, rpcEndpoint, epoch, pairingList, "BackupProvider_")
}

// createProviderSessionsFromConfig is a unified helper function that transforms provider configurations
// into ConsumerSessionsWithProvider objects. It handles both static and backup providers by creating
// the appropriate session objects with proper indexing to avoid conflicts with existing providers.
func (pu *PairingUpdater) createProviderSessionsFromConfig(
	providers []*lavasession.RPCStaticProviderEndpoint,
	rpcEndpoint lavasession.RPCEndpoint,
	epoch uint64,
	existingProviders map[uint64]*lavasession.ConsumerSessionsWithProvider,
	providerNamePrefix string,
) map[uint64]*lavasession.ConsumerSessionsWithProvider {
	providerSessions := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

	// Calculate start index to avoid conflicts with existing provider indices
	startIdx := uint64(0)
	for key := range existingProviders {
		if key >= startIdx {
			startIdx = key + 1
		}
	}

	for idx, provider := range providers {
		// Only take the provider entries relevant for this apiInterface
		if provider.ApiInterface != rpcEndpoint.ApiInterface || provider.ChainID != rpcEndpoint.ChainID {
			continue
		}

		endpoints := []*lavasession.Endpoint{}
		for _, url := range provider.NodeUrls {
			extensions := map[string]struct{}{}
			for _, extension := range url.Addons {
				extensions[extension] = struct{}{}
			}

			// TODO might be problematic adding both addons and extensions with same map.
			endpoint := &lavasession.Endpoint{
				NetworkAddress: url.Url,
				Enabled:        true,
				Addons:         extensions,
				Extensions:     extensions,
				Connections:    []*lavasession.EndpointConnection{},
			}
			endpoints = append(endpoints, endpoint)
		}

		// Create provider entry with high compute units and stake for availability
		providerEntry := lavasession.NewConsumerSessionWithProvider(
			provider.Name,
			endpoints,
			math.MaxUint64/2, // High compute units for availability
			epoch,
			sdk.NewInt64Coin("ulava", 1000000000000000), // 1b LAVA stake
		)
		providerEntry.StaticProvider = true // Mark as static provider for simplified handling
		providerSessions[startIdx+uint64(idx)] = providerEntry
	}

	return providerSessions
}

func (pu *PairingUpdater) filterPairingListByEndpoint(ctx context.Context, currentGeo planstypes.Geolocation, pairingList []epochstoragetypes.StakeEntry, rpcEndpoint lavasession.RPCEndpoint, epoch uint64) (filteredList map[uint64]*lavasession.ConsumerSessionsWithProvider, err error) {
	// go over stake entries, and filter endpoints that match geolocation and api interface
	pairing := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	for providerIdx, provider := range pairingList {
		//
		// Sanity
		// only take into account endpoints that use the same api interface and the same geolocation
		// we take all geolocations provided by the chain. the provider optimizer will prioritize the relevant ones
		relevantEndpoints := getRelevantEndpointsFromProvider(provider, rpcEndpoint)
		if len(relevantEndpoints) == 0 {
			utils.LavaFormatError("skipping provider, No relevant endpoints for apiInterface", nil, utils.Attribute{Key: "Address", Value: provider.Address}, utils.Attribute{Key: "ChainID", Value: provider.Chain}, utils.Attribute{Key: "apiInterface", Value: rpcEndpoint.ApiInterface}, utils.Attribute{Key: "Endpoints", Value: provider.GetEndpoints()})
			continue
		}

		maxCu, err := pu.stateQuery.GetMaxCUForUser(ctx, provider.Chain, epoch)
		if err != nil {
			return nil, err
		}
		//
		pairingEndpoints := make([]*lavasession.Endpoint, len(relevantEndpoints))
		for idx, relevantEndpoint := range relevantEndpoints {
			addons := map[string]struct{}{}
			extensions := map[string]struct{}{}
			for _, addon := range relevantEndpoint.Addons {
				addons[addon] = struct{}{}
			}
			for _, extension := range relevantEndpoint.Extensions {
				extensions[extension] = struct{}{}
			}

			endp := &lavasession.Endpoint{Connections: []*lavasession.EndpointConnection{}, Geolocation: planstypes.Geolocation(relevantEndpoint.Geolocation), NetworkAddress: relevantEndpoint.IPPORT, Enabled: true, ConnectionRefusals: 0, Addons: addons, Extensions: extensions}
			pairingEndpoints[idx] = endp
		}
		lavasession.SortByGeolocations(pairingEndpoints, currentGeo)
		totalStakeAmount := provider.Stake.Amount
		if !provider.DelegateTotal.Amount.IsNil() {
			totalStakeAmount = totalStakeAmount.Add(provider.DelegateTotal.Amount)
		}
		totalStakeIncludingDelegation := sdk.Coin{Denom: provider.Stake.Denom, Amount: totalStakeAmount}
		pairing[uint64(providerIdx)] = lavasession.NewConsumerSessionWithProvider(
			provider.Address,
			pairingEndpoints,
			maxCu,
			epoch,
			totalStakeIncludingDelegation,
		)
	}
	if len(pairing)+pu.getNumberOfStaticProviders() == 0 {
		return nil, utils.LavaFormatError("Failed getting pairing for consumer, pairing is empty", err, utils.Attribute{Key: "apiInterface", Value: rpcEndpoint.ApiInterface}, utils.Attribute{Key: "ChainID", Value: rpcEndpoint.ChainID}, utils.Attribute{Key: "geolocation", Value: rpcEndpoint.Geolocation})
	}
	// replace previous pairing with new providers
	return pairing, nil
}

func getRelevantEndpointsFromProvider(provider epochstoragetypes.StakeEntry, rpcEndpoint lavasession.RPCEndpoint) []epochstoragetypes.Endpoint {
	providerEndpoints := provider.GetEndpoints()
	if len(providerEndpoints) == 0 {
		utils.LavaFormatError("skipping provider with no endoints", nil, utils.Attribute{Key: "Address", Value: provider.Address}, utils.Attribute{Key: "ChainID", Value: provider.Chain})
		return nil
	}

	relevantEndpoints := []epochstoragetypes.Endpoint{}
	for _, endpoint := range providerEndpoints {
		for _, endpointApiInterface := range endpoint.ApiInterfaces {
			if endpointApiInterface == rpcEndpoint.ApiInterface {
				relevantEndpoints = append(relevantEndpoints, endpoint)
				break
			}
		}
	}
	return relevantEndpoints
}
