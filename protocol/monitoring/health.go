package monitoring

import (
	"context"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gogo/status"
	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type LavaEntity struct {
	Address string
	SpecId  string
}

type ReplyData struct {
	block   int64
	latency time.Duration
}

type SubscriptionData struct {
	FullMonthsLeft               uint64
	UsagePercentageLeftThisMonth float64
}

type HealthResults struct {
	LatestBlocks       map[string]uint64
	ProviderData       map[LavaEntity]ReplyData
	ConsumerBlocks     map[LavaEntity]uint64
	SubscriptionsData  map[string]SubscriptionData
	FrozenProviders    map[LavaEntity]struct{}
	UnhealthyProviders map[LavaEntity]string
	Specs              map[string]*spectypes.Spec
}

func RunHealth(ctx context.Context,
	clientCtx client.Context,
	subscriptionAddresses []string,
	providerAddresses []string,
	consumerEndpoints []*lavasession.RPCEndpoint,
	referenceEndpoints []*lavasession.RPCEndpoint,
	prometheusListenAddr string) (*HealthResults, error) {
	specQuerier := spectypes.NewQueryClient(clientCtx)
	healthResults := &HealthResults{
		LatestBlocks:      map[string]uint64{},
		ProviderData:      map[LavaEntity]ReplyData{},
		ConsumerBlocks:    map[LavaEntity]uint64{},
		SubscriptionsData: map[string]SubscriptionData{},
		FrozenProviders:   map[LavaEntity]struct{}{},
		Specs:             map[string]*spectypes.Spec{},
	}
	resultStatus, err := clientCtx.Client.Status(ctx)
	if err != nil {
		return nil, err
	}
	currentBlock := resultStatus.SyncInfo.LatestBlockHeight
	// get a list of all necessary specs for the test
	necessaryChains := map[string]*spectypes.Spec{}
	dualStakingQuerier := dualstakingtypes.NewQueryClient(clientCtx)
	for _, providerAddress := range providerAddresses {
		response, err := dualStakingQuerier.DelegatorProviders(ctx, &dualstakingtypes.QueryDelegatorProvidersRequest{
			Delegator:   providerAddress,
			WithPending: false,
		})
		if err != nil || response == nil {
			continue
		}
		delegations := response.GetDelegations()
		for _, delegation := range delegations {
			if delegation.Provider == providerAddress {
				necessaryChains[delegation.ChainID] = &spectypes.Spec{}
				healthResults.ProviderData[LavaEntity{
					Address: providerAddress,
					SpecId:  delegation.ChainID,
				}] = ReplyData{}
			}
		}

	}

	for _, consumerEndpoint := range consumerEndpoints {
		necessaryChains[consumerEndpoint.ChainID] = &spectypes.Spec{}
	}

	for _, referenceEndpoint := range referenceEndpoints {
		necessaryChains[referenceEndpoint.ChainID] = &spectypes.Spec{}
	}

	// populate the specs
	for specId := range necessaryChains {
		specResp, err := specQuerier.SpecRaw(ctx, &spectypes.QueryGetSpecRequest{
			ChainID: specId,
		})
		if err != nil || specResp == nil {
			return nil, err
		}
		spec := specResp.GetSpec()
		necessaryChains[specId] = &spec
	}
	healthResults.Specs = necessaryChains
	pairingQuerier := pairingtypes.NewQueryClient(clientCtx)

	stakeEntries := map[LavaEntity]epochstoragetypes.StakeEntry{}

	// get provider stake entries
	for specId := range healthResults.Specs {
		response, err := pairingQuerier.Providers(ctx, &pairingtypes.QueryProvidersRequest{
			ChainID:    specId,
			ShowFrozen: true,
		})
		if err != nil || response == nil {
			return nil, err
		}

		for _, providerEntry := range response.StakeEntry {
			providerKey := LavaEntity{
				Address: providerEntry.Address,
				SpecId:  specId,
			}
			if _, ok := healthResults.ProviderData[providerKey]; ok {
				if providerEntry.StakeAppliedBlock > uint64(currentBlock) {
					healthResults.FrozenProviders[providerKey] = struct{}{}
				} else {
					stakeEntries[providerKey] = providerEntry
				}
			}
		}
	}

	err = checkSubscriptions(ctx, clientCtx, subscriptionAddresses, healthResults)
	if err != nil {
		return nil, err
	}

	CheckProviders(ctx, clientCtx, healthResults, stakeEntries)

	return healthResults, nil
}

func checkSubscriptions(ctx context.Context, clientCtx client.Context, subscriptionAddresses []string, healthResults *HealthResults) error {
	subscriptionQuerier := subscriptiontypes.NewQueryClient(clientCtx)
	for _, subscriptionAddr := range subscriptionAddresses {
		response, err := subscriptionQuerier.Current(ctx, &subscriptiontypes.QueryCurrentRequest{
			Consumer: subscriptionAddr,
		})
		if err != nil {
			return err
		}
		healthResults.SubscriptionsData[subscriptionAddr] = SubscriptionData{
			FullMonthsLeft:               response.Sub.DurationLeft,
			UsagePercentageLeftThisMonth: float64(response.Sub.MonthCuLeft) / float64(response.Sub.MonthCuTotal),
		}
	}
	return nil
}

func CheckProviders(ctx context.Context, clientCtx client.Context, healthResults *HealthResults, providerEntries map[LavaEntity]epochstoragetypes.StakeEntry) error {
	protocolQuerier := protocoltypes.NewQueryClient(clientCtx)
	param, err := protocolQuerier.Params(ctx, &protocoltypes.QueryParamsRequest{})
	if err != nil {
		return err
	}
	lavaVersion := param.GetParams().Version
	if err != nil {
		return err
	}
	targetVersion := lvutil.ParseToSemanticVersion(lavaVersion.ProviderTarget)
	for _, providerEntry := range providerEntries {
		providerKey := LavaEntity{
			Address: providerEntry.Address,
			SpecId:  providerEntry.Chain,
		}
		for _, endpoint := range providerEntry.Endpoints {
			checkOneProvider := func(apiInterface string, addon string) (time.Duration, string, int64, error) {
				cswp := lavasession.ConsumerSessionsWithProvider{}
				relayerClientPt, conn, err := cswp.ConnectRawClientWithTimeout(ctx, endpoint.IPPORT)
				if err != nil {
					return 0, "", 0, utils.LavaFormatWarning("failed connecting to provider endpoint", err, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				defer conn.Close()
				relayerClient := *relayerClientPt
				guid := uint64(rand.Int63())
				relaySentTime := time.Now()
				probeReq := &pairingtypes.ProbeRequest{
					Guid:         guid,
					SpecId:       providerEntry.Chain,
					ApiInterface: apiInterface,
				}
				var trailer metadata.MD
				probeResp, err := relayerClient.Probe(ctx, probeReq, grpc.Trailer(&trailer))
				if err != nil {
					return 0, "", 0, utils.LavaFormatWarning("failed probing provider endpoint", err, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				versions := strings.Join(trailer.Get(common.VersionMetadataKey), ",")
				relayLatency := time.Since(relaySentTime)
				if guid != probeResp.GetGuid() {
					return 0, versions, 0, utils.LavaFormatWarning("probe returned invalid value", err, utils.Attribute{Key: "returnedGuid", Value: probeResp.GetGuid()}, utils.Attribute{Key: "guid", Value: guid}, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}

				// CORS check
				if err := rpcprovider.PerformCORSCheck(endpoint); err != nil {
					return 0, versions, 0, err
				}

				relayRequest := &pairingtypes.RelayRequest{
					RelaySession: &pairingtypes.RelaySession{SpecId: providerEntry.Chain},
					RelayData:    &pairingtypes.RelayPrivateData{ApiInterface: apiInterface, Addon: addon},
				}
				_, err = relayerClient.Relay(ctx, relayRequest)
				if err == nil {
					return 0, "", 0, utils.LavaFormatWarning("relay Without signature did not error, unexpected", nil, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				code := status.Code(err)
				if code != codes.Code(lavasession.EpochMismatchError.ABCICode()) {
					return 0, versions, 0, utils.LavaFormatWarning("relay returned unexpected error", err, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "chainID", Value: providerEntry.Chain}, utils.Attribute{Key: "network address", Value: endpoint.IPPORT})
				}
				return relayLatency, versions, probeResp.GetLatestBlock(), nil
			}
			endpointServices := endpoint.GetSupportedServices()
			if len(endpointServices) == 0 {
				utils.LavaFormatWarning("endpoint has no supported services", nil, utils.Attribute{Key: "endpoint", Value: endpoint})
			}
			for _, endpointService := range endpointServices {
				probeLatency, version, latestBlockFromProbe, err := checkOneProvider(endpointService.ApiInterface, endpointService.Addon)
				if err != nil {
					healthResults.UnhealthyProviders[providerKey] = err.Error()
					continue
				}
				parsedVer := lvutil.ParseToSemanticVersion(strings.TrimPrefix(version, "v"))
				if lvutil.IsVersionLessThan(parsedVer, targetVersion) || lvutil.IsVersionGreaterThan(parsedVer, targetVersion) {
					healthResults.UnhealthyProviders[providerKey] = "Version:" + version + " should be: " + lavaVersion.ProviderTarget
					continue
				}
				latestData := ReplyData{
					block:   latestBlockFromProbe,
					latency: probeLatency,
				}
				if existing, ok := healthResults.ProviderData[providerKey]; ok {
					latestData.block = min(existing.block, latestBlockFromProbe)
				}
				healthResults.ProviderData[providerKey] = latestData
			}
		}
	}
	return nil
}
