package statetracker

import (
	"sync"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"golang.org/x/net/context"
)

const (
	CallbackKeyForPairingUpdate = "pairing-update"
)

type PairingUpdatable interface {
	UpdateEpoch(epoch uint64)
}

type PairingUpdater struct {
	lock                       sync.RWMutex
	consumerSessionManagersMap map[string][]*lavasession.ConsumerSessionManager // key is chainID so we don;t run getPairing more than once per chain
	nextBlockForUpdate         uint64
	stateQuery                 *ConsumerStateQuery
	pairingUpdatables          []*PairingUpdatable
}

func NewPairingUpdater(stateQuery *ConsumerStateQuery) *PairingUpdater {
	return &PairingUpdater{consumerSessionManagersMap: map[string][]*lavasession.ConsumerSessionManager{}, stateQuery: stateQuery}
}

func (pu *PairingUpdater) RegisterPairing(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) error {
	chainID := consumerSessionManager.RPCEndpoint().ChainID
	pairingList, epoch, nextBlockForUpdate, err := pu.stateQuery.GetPairing(context.Background(), chainID, -1)
	if err != nil {
		return err
	}
	pu.updateConsummerSessionManager(ctx, pairingList, consumerSessionManager, epoch)
	if nextBlockForUpdate > pu.nextBlockForUpdate {
		// make sure we don't update twice, this updates pu.nextBlockForUpdate
		pu.Update(int64(nextBlockForUpdate))
	}
	pu.lock.Lock()
	defer pu.lock.Unlock()
	consumerSessionsManagersList, ok := pu.consumerSessionManagersMap[chainID]
	if !ok {
		pu.consumerSessionManagersMap[chainID] = []*lavasession.ConsumerSessionManager{consumerSessionManager}
		return nil
	}
	pu.consumerSessionManagersMap[chainID] = append(consumerSessionsManagersList, consumerSessionManager)
	return nil
}

func (pu *PairingUpdater) RegisterPairingUpdatable(ctx context.Context, pairingUpdatable *PairingUpdatable) error {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	_, epoch, _, err := pu.stateQuery.GetPairing(ctx, "", -1)
	if err != nil {
		return err
	}

	(*pairingUpdatable).UpdateEpoch(epoch)
	pu.pairingUpdatables = append(pu.pairingUpdatables, pairingUpdatable)
	return nil
}

func (pu *PairingUpdater) UpdaterKey() string {
	return CallbackKeyForPairingUpdate
}

