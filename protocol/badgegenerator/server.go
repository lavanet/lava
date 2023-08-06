package badgegenerator

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"google.golang.org/grpc/metadata"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/badgegenerator/grpc"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type Server struct {
	pairingtypes.UnimplementedBadgeGeneratorServer
	ProjectsConfiguration map[string]map[string]*ProjectConfiguration // geolocation/project_id/project_data
	epoch                 uint64
	grpcFetcher           *grpc.GRPCFetcher
	ChainId               string
	IpService             *IpService
	metrics               *MetricsService
}

func NewServer(ipService *IpService, grpcUrl string, chainId string, userData string) (*Server, error) {
	server := &Server{
		ProjectsConfiguration: map[string]map[string]*ProjectConfiguration{},
		ChainId:               chainId,
		IpService:             ipService,
	}

	if userData != "" {
		projectsData := make(map[string]map[string]*ProjectConfiguration)
		err := json.Unmarshal([]byte(userData), &projectsData)
		if err != nil {
			return nil, err
		}
		server.ProjectsConfiguration = projectsData
	}
	grpcFetch, err := grpc.NewGRPCFetcher(grpcUrl)
	if err != nil {
		return nil, err
	}
	server.grpcFetcher = grpcFetch
	server.metrics = InitMetrics()
	return server, nil
}

func (s *Server) UpdateEpoch(epoch uint64) {
	utils.LavaFormatDebug("Got epoch update", utils.Attribute{Key: "epoch", Value: epoch})
	atomic.StoreUint64(&s.epoch, epoch)
}

func (s *Server) GetEpoch() uint64 {
	return atomic.LoadUint64(&s.epoch)
}

func (s *Server) GenerateBadge(ctx context.Context, req *pairingtypes.GenerateBadgeRequest) (*pairingtypes.GenerateBadgeResponse, error) {
	metadata, _ := metadata.FromIncomingContext(ctx)
	clientAddress := metadata.Get(RefererHeaderKey)
	ipAddress := ""
	if len(clientAddress) > 0 {
		ipAddress = clientAddress[0]
	}
	projectData, err := s.validateRequest(ipAddress, req)
	if err != nil {
		s.metrics.AddRequest(false)
		return nil, err
	}
	badge := pairingtypes.Badge{
		CuAllocation: uint64(projectData.EpochsMaxCu),
		Epoch:        s.GetEpoch(),
		Address:      req.BadgeAddress,
		LavaChainId:  s.ChainId,
	}
	result := pairingtypes.GenerateBadgeResponse{
		Badge:              &badge,
		BadgeSignerAddress: projectData.ProjectPublicKey,
		PairingList:        make([]*epochstoragetypes.StakeEntry, 0),
	}

	err = s.addPairingListToResponse(req, projectData, &result)
	if err != nil {
		s.metrics.AddRequest(false)
		return nil, err
	}

	err = signTheResponse(projectData.ProjectPrivateKey, &result)
	if err != nil {
		s.metrics.AddRequest(false)
		return nil, err
	}
	s.metrics.AddRequest(true)
	return &result, nil
}

func (s *Server) validateRequest(clientAddress string, in *pairingtypes.GenerateBadgeRequest) (*ProjectConfiguration, error) {
	if in == nil {
		err := fmt.Errorf("invalid request, no input data provided")
		utils.LavaFormatError("Validation failed", err)
		return nil, err
	}
	if in.BadgeAddress == "" || in.ProjectId == "" {
		err := fmt.Errorf("bad request, no valid input data provided")
		utils.LavaFormatError("Validation failed", err)
		return nil, err
	}
	geolocation := s.getClientGeolocationOrDefault(clientAddress)
	geolocationData, exist := s.ProjectsConfiguration[geolocation]
	if !exist {
		err := fmt.Errorf("invalid configuration for this geolocation")
		utils.LavaFormatError(
			"invalid configuration",
			err,
			utils.Attribute{
				Key:   "BadgeAddress",
				Value: in.BadgeAddress,
			}, utils.Attribute{
				Key:   "ProjectId",
				Value: in.ProjectId,
			},
			utils.Attribute{
				Key:   "geolocation",
				Value: geolocation,
			},
			utils.Attribute{
				Key:   "ip",
				Value: clientAddress,
			},
		)
		return nil, err
	}
	projectData, exist := geolocationData[in.ProjectId]
	if !exist {
		projectData, exist = geolocationData[DefaultProjectId]
		if !exist {
			err := fmt.Errorf("default project not found")
			utils.LavaFormatError(
				"Validation failed",
				err,
				utils.Attribute{
					Key:   "BadgeAddress",
					Value: in.BadgeAddress,
				}, utils.Attribute{
					Key:   "ProjectId",
					Value: in.ProjectId,
				},
				utils.Attribute{
					Key:   "geolocation",
					Value: geolocation,
				},
			)
			return nil, err
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

func (s *Server) addPairingListToResponse(request *pairingtypes.GenerateBadgeRequest, configurations *ProjectConfiguration, response *pairingtypes.GenerateBadgeResponse) error {
	if request.SpecId != "" {
		if configurations.PairingList == nil || response.Badge.Epoch != configurations.UpdatedEpoch {
			pairings, _, err := s.grpcFetcher.FetchPairings(request.SpecId, configurations.ProjectPublicKey)
			if err != nil {
				utils.LavaFormatError("Failed to get pairings", err,
					utils.Attribute{Key: "epoch", Value: s.GetEpoch()},
					utils.Attribute{Key: "BadgeAddress", Value: request.GetBadgeAddress()},
					utils.Attribute{Key: "ProjectId", Value: request.ProjectId})
				return err
			}
			configurations.PairingList = pairings
			configurations.UpdatedEpoch = response.Badge.Epoch
		}

		for _, entry := range *configurations.PairingList {
			marshalled, _ := entry.Marshal()
			newEntry := &epochstoragetypes.StakeEntry{}
			newEntry.Unmarshal(marshalled)
			response.PairingList = append(response.PairingList, newEntry)
		}
	}
	return nil
}

// note this update the signature of the response
func signTheResponse(privateKeyString string, response *pairingtypes.GenerateBadgeResponse) error {
	privateKeyBytes, _ := hex.DecodeString(privateKeyString)
	privateKey, _ := btcSecp256k1.PrivKeyFromBytes(btcSecp256k1.S256(), privateKeyBytes)
	signature, err := sigs.Sign(privateKey, *response.Badge)
	if err != nil {
		return err
	}

	response.Badge.ProjectSig = signature
	return nil
}
