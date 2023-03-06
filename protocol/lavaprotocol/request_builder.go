package lavaprotocol

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	SupportedNumberOfVRFs = 2
)

type RelayRequestCommonData struct {
	ChainID        string `protobuf:"bytes,1,opt,name=chainID,proto3" json:"chainID,omitempty"`
	ConnectionType string `protobuf:"bytes,2,opt,name=connection_type,json=connectionType,proto3" json:"connection_type,omitempty"`
	ApiUrl         string `protobuf:"bytes,3,opt,name=api_url,json=apiUrl,proto3" json:"api_url,omitempty"`
	Data           []byte `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
	RequestBlock   int64  `protobuf:"varint,5,opt,name=request_block,json=requestBlock,proto3" json:"request_block,omitempty"`
	ApiInterface   string `protobuf:"bytes,6,opt,name=apiInterface,proto3" json:"apiInterface,omitempty"`
}

type RelayResult struct {
	Request         *pairingtypes.RelayRequest
	Reply           *pairingtypes.RelayReply
	ProviderAddress string
	ReplyServer     *pairingtypes.Relayer_RelaySubscribeClient
	Finalized       bool
}

func NewRelayRequestCommonData(chainID string, connectionType string, apiUrl string, data []byte, requestBlock int64, apiInterface string) RelayRequestCommonData {
	return RelayRequestCommonData{
		ChainID:        chainID,
		ConnectionType: connectionType,
		ApiUrl:         apiUrl,
		Data:           data,
		RequestBlock:   requestBlock,
		ApiInterface:   apiInterface,
	}
}

func ConstructRelayRequest(ctx context.Context, privKey *btcec.PrivateKey, chainID string, relayRequestCommonData RelayRequestCommonData, providerPublicAddress string, consumerSession *lavasession.SingleConsumerSession, epoch int64, reportedProviders []byte) (*pairingtypes.RelayRequest, error) {
	relayRequest := &pairingtypes.RelayRequest{
		Provider:              providerPublicAddress,
		ConnectionType:        relayRequestCommonData.ConnectionType,
		ApiUrl:                relayRequestCommonData.ApiUrl,
		Data:                  relayRequestCommonData.Data,
		SessionId:             uint64(consumerSession.SessionId),
		ChainID:               chainID,
		CuSum:                 consumerSession.CuSum + consumerSession.LatestRelayCu, // add the latestRelayCu which will be applied when session is returned properly
		BlockHeight:           epoch,
		RelayNum:              consumerSession.RelayNum + lavasession.RelayNumberIncrement, // increment the relay number. which will be applied when session is returned properly
		RequestBlock:          relayRequestCommonData.RequestBlock,
		QoSReport:             consumerSession.QoSInfo.LastQoSReport,
		DataReliability:       nil,
		UnresponsiveProviders: reportedProviders,
		ApiInterface:          relayRequestCommonData.ApiInterface,
	}
	sig, err := sigs.SignRelay(privKey, *relayRequest)
	if err != nil {
		return nil, err
	}
	relayRequest.Sig = sig
	return relayRequest, nil
}

func GetTimePerCu(cu uint64) time.Duration {
	return chainlib.LocalNodeTimePerCu(cu) + chainlib.MinimumTimePerRelayDelay
}

func UpdateRequestedBlock(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply) {
	// since sometimes the user is sending requested block that is a magic like latest, or earliest we need to specify to the reliability what it is
	request.RequestBlock = ReplaceRequestedBlock(request.RequestBlock, response.LatestBlock)
}

func ReplaceRequestedBlock(requestedBlock int64, latestBlock int64) int64 {
	switch requestedBlock {
	case spectypes.LATEST_BLOCK:
		return latestBlock
	case spectypes.SAFE_BLOCK:
		return latestBlock
	case spectypes.FINALIZED_BLOCK:
		return latestBlock
	case spectypes.EARLIEST_BLOCK:
		return spectypes.NOT_APPLICABLE // TODO: add support for earliest block reliability
	}
	return requestedBlock
}

func DataReliabilityThresholdToSession(vrfs [][]byte, uniqueIdentifiers []bool, reliabilityThreshold uint32, originalProvidersCount uint32) (indexes map[int64]bool) {
	// check for the VRF thresholds and if holds true send a relay to the provider
	// TODO: improve with blocklisted address, and the module-1
	indexes = make(map[int64]bool, len(vrfs))
	for vrfIndex, vrf := range vrfs {
		index, err := utils.GetIndexForVrf(vrf, originalProvidersCount, reliabilityThreshold)
		if index == -1 || err != nil {
			continue // no reliability this time.
		}
		if _, ok := indexes[index]; !ok {
			indexes[index] = uniqueIdentifiers[vrfIndex]
		}
	}
	return
}

func NewVRFData(differentiator bool, vrf_res []byte, vrf_proof []byte, request *pairingtypes.RelayRequest, reply *pairingtypes.RelayReply) *pairingtypes.VRFData {
	dataReliability := &pairingtypes.VRFData{
		Differentiator: differentiator,
		VrfValue:       vrf_res,
		VrfProof:       vrf_proof,
		ProviderSig:    reply.Sig,
		AllDataHash:    sigs.AllDataHash(reply, request),
		QueryHash:      utils.CalculateQueryHash(*request),
		Sig:            nil,
	}
	return dataReliability
}

func ConstructDataReliabilityRelayRequest(ctx context.Context, vrfData *pairingtypes.VRFData, privKey *btcec.PrivateKey, chainID string, relayRequestCommonData *RelayRequestCommonData, providerPublicAddress string, epoch int64, reportedProviders []byte) (*pairingtypes.RelayRequest, error) {
	if relayRequestCommonData.RequestBlock < 0 {
		return nil, utils.LavaFormatError("tried to construct data reliability relay with invalid request block, need to specify exactly what block is required", nil,
			&map[string]string{"requested_common_data": fmt.Sprintf("%+v", relayRequestCommonData), "epoch": strconv.FormatInt(epoch, 10), "chainID": chainID})
	}
	relayRequest := &pairingtypes.RelayRequest{
		Provider:              providerPublicAddress,
		ConnectionType:        relayRequestCommonData.ConnectionType,
		ApiUrl:                relayRequestCommonData.ApiUrl,
		Data:                  relayRequestCommonData.Data,
		SessionId:             lavasession.DataReliabilitySessionId, // sessionID for reliability is 0
		ChainID:               chainID,
		CuSum:                 lavasession.DataReliabilityCuSum, // consumerSession.CuSum == 0
		BlockHeight:           epoch,
		RelayNum:              0, // consumerSession.RelayNum == 0
		RequestBlock:          relayRequestCommonData.RequestBlock,
		QoSReport:             nil,
		DataReliability:       vrfData,
		UnresponsiveProviders: reportedProviders,
	}
	sig, err := sigs.SignRelay(privKey, *relayRequest)
	if err != nil {
		return nil, err
	}
	relayRequest.Sig = sig

	sig, err = sigs.SignVRFData(privKey, relayRequest.DataReliability)
	if err != nil {
		return nil, err
	}
	relayRequest.DataReliability.Sig = sig
	return relayRequest, nil
}

func VerifyReliabilityResults(originalResult *RelayResult, dataReliabilityResults []*RelayResult, totalNumberOfSessions int) (conflict bool, conflicts []*conflicttypes.ResponseConflict) {
	verificationsLength := len(dataReliabilityResults)
	participatingProviders := make(map[string]string, verificationsLength+1)
	participatingProviders["originalAddress"] = originalResult.ProviderAddress
	for idx, dataReliabilityResult := range dataReliabilityResults {
		add := dataReliabilityResult.ProviderAddress
		participatingProviders["address"+strconv.Itoa(idx)] = add
		conflict_now, detectionMessage := compareRelaysFindConflict(originalResult, dataReliabilityResult)
		if conflict_now {
			conflicts = []*conflicttypes.ResponseConflict{detectionMessage}
			conflict = true
		}
	}
	if conflict {
		// CompareRelaysAndReportConflict to each one of the data reliability relays to confirm that the first relay was'nt ok
		for idx1 := 0; idx1 < verificationsLength; idx1++ {
			for idx2 := (idx1 + 1); idx2 < verificationsLength; idx2++ {
				conflict_responses, moreDetectionMessages := compareRelaysFindConflict(dataReliabilityResults[idx1], dataReliabilityResults[idx2])
				if conflict_responses {
					conflicts = append(conflicts, moreDetectionMessages)
				}
			}
		}
	}

	if !conflict && totalNumberOfSessions == verificationsLength { // if no conflict was detected data reliability was successful
		// all reliability sessions succeeded
		utils.LavaFormatInfo("Reliability verified successfully!", &participatingProviders)
	} else {
		utils.LavaFormatInfo("Data is not Reliability verified!", &participatingProviders)
	}
	return conflict, conflicts
}

func compareRelaysFindConflict(result1 *RelayResult, result2 *RelayResult) (conflict bool, responseConflict *conflicttypes.ResponseConflict) {
	compare_result := bytes.Compare(result1.Reply.Data, result2.Reply.Data)
	if compare_result == 0 {
		// they have equal data
		return false, nil
	}
	// they have different data! report!
	utils.LavaFormatWarning("Simulation: DataReliability detected mismatching results, Reporting...", nil, &map[string]string{"Data0": string(result1.Reply.Data), "Data1": string(result2.Reply.Data)})
	responseConflict = &conflicttypes.ResponseConflict{
		ConflictRelayData0: &conflicttypes.ConflictRelayData{Reply: result1.Reply, Request: result1.Request},
		ConflictRelayData1: &conflicttypes.ConflictRelayData{Reply: result2.Reply, Request: result2.Request},
	}
	return
}
