package lavaprotocol

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v4/x/conflict/types"
	conflictconstruct "github.com/lavanet/lava/v4/x/conflict/types/construct"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
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

func SetSalt(requestData *pairingtypes.RelayPrivateData, value uint64) {
	nonceBytes := sigs.EncodeUint64(value)
	requestData.Salt = nonceBytes
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

	copiedQOS := copyQoSServiceReport(singleConsumerSession.QoSInfo.LastQoSReport)
	copiedExcellenceQOS := copyQoSServiceReport(singleConsumerSession.QoSInfo.LastExcellenceQoSReportRaw) // copy raw report for the node

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
		QosExcellenceReport:   copiedExcellenceQOS,
	}
}

func ConstructRelayRequest(ctx context.Context, privKey *btcec.PrivateKey, lavaChainID, chainID string, relayRequestData *pairingtypes.RelayPrivateData, providerPublicAddress string, consumerSession *lavasession.SingleConsumerSession, epoch int64, reportedProviders []*pairingtypes.ReportedProvider) (*pairingtypes.RelayRequest, error) {
	relayRequest := &pairingtypes.RelayRequest{
		RelayData:    relayRequestData,
		RelaySession: ConstructRelaySession(lavaChainID, relayRequestData, chainID, providerPublicAddress, consumerSession, epoch, reportedProviders),
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

// currently used when cache hits. we don't want DR.
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

func VerifyReliabilityResults(ctx context.Context, originalResult, dataReliabilityResult *common.RelayResult, apiCollection *spectypes.ApiCollection, headerFilterer HeaderFilterer) (conflicts *conflicttypes.ResponseConflict) {
	conflict_now, detectionMessage := compareRelaysFindConflict(ctx, *originalResult.Reply, *originalResult.Request, *dataReliabilityResult.Reply, *dataReliabilityResult.Request, apiCollection, headerFilterer)
	if conflict_now {
		return detectionMessage
	}
	utils.LavaFormatInfo("Reliability verified successfully!", utils.Attribute{Key: "GUID", Value: ctx})
	return nil
}

func compareRelaysFindConflict(ctx context.Context, reply1 pairingtypes.RelayReply, request1 pairingtypes.RelayRequest, reply2 pairingtypes.RelayReply, request2 pairingtypes.RelayRequest, apiCollection *spectypes.ApiCollection, headerFilterer HeaderFilterer) (conflict bool, responseConflict *conflicttypes.ResponseConflict) {
	// remove ignored headers so we can compare metadata and also send the signatures properly on chain
	reply1.Metadata, _, _ = headerFilterer.HandleHeaders(reply1.Metadata, apiCollection, spectypes.Header_pass_reply)
	reply2.Metadata, _, _ = headerFilterer.HandleHeaders(reply2.Metadata, apiCollection, spectypes.Header_pass_reply)
	compare_result := bytes.Compare(reply1.Data, reply2.Data)
	// TODO: compare metadata too
	if compare_result == 0 {
		// they have equal data
		return false, nil
	}

	// they have different data! report!
	utils.LavaFormatWarning("Simulation: DataReliability detected mismatching results, Reporting...", nil,
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("Request0", request1.RelayData),
		utils.LogAttr("Data0", string(reply1.Data)),
		utils.LogAttr("Request1", request2.RelayData),
		utils.LogAttr("Data1", string(reply2.Data)),
	)

	responseConflict = &conflicttypes.ResponseConflict{
		ConflictRelayData0: conflictconstruct.ConstructConflictRelayData(&reply1, &request1),
		ConflictRelayData1: conflictconstruct.ConstructConflictRelayData(&reply2, &request2),
	}
	if utils.IsTraceLogLevelEnabled() {
		firstAsString := string(reply1.Data)
		secondAsString := string(reply2.Data)
		_, idxDiff := findFirstDifferentChar(firstAsString, secondAsString)
		if idxDiff > 0 && idxDiff+100 < len(firstAsString) && idxDiff+100 < len(secondAsString) {
			utils.LavaFormatTrace("difference in responses detected",
				utils.LogAttr("index", idxDiff),
				utils.LogAttr("first_diff", firstAsString[idxDiff:idxDiff+100]),
				utils.LogAttr("second_diff", secondAsString[idxDiff:idxDiff+100]),
			)
		}
	}
	return true, responseConflict
}

func findFirstDifferentChar(str1, str2 string) (rune, int) {
	// Find the minimum length between the two strings
	minLen := len(str1)
	if len(str2) < minLen {
		minLen = len(str2)
	}

	// Iterate over the characters and find the first difference
	for i := 0; i < minLen; i++ {
		if str1[i] != str2[i] {
			return rune(str1[i]), i
		}
	}

	// If the loop completes without finding a difference,
	// return the first extra character from the longer string
	if len(str1) != len(str2) {
		if len(str1) < len(str2) {
			return rune(str2[minLen]), minLen
		}
		return rune(str1[minLen]), minLen
	}

	// Return -1 if the strings are identical
	return -1, -1
}
