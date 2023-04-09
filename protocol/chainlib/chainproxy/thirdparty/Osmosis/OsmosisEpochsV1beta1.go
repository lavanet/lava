package osmosis_thirdparty

import (
	"context"
	"encoding/json"

	pb_pkg "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/epochs/types"

	"github.com/golang/protobuf/proto"
	"github.com/lavanet/lava/utils"
)

type implementedOsmosisEpochsV1beta1 struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedOsmosisEpochsV1beta1

func (is *implementedOsmosisEpochsV1beta1) CurrentEpoch(ctx context.Context, req *pb_pkg.QueryCurrentEpochRequest) (*pb_pkg.QueryCurrentEpochResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "osmosis.epochs.v1beta1.Query/CurrentEpoch", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryCurrentEpochResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisEpochsV1beta1) EpochInfos(ctx context.Context, req *pb_pkg.QueryEpochsInfoRequest) (*pb_pkg.QueryEpochsInfoResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "osmosis.epochs.v1beta1.Query/EpochInfos", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryEpochsInfoResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
