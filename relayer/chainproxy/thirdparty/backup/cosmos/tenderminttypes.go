package cosmos_thirdparty

import (
	"context"
	"encoding/json"

	tm "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	"github.com/lavanet/lava/utils"
	"google.golang.org/protobuf/proto"
)

type implementedQueryServer struct {
	tm.UnimplementedServiceServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

func (qs *implementedQueryServer) GetNodeInfo(ctx context.Context, req *tm.GetNodeInfoRequest) (*tm.GetNodeInfoResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetNodeInfo", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &tm.GetNodeInfoResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
func (qs *implementedQueryServer) GetSyncing(ctx context.Context, req *tm.GetSyncingRequest) (*tm.GetSyncingResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetSyncing", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &tm.GetSyncingResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
func (qs *implementedQueryServer) GetLatestBlock(ctx context.Context, req *tm.GetLatestBlockRequest) (*tm.GetLatestBlockResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetLatestBlock", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &tm.GetLatestBlockResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
func (qs *implementedQueryServer) GetBlockByHeight(ctx context.Context, req *tm.GetBlockByHeightRequest) (*tm.GetBlockByHeightResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetBlockByHeight", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &tm.GetBlockByHeightResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
func (qs *implementedQueryServer) GetLatestValidatorSet(ctx context.Context, req *tm.GetLatestValidatorSetRequest) (*tm.GetLatestValidatorSetResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetLatestValidatorSet", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &tm.GetLatestValidatorSetResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
func (qs *implementedQueryServer) GetValidatorSetByHeight(ctx context.Context, req *tm.GetValidatorSetByHeightRequest) (*tm.GetValidatorSetByHeightResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetValidatorSetByHeight", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &tm.GetValidatorSetByHeightResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
func (qs *implementedQueryServer) ABCIQuery(ctx context.Context, req *tm.ABCIQueryRequest) (*tm.ABCIQueryResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.ABCIQuery", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &tm.ABCIQueryResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
