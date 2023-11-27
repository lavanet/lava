package rpcprovider

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	SpecValidationIntervalSecFlagName = "spec-validation-interval-sec"
)

var SpecValidationIntervalSec = uint64(10800) // 3 Hours

type SpecValidator struct {
	lock sync.RWMutex

	chainFetchers     map[string][]*chainlib.ChainFetcherIf // key is chainId
	providerListeners map[string]*ProviderListener          // key is address
}

func NewSpecValidator() *SpecValidator {
	return &SpecValidator{
		lock:              sync.RWMutex{},
		chainFetchers:     make(map[string][]*chainlib.ChainFetcherIf),
		providerListeners: make(map[string]*ProviderListener),
	}
}

func (sv *SpecValidator) GetUniqueName() string {
	return "spec_validator"
}

func (sv *SpecValidator) Start(ctx context.Context) {
	go sv.validateAllChainsLoop(ctx)
}

func (sv *SpecValidator) validateAllChainsLoop(ctx context.Context) {
	timerInterval := time.Duration(SpecValidationIntervalSec) * time.Second
	ticker := time.NewTicker(timerInterval)
	for {
		select {
		case <-ticker.C:
			func() {
				sv.lock.Lock()
				defer sv.lock.Unlock()
				sv.validateAllChains(ctx)
			}()
		case <-ctx.Done():
			ticker.Stop()
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

func (sv *SpecValidator) SetSpec(spec spectypes.Spec) {
	sv.lock.Lock()
	defer sv.lock.Unlock()

	chainId := spec.Index
	if _, found := sv.chainFetchers[chainId]; !found {
		utils.LavaFormatError("Could not find chainFetchers with given chainId", nil, utils.Attribute{Key: "chainId", Value: chainId})
		return
	}
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

func (sv *SpecValidator) validateChain(ctx context.Context, chainId string) error {
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
		return utils.LavaFormatError("Validation chainId failed with errors", nil,
			utils.Attribute{Key: "chainId", Value: chainId},
			utils.Attribute{Key: "errors", Value: errors},
		)
	}
	return nil
}
