package lavaprotocol

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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

func GetSalt(requestData *pairingtypes.RelayPrivateData) uint64 {
	salt := requestData.Salt
	if len(salt) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(salt)
}

func SetSalt(requestData *pairingtypes.RelayPrivateData, value uint64) {
	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, value)
	requestData.Salt = nonceBytes
}

func NewRelayData(ctx context.Context, connectionType string, apiUrl string, data []byte, requestBlock int64, apiInterface string) *pairingtypes.RelayPrivateData {
	relayData := &pairingtypes.RelayPrivateData{
		ConnectionType: connectionType,
		ApiUrl:         apiUrl,
		Data:           data,
		RequestBlock:   requestBlock,
		ApiInterface:   apiInterface,
	}
	guid, found := utils.GetUniqueIdentifier(ctx)
	if !found {
		guid = utils.GenerateUniqueIdentifier()
	}
	SetSalt(relayData, guid)
	return relayData
}

func ConstructRelaySession(lavaChainID string, relayRequestData *pairingtypes.RelayPrivateData, chainID string, providerPublicAddress string, singleConsumerSession *lavasession.SingleConsumerSession, epoch int64, reportedProviders []byte) *pairingtypes.RelaySession {
	return &pairingtypes.RelaySession{
		SpecId:                chainID,
		ContentHash:           sigs.CalculateContentHashForRelayData(relayRequestData),
		SessionId:             uint64(singleConsumerSession.SessionId),
		CuSum:                 singleConsumerSession.CuSum + singleConsumerSession.LatestRelayCu, // add the latestRelayCu which will be applied when session is returned properly,
		Provider:              providerPublicAddress,
		RelayNum:              singleConsumerSession.RelayNum, // RelayNum is always incremented
		QosReport:             singleConsumerSession.QoSInfo.LastQoSReport,
		Epoch:                 epoch,
		UnresponsiveProviders: reportedProviders,
		LavaChainId:           lavaChainID,
		Sig:                   nil,
	}
}

func dataReliabilityRelaySession(lavaChainID string, relayRequestData *pairingtypes.RelayPrivateData, chainID string, providerPublicAddress string, epoch int64, relayNum uint64) *pairingtypes.RelaySession {
	return &pairingtypes.RelaySession{
		SpecId:                chainID,
		ContentHash:           sigs.CalculateContentHashForRelayData(relayRequestData),
		SessionId:             lavasession.DataReliabilitySessionId, // sessionID for reliability is 0
		CuSum:                 lavasession.DataReliabilityCuSum,     // consumerSession.CuSum == 0
		Provider:              providerPublicAddress,
		RelayNum:              relayNum,
		QosReport:             nil,
		Epoch:                 epoch,
		UnresponsiveProviders: nil,
		LavaChainId:           lavaChainID,
		Sig:                   nil,
	}
}

func ConstructRelayRequest(ctx context.Context, privKey *btcec.PrivateKey, lavaChainID string, chainID string, relayRequestData *pairingtypes.RelayPrivateData, providerPublicAddress string, consumerSession *lavasession.SingleConsumerSession, epoch int64, reportedProviders []byte) (*pairingtypes.RelayRequest, error) {
	relayRequest := &pairingtypes.RelayRequest{
		RelayData:    relayRequestData,
		RelaySession: ConstructRelaySession(lavaChainID, relayRequestData, chainID, providerPublicAddress, consumerSession, epoch, reportedProviders),
	}
	sig, err := sigs.SignRelay(privKey, *relayRequest.RelaySession)
	if err != nil {
		return nil, err
	}
	relayRequest.RelaySession.Sig = sig
	return relayRequest, nil
}

func UpdateRequestedBlock(request *pairingtypes.RelayPrivateData, response *pairingtypes.RelayReply) {
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

func ConstructDataReliabilityRelayRequest(ctx context.Context, lavaChainID string, privKey *btcec.PrivateKey, chainID string, relayRequestData *pairingtypes.RelayPrivateData, providerPublicAddress string, epoch int64, reportedProviders []byte, relayNum uint64) (*pairingtypes.RelayRequest, error) {
	if relayRequestData.RequestBlock < 0 {
		return nil, utils.LavaFormatError("tried to construct data reliability relay with invalid request block, need to specify exactly what block is required", nil,
			utils.Attribute{Key: "requested_common_data", Value: relayRequestData}, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "chainID", Value: chainID})
	}
	relayRequest := &pairingtypes.RelayRequest{
		RelayData:    relayRequestData,
		RelaySession: dataReliabilityRelaySession(lavaChainID, relayRequestData, chainID, providerPublicAddress, epoch, relayNum),
	}
	sig, err := sigs.SignRelay(privKey, *relayRequest.RelaySession)
	if err != nil {
		return nil, err
	}
	relayRequest.RelaySession.Sig = sig

	return relayRequest, nil
}

func VerifyReliabilityResults(originalResult *RelayResult, dataReliabilityResult *RelayResult) (conflicts *conflicttypes.ResponseConflict) {
	conflict_now, detectionMessage := compareRelaysFindConflict(originalResult, dataReliabilityResult)
	if conflict_now {
		utils.LavaFormatInfo("Reliability verified successfully!")
		return detectionMessage
	}
	return nil
}

func compareRelaysFindConflict(result1 *RelayResult, result2 *RelayResult) (conflict bool, responseConflict *conflicttypes.ResponseConflict) {
	compare_result := bytes.Compare(result1.Reply.Data, result2.Reply.Data)
	if compare_result == 0 {
		// they have equal data
		return false, nil
	}
	// they have different data! report!
	utils.LavaFormatWarning("Simulation: DataReliability detected mismatching results, Reporting...", nil, utils.Attribute{Key: "Data0", Value: string(result1.Reply.Data)}, utils.Attribute{Key: "Data1", Value: result2.Reply.Data})
	responseConflict = &conflicttypes.ResponseConflict{
		ConflictRelayData0: &conflicttypes.ConflictRelayData{Reply: result1.Reply, Request: result1.Request},
		ConflictRelayData1: &conflicttypes.ConflictRelayData{Reply: result2.Reply, Request: result2.Request},
	}
	return
}
