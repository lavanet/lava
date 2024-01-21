package badgeserver

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc/metadata"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const dummyApiInterface = "badgeApiInterface"

type Server struct {
	pairingtypes.UnimplementedBadgeGeneratorServer
	ProjectsConfiguration GelocationToProjectsConfiguration // geolocation/project_id/project_data
	epoch                 uint64
	chainFetcher          *chainlib.LavaChainFetcher
	ChainId               string
	IpService             *IpService
	metrics               *MetricsService
	stateTracker          *BadgeStateTracker
	specs                 map[string]spectypes.Spec // holding the specs for all chains
	specLock              sync.RWMutex
	clientCtx             client.Context
	projectPublicKey      string
	projectPrivateKey     *btcSecp256k1.PrivateKey
}

func NewServer(ipService *IpService, chainId string, projectsData GelocationToProjectsConfiguration, chainFetcher *chainlib.LavaChainFetcher, clientCtx client.Context, projectPublicKey string, projectPrivateKey *btcSecp256k1.PrivateKey) (*Server, error) {
	server := &Server{
		ProjectsConfiguration: GelocationToProjectsConfiguration{},
		ChainId:               chainId,
		IpService:             ipService,
		specs:                 map[string]spectypes.Spec{},
		chainFetcher:          chainFetcher,
		clientCtx:             clientCtx,
		projectPublicKey:      projectPublicKey,
		projectPrivateKey:     projectPrivateKey,
	}

	server.ProjectsConfiguration = projectsData
	server.metrics = InitMetrics()
	return server, nil
}

func (s *Server) GetUniqueName() string {
	return "badge_server"
}

func (s *Server) InitializeStateTracker(tracker *BadgeStateTracker) {
	if s.stateTracker != nil {
		utils.LavaFormatFatal("state tracker already initialized", nil)
	}
	s.stateTracker = tracker
}

func (s *Server) SetSpec(specUpdate spectypes.Spec) {
	s.specLock.Lock()
	defer s.specLock.Unlock()
	s.specs[specUpdate.Index] = specUpdate
}

func (s *Server) UpdateEpoch(epoch uint64) {
	utils.LavaFormatDebug("Got epoch update", utils.Attribute{Key: "epoch", Value: epoch})
	atomic.StoreUint64(&s.epoch, epoch)
}

func (s *Server) GetEpoch() uint64 {
	return atomic.LoadUint64(&s.epoch)
}

func (s *Server) checkSpecExists(specID string) (spectypes.Spec, bool) {
	s.specLock.RLock()
	defer s.specLock.RUnlock()
	spec, found := s.specs[specID]
	return spec, found
}

func (s *Server) getSpec(ctx context.Context, specId string) (spectypes.Spec, error) {
	_, found := s.checkSpecExists(specId)
	if !found {
		err := s.stateTracker.RegisterForSpecUpdates(ctx, s, lavasession.RPCEndpoint{ChainID: specId, ApiInterface: dummyApiInterface})
		if err != nil {
			return spectypes.Spec{}, utils.LavaFormatError("BadgeServer Failed registering for spec updates", err)
		}
	}
	// we should have the spec now after fetching it from the chain. if we don't have it badge server failed getting the spec
	spec, found := s.checkSpecExists(specId)
	if !found {
		return spectypes.Spec{}, utils.LavaFormatError("Failed fetching spec without getting error, shouldn't get here", nil)
	}
	return spec, nil
}

func (s *Server) Active() bool {
	return true
}

func (s *Server) GenerateBadge(ctx context.Context, req *pairingtypes.GenerateBadgeRequest) (*pairingtypes.GenerateBadgeResponse, error) {
	spec, err := s.getSpec(ctx, req.SpecId)
	if err != nil {
		return nil, utils.LavaFormatError("badge server failed fetching spec", err)
	}
	metadata, _ := metadata.FromIncomingContext(ctx)
	clientAddress := metadata.Get(RefererHeaderKey)
	ipAddress := ""
	if len(clientAddress) > 0 {
		ipAddress = clientAddress[0]
	}
	projectData, err := s.validateRequestAndGetProjectData(ipAddress, req)
	if err != nil {
		s.metrics.AddRequest(false)
		return nil, err
	}
	badge := pairingtypes.Badge{
		CuAllocation: uint64(projectData.EpochsMaxCu),
		Epoch:        s.GetEpoch(),
		Address:      req.BadgeAddress,
		LavaChainId:  s.ChainId,
		VirtualEpoch: s.stateTracker.GetLatestVirtualEpoch(),
	}

	result := pairingtypes.GenerateBadgeResponse{
		Badge:              &badge,
		BadgeSignerAddress: s.projectPublicKey,
		Spec:               &spec,
	}

	err = s.addPairingListToResponse(ctx, req, projectData, &result)
	if err != nil {
		s.metrics.AddRequest(false)
		return nil, err
	}

	err = signTheResponse(s.projectPrivateKey, &result)
	if err != nil {
		s.metrics.AddRequest(false)
		return nil, err
	}
	s.metrics.AddRequest(true)
	return &result, nil
}

