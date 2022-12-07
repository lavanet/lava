package cosmos_thirdparty

import (
	"context"
	"encoding/json"

	pb_pkg "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	"github.com/lavanet/lava/utils"
)

type implementedCosmosBaseTendermintV1beta1 struct {
	pb_pkg.UnimplementedServiceServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedCosmosBaseTendermintV1beta1

func (is *implementedCosmosBaseTendermintV1beta1) GetLatestBlock(ctx context.Context, req *pb_pkg.GetLatestBlockRequest) (*pb_pkg.GetLatestBlockResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetLatestBlock", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetLatestBlockResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBaseTendermintV1beta1) GetBlockByHeight(ctx context.Context, req *pb_pkg.GetBlockByHeightRequest) (*pb_pkg.GetBlockByHeightResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetBlockByHeight", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetBlockByHeightResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBaseTendermintV1beta1) GetLatestValidatorSet(ctx context.Context, req *pb_pkg.GetLatestValidatorSetRequest) (*pb_pkg.GetLatestValidatorSetResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetLatestValidatorSet", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetLatestValidatorSetResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBaseTendermintV1beta1) GetNodeInfo(ctx context.Context, req *pb_pkg.GetNodeInfoRequest) (*pb_pkg.GetNodeInfoResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetNodeInfo", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetNodeInfoResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBaseTendermintV1beta1) GetSyncing(ctx context.Context, req *pb_pkg.GetSyncingRequest) (*pb_pkg.GetSyncingResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetSyncing", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetSyncingResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBaseTendermintV1beta1) GetValidatorSetByHeight(ctx context.Context, req *pb_pkg.GetValidatorSetByHeightRequest) (*pb_pkg.GetValidatorSetByHeightResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.base.tendermint.v1beta1.Service.GetValidatorSetByHeight", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetValidatorSetByHeightResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
