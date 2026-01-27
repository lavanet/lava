package lavaprotocol

import (
	"context"
	"encoding/binary"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

type HeaderFilterer interface {
	HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) (filtered []pairingtypes.Metadata, overwriteReqBlock string, ignoredMetadata []pairingtypes.Metadata)
}

type RelayRequestCommonData struct {
	ChainID        string `protobuf:"bytes,1,opt,name=chainID,proto3" json:"chainID,omitempty"`
	ConnectionType string `protobuf:"bytes,2,opt,name=connection_type,json=connectionType,proto3" json:"connection_type,omitempty"`
	ApiUrl         string `protobuf:"bytes,3,opt,name=api_url,json=apiUrl,proto3" json:"api_url,omitempty"`
	Data           []byte `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
	RequestBlock   int64  `protobuf:"varint,5,opt,name=request_block,json=requestBlock,proto3" json:"request_block,omitempty"`
	ApiInterface   string `protobuf:"bytes,6,opt,name=apiInterface,proto3" json:"apiInterface,omitempty"`
}

func GetSalt(requestData *pairingtypes.RelayPrivateData) uint64 {
	salt := requestData.Salt
	if len(salt) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(salt)
}

func GetRequestId(requestData *pairingtypes.RelayPrivateData) string {
	return requestData.RequestId
}

func GetTaskId(requestData *pairingtypes.RelayPrivateData) string {
	return requestData.GetTaskId()
}

func GetTxId(requestData *pairingtypes.RelayPrivateData) string {
	return requestData.GetTxId()
}

func SetSalt(requestData *pairingtypes.RelayPrivateData, value uint64) {
	nonceBytes := sigs.EncodeUint64(value)
	requestData.Salt = nonceBytes
}

func SetRequestId(requestData *pairingtypes.RelayPrivateData, value string) {
	requestData.RequestId = value
}

func SetTaskId(requestData *pairingtypes.RelayPrivateData, value string) {
	requestData.XTaskId = &pairingtypes.RelayPrivateData_TaskId{
		TaskId: value,
	}
}

func SetTxId(requestData *pairingtypes.RelayPrivateData, value string) {
	requestData.XTxId = &pairingtypes.RelayPrivateData_TxId{
		TxId: value,
	}
}

func NewRelayData(ctx context.Context, connectionType, apiUrl string, data []byte, seenBlock int64, requestBlock int64, apiInterface string, metadata []pairingtypes.Metadata, addon string, extensions []string) *pairingtypes.RelayPrivateData {
	relayData := &pairingtypes.RelayPrivateData{
		ConnectionType: connectionType,
		ApiUrl:         apiUrl,
		Data:           data,
		RequestBlock:   requestBlock,
		SeenBlock:      seenBlock,
		ApiInterface:   apiInterface,
		Metadata:       metadata,
		Addon:          addon,
		Extensions:     extensions,
	}
	guid, found := utils.GetUniqueIdentifier(ctx)
	if !found {
		guid = utils.GenerateUniqueIdentifier()
	}
	SetSalt(relayData, guid)

	if reqId, found := utils.GetRequestId(ctx); found {
		SetRequestId(relayData, reqId)
	}
	if taskId, found := utils.GetTaskId(ctx); found {
		SetTaskId(relayData, taskId)
	}
	if txId, found := utils.GetTxId(ctx); found {
		SetTxId(relayData, txId)
	}

	return relayData
}

func ConstructRelaySession(lavaChainID string, relayRequestData *pairingtypes.RelayPrivateData, chainID, providerPublicAddress string, singleConsumerSession *lavasession.SingleConsumerSession, epoch int64, reportedProviders []*pairingtypes.ReportedProvider) *pairingtypes.RelaySession {
	copyQoSServiceReport := func(reportToCopy *pairingtypes.QualityOfServiceReport) *pairingtypes.QualityOfServiceReport {
		if reportToCopy != nil {
			QOS := *reportToCopy
			return &QOS
		}
		return nil
	}

	copiedQOS := copyQoSServiceReport(singleConsumerSession.QoSManager.GetLastQoSReport(uint64(epoch), singleConsumerSession.SessionId))
	copiedReputation := copyQoSServiceReport(singleConsumerSession.QoSManager.GetLastReputationQoSReport(uint64(epoch), singleConsumerSession.SessionId)) // copy reputation report for the node

	return &pairingtypes.RelaySession{
		SpecId:                chainID,
		ContentHash:           sigs.HashMsg(relayRequestData.GetContentHashData()),
		SessionId:             uint64(singleConsumerSession.SessionId),
		CuSum:                 singleConsumerSession.CuSum + singleConsumerSession.LatestRelayCu, // add the latestRelayCu which will be applied when session is returned properly,
		Provider:              providerPublicAddress,
		RelayNum:              singleConsumerSession.RelayNum, // RelayNum is always incremented
		QosReport:             copiedQOS,
		Epoch:                 epoch,
		UnresponsiveProviders: reportedProviders,
		LavaChainId:           lavaChainID,
		Sig:                   nil,
		Badge:                 nil,
		QosExcellenceReport:   copiedReputation,
	}
}

// SkipRelaySigning when true, skips cryptographic signing of relay requests/responses.
// This is useful for static providers (smart router) where signatures are not needed
// for blockchain reward claiming, and skipping them reduces CPU and memory usage.
var SkipRelaySigning = false

func ConstructRelayRequest(ctx context.Context, privKey *btcec.PrivateKey, lavaChainID, chainID string, relayRequestData *pairingtypes.RelayPrivateData, providerPublicAddress string, consumerSession *lavasession.SingleConsumerSession, epoch int64, reportedProviders []*pairingtypes.ReportedProvider) (*pairingtypes.RelayRequest, error) {
	relayRequest := &pairingtypes.RelayRequest{
		RelayData:    relayRequestData,
		RelaySession: ConstructRelaySession(lavaChainID, relayRequestData, chainID, providerPublicAddress, consumerSession, epoch, reportedProviders),
	}
	// Skip signing when configured (e.g., smart router mode) to save CPU/memory
	if SkipRelaySigning {
		return relayRequest, nil
	}
	sig, err := sigs.Sign(privKey, *relayRequest.RelaySession)
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

// currently used when cache hits.
func SetRequestedBlockNotApplicable(request *pairingtypes.RelayPrivateData) {
	request.RequestBlock = spectypes.NOT_APPLICABLE
}

func ReplaceRequestedBlock(requestedBlock, latestBlock int64) int64 {
	switch requestedBlock {
	case spectypes.LATEST_BLOCK:
		return latestBlock
	case spectypes.SAFE_BLOCK:
		return latestBlock
	case spectypes.FINALIZED_BLOCK:
		return latestBlock
	case spectypes.PENDING_BLOCK:
		return latestBlock
	case spectypes.EARLIEST_BLOCK:
		return spectypes.NOT_APPLICABLE // TODO: add support for earliest block reliability
	}
	return requestedBlock
}

// These functions were used for conflict detection between provider responses