func (s *Server) validateRequestAndGetProjectData(clientAddress string, in *pairingtypes.GenerateBadgeRequest) (*ProjectConfiguration, error) {
	if in == nil {
		return nil, utils.LavaFormatError("Validation failed", fmt.Errorf("invalid request, no input data provided"))
	}

	if in.BadgeAddress == "" || in.ProjectId == "" {
		fmt.Println("In: ", in)
		return nil, utils.LavaFormatError("Validation failed", fmt.Errorf("bad request, no valid input data provided"))
	}

	projectId := strings.ToLower(in.ProjectId)
	geolocation := s.getClientGeolocationOrDefault(clientAddress)
	geolocationData, exist := s.ProjectsConfiguration[geolocation]
	if !exist {
		return nil, utils.LavaFormatError(
			"invalid configuration",
			fmt.Errorf("invalid configuration for this geolocation"),
			utils.LogAttr("BadgeAddress", in.BadgeAddress),
			utils.LogAttr("ProjectId", projectId),
			utils.LogAttr("geolocation", geolocation),
			utils.LogAttr("ip", clientAddress),
		)
	}

	projectData, exist := geolocationData[projectId]
	if !exist {
		projectData, exist = geolocationData[DefaultProjectId]
		if !exist {
			return nil, utils.LavaFormatError(
				"Validation failed",
				fmt.Errorf("default project not found"),
				utils.LogAttr("BadgeAddress", in.BadgeAddress),
				utils.LogAttr("ProjectId", projectId),
				utils.LogAttr("geolocation", geolocation),
			)
		}
	}
	return projectData, nil
}

func (s *Server) getClientGeolocationOrDefault(clientIpAddress string) string {
	if s.IpService != nil && len(clientIpAddress) > 0 {
		utils.LavaFormatDebug("searching for ip", utils.Attribute{
			Key:   "clientIp",
			Value: clientIpAddress,
		})
		ip, err := s.IpService.SearchForIp(clientIpAddress)
		if err != nil {
			utils.LavaFormatError("error searching for client ip-geolocation", err)
		} else if ip == nil {
			utils.LavaFormatInfo("ip not found")
		} else {
			return fmt.Sprintf("%d", ip.Geolocation)
		}
	} else {
		utils.LavaFormatInfo("Ip service not configured correctly")
	}
	return fmt.Sprintf("%d", s.IpService.DefaultGeolocation)
}

func (s *Server) addPairingListToResponse(ctx context.Context, request *pairingtypes.GenerateBadgeRequest,
	configurations *ProjectConfiguration, response *pairingtypes.GenerateBadgeResponse) error {
	chainID := request.SpecId
	if chainID == "" {
		return nil
	}

	if configurations.PairingList == nil {
		configurations.PairingList = make(map[string]*pairingtypes.QueryGetPairingResponse)
	}

	if configurations.UpdatedEpoch == nil {
		configurations.UpdatedEpoch = make(map[string]uint64)
	}
	if configurations.PairingList[chainID] == nil || response.Badge.Epoch != configurations.UpdatedEpoch[chainID] {
		querier := pairingtypes.NewQueryClient(s.clientCtx)
		getPairingResponse, err := querier.GetPairing(ctx, &pairingtypes.QueryGetPairingRequest{
			ChainID: chainID,
			Client:  s.projectPublicKey,
		})

		if err != nil {
			return utils.LavaFormatError("Failed to get pairings", err,
				utils.Attribute{Key: "epoch", Value: s.GetEpoch()},
				utils.Attribute{Key: "BadgeAddress", Value: request.GetBadgeAddress()},
				utils.Attribute{Key: "ProjectId", Value: request.ProjectId})
		}
		configurations.PairingList[chainID] = getPairingResponse
		configurations.UpdatedEpoch[chainID] = response.Badge.Epoch
	}
	response.GetPairingResponse = configurations.PairingList[chainID]
	return nil
}

// note this update the signature of the response
func signTheResponse(privateKey *btcSecp256k1.PrivateKey, response *pairingtypes.GenerateBadgeResponse) error {
	signature, err := sigs.Sign(privateKey, *response.Badge)
	if err != nil {
		return err
	}

	response.Badge.ProjectSig = signature
	return nil
}
