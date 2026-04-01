package lavaprotocol

import (
	"context"

	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v5/types/relay"
)

func setSalt(requestData *pairingtypes.RelayPrivateData, value uint64) {
	requestData.Salt = sigs.EncodeUint64(value)
}

func setRequestId(requestData *pairingtypes.RelayPrivateData, value string) {
	requestData.RequestId = value
}

func setTaskId(requestData *pairingtypes.RelayPrivateData, value string) {
	requestData.XTaskId = &pairingtypes.RelayPrivateData_TaskId{
		TaskId: value,
	}
}

func setTxId(requestData *pairingtypes.RelayPrivateData, value string) {
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
	setSalt(relayData, guid)

	if reqId, found := utils.GetRequestId(ctx); found {
		setRequestId(relayData, reqId)
	}
	if taskId, found := utils.GetTaskId(ctx); found {
		setTaskId(relayData, taskId)
	}
	if txId, found := utils.GetTxId(ctx); found {
		setTxId(relayData, txId)
	}

	return relayData
}
