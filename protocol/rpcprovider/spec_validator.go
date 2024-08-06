package rpcprovider

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const (
	SpecValidationIntervalFlagName               = "spec-validation-interval"
	SpecValidationIntervalDisabledChainsFlagName = "spec-validation-interval-disabled-chains"
)

var (
	SpecValidationInterval               = 3 * time.Hour
	SpecValidationIntervalDisabledChains = 3 * time.Minute
)

type SpecValidator struct {
	lock sync.RWMutex

	chainFetchers     map[string][]*chainlib.ChainFetcherIf // key is chainId
	providerListeners map[string]*ProviderListener          // key is address
	skipValidations   map[string]struct{}                   // key is a validation name to skip
}

func NewSpecValidator() *SpecValidator {
	return &SpecValidator{
		lock:              sync.RWMutex{},
		chainFetchers:     make(map[string][]*chainlib.ChainFetcherIf),
		providerListeners: make(map[string]*ProviderListener),
		skipValidations:   make(map[string]struct{}),
	}
}

func (sv *SpecValidator) GetUniqueName() string {
	return "spec_validator"
}

func (sv *SpecValidator) Start(ctx context.Context) {
	go sv.validateAllChainsLoop(ctx)
}

func (sv *SpecValidator) validateAllChainsLoop(ctx context.Context) {
	validationTicker := time.NewTicker(SpecValidationInterval)
	validationTickerForDisabled := time.NewTicker(SpecValidationIntervalDisabledChains)
	for {
		select {
		case <-validationTicker.C:
			func() {
				sv.lock.Lock()
				defer sv.lock.Unlock()
				sv.validateAllChains(ctx)
				// we just ran validate on all chains no reason to do disabled chains in the next interval
				validationTickerForDisabled.Reset(SpecValidationIntervalDisabledChains)
			}()
		case <-validationTickerForDisabled.C:
			func() {
				sv.lock.Lock()
				defer sv.lock.Unlock()
				sv.validateAllDisabledChains(ctx)
			}()
		case <-ctx.Done():
			validationTicker.Stop()
			return
		}
	}
}

func (sv *SpecValidator) AddChainFetcher(ctx context.Context, chainFetcher *chainlib.ChainFetcherIf, chainId string) error {
	sv.lock.Lock()
	defer sv.lock.Unlock()
	err := (*chainFetcher).Validate(ctx)
	if err != nil {
		return err
	}

	if _, found := sv.chainFetchers[chainId]; !found {
		sv.chainFetchers[chainId] = []*chainlib.ChainFetcherIf{}
	}
	sv.chainFetchers[chainId] = append(sv.chainFetchers[chainId], chainFetcher)

	return nil
}

func (sv *SpecValidator) AddRPCProviderListener(address string, providerListener *ProviderListener) {
	sv.lock.Lock()
	defer sv.lock.Unlock()
	sv.providerListeners[address] = providerListener
}

func (sv *SpecValidator) VerifySpec(spec spectypes.Spec) {
	sv.lock.Lock()
	defer sv.lock.Unlock()

	chainId := spec.Index
	if _, found := sv.chainFetchers[chainId]; !found {
		utils.LavaFormatError("Could not find chainFetchers with given chainId", nil, utils.Attribute{Key: "chainId", Value: chainId})
		return
	}
	utils.LavaFormatDebug("Running spec verification for chainId", utils.LogAttr("chainId", chainId))
	sv.validateChain(context.Background(), chainId)
}

func (sv *SpecValidator) Active() bool {
	return true
}

func (sv *SpecValidator) getRpcProviderEndpointFromChainFetcher(chainFetcher *chainlib.ChainFetcherIf) *lavasession.RPCEndpoint {
	endpoint := (*chainFetcher).FetchEndpoint()
	return &lavasession.RPCEndpoint{
		NetworkAddress: endpoint.NetworkAddress.Address,
		ChainID:        endpoint.ChainID,
		ApiInterface:   endpoint.ApiInterface,
		Geolocation:    endpoint.Geolocation,
	}
}

func (sv *SpecValidator) validateAllChains(ctx context.Context) {
	for chainId := range sv.chainFetchers {
		sv.validateChain(ctx, chainId)
	}
}

func (sv *SpecValidator) validateAllDisabledChains(ctx context.Context) {
	for chainId := range sv.getDisabledChains() {
		sv.validateChain(ctx, chainId)
	}
}

func (sv *SpecValidator) getDisabledChains() map[string]struct{} {
	disabledChains := map[string]struct{}{}
	for _, chainFetchersList := range sv.chainFetchers {
		for _, chainFetcher := range chainFetchersList {
			rpcEndpoint := sv.getRpcProviderEndpointFromChainFetcher(chainFetcher)
			providerListener, found := sv.providerListeners[rpcEndpoint.NetworkAddress]
			if !found {
				continue
			}
			relayReceiver, found := providerListener.relayServer.relayReceivers[rpcEndpoint.Key()]
			if !found {
				continue
			}
			if !relayReceiver.enabled {
				disabledChains[rpcEndpoint.ChainID] = struct{}{}
			}
		}
	}
	return disabledChains
}

func (sv *SpecValidator) validateChain(ctx context.Context, chainId string) {
	errors := []error{}
	for _, chainFetcher := range sv.chainFetchers[chainId] {
		err := (*chainFetcher).Validate(ctx)
		rpcEndpoint := sv.getRpcProviderEndpointFromChainFetcher(chainFetcher)
		providerListener, found := sv.providerListeners[rpcEndpoint.NetworkAddress]
		if !found {
			if err != nil {
				utils.LavaFormatWarning("Verification failed for endpoint", nil, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
				errors = append(errors, err)
			}
			continue
		}

		relayReceiver, found := providerListener.relayServer.relayReceivers[rpcEndpoint.Key()]
		if !found {
			if err != nil {
				utils.LavaFormatWarning("Verification failed for endpoint", nil, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
				errors = append(errors, err)
			}
			continue
		}

		if err != nil {
			relayReceiver.enabled = false
			utils.LavaFormatError("[-] Verification failed for endpoint. Disabling endpoint.", nil, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
			errors = append(errors, err)
		} else if !relayReceiver.enabled {
			relayReceiver.enabled = true
			utils.LavaFormatError("[+] Verification passed for disabled endpoint. Enabling endpoint.", nil, utils.Attribute{Key: "endpoint", Value: rpcEndpoint})
		}
	}
	if len(errors) > 0 {
		utils.LavaFormatError("Validation chainId failed with errors", nil,
			utils.Attribute{Key: "chainId", Value: chainId},
			utils.Attribute{Key: "errors", Value: errors},
		)
	}
}