func (pu *PairingUpdater) Update(latestBlock int64) {
	pu.lock.RLock()
	defer pu.lock.RUnlock()
	ctx := context.Background()

	if int64(pu.nextBlockForUpdate) > latestBlock {
		return
	}
	nextBlockForUpdateList := []uint64{}
	for chainID, consumerSessionManagerList := range pu.consumerSessionManagersMap {
		pairingList, epoch, nextBlockForUpdate, err := pu.stateQuery.GetPairing(ctx, chainID, latestBlock)
		if err != nil {
			utils.LavaFormatError("could not update pairing for chain, trying again next block", err, utils.Attribute{Key: "chain", Value: chainID})
			nextBlockForUpdateList = append(nextBlockForUpdateList, pu.nextBlockForUpdate+1)
			continue
		} else {
			nextBlockForUpdateList = append(nextBlockForUpdateList, nextBlockForUpdate)
		}
		for _, consumerSessionManager := range consumerSessionManagerList {
			// same pairing for all apiInterfaces, they pick the right endpoints from inside using our filter function
			err = pu.updateConsummerSessionManager(ctx, pairingList, consumerSessionManager, epoch)
			if err != nil {
				utils.LavaFormatError("failed updating consumer session manager", err, utils.Attribute{Key: "chainID", Value: chainID}, utils.Attribute{Key: "apiInterface", Value: consumerSessionManager.RPCEndpoint().ApiInterface}, utils.Attribute{Key: "pairingListLen", Value: len(pairingList)})
				continue
			}
		}
	}

	// get latest epoch from cache
	_, epoch, _, err := pu.stateQuery.GetPairing(ctx, "", latestBlock)
	if err != nil {
		utils.LavaFormatError("could not update pairing for updatables, trying again next block", err)
		nextBlockForUpdateList = append(nextBlockForUpdateList, pu.nextBlockForUpdate+1)
	} else {
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

func (pu *PairingUpdater) updateConsummerSessionManager(ctx context.Context, pairingList []epochstoragetypes.StakeEntry, consumerSessionManager *lavasession.ConsumerSessionManager, epoch uint64) (err error) {
	pairingListForThisCSM, err := pu.filterPairingListByEndpoint(ctx, planstypes.Geolocation(consumerSessionManager.RPCEndpoint().Geolocation), pairingList, consumerSessionManager.RPCEndpoint(), epoch)
	if err != nil {
		return err
	}
	err = consumerSessionManager.UpdateAllProviders(epoch, pairingListForThisCSM)
	return
}

func (pu *PairingUpdater) filterPairingListByEndpoint(ctx context.Context, currentGeo planstypes.Geolocation, pairingList []epochstoragetypes.StakeEntry, rpcEndpoint lavasession.RPCEndpoint, epoch uint64) (filteredList map[uint64]*lavasession.ConsumerSessionsWithProvider, err error) {
	// go over stake entries, and filter endpoints that match geolocation and api interface
	pairing := map[uint64]*lavasession.ConsumerSessionsWithProvider{}
	for providerIdx, provider := range pairingList {
		//
		// Sanity
		providerEndpoints := provider.GetEndpoints()
		if len(providerEndpoints) == 0 {
			utils.LavaFormatError("skipping provider with no endoints", nil, utils.Attribute{Key: "Address", Value: provider.Address}, utils.Attribute{Key: "ChainID", Value: provider.Chain})
			continue
		}

		relevantEndpoints := []epochstoragetypes.Endpoint{}
		for _, endpoint := range providerEndpoints {
			// only take into account endpoints that use the same api interface and the same geolocation
			for _, endpointApiInterface := range endpoint.ApiInterfaces {
				if endpointApiInterface == rpcEndpoint.ApiInterface { // we take all geolocations provided by the chain. the provider optimizer will prioritize the relevant ones
					relevantEndpoints = append(relevantEndpoints, endpoint)
					break
				}
			}
		}
		if len(relevantEndpoints) == 0 {
			utils.LavaFormatError("skipping provider, No relevant endpoints for apiInterface", nil, utils.Attribute{Key: "Address", Value: provider.Address}, utils.Attribute{Key: "ChainID", Value: provider.Chain}, utils.Attribute{Key: "apiInterface", Value: rpcEndpoint.ApiInterface}, utils.Attribute{Key: "Endpoints", Value: providerEndpoints})
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

			endp := &lavasession.Endpoint{Geolocation: planstypes.Geolocation(relevantEndpoint.Geolocation), NetworkAddress: relevantEndpoint.IPPORT, Enabled: true, Client: nil, ConnectionRefusals: 0, Addons: addons, Extensions: extensions}
			pairingEndpoints[idx] = endp
		}
		lavasession.SortByGeolocations(pairingEndpoints, currentGeo)
		pairing[uint64(providerIdx)] = lavasession.NewConsumerSessionWithProvider(
			provider.Address,
			pairingEndpoints,
			maxCu,
			epoch,
			provider.Stake,
		)
	}
	if len(pairing) == 0 {
		return nil, utils.LavaFormatError("Failed getting pairing for consumer, pairing is empty", err, utils.Attribute{Key: "apiInterface", Value: rpcEndpoint.ApiInterface}, utils.Attribute{Key: "ChainID", Value: rpcEndpoint.ChainID}, utils.Attribute{Key: "geolocation", Value: rpcEndpoint.Geolocation})
	}
	// replace previous pairing with new providers
	return pairing, nil
}
