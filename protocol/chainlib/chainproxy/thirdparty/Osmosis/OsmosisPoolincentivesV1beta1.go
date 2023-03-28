package osmosis_thirdparty

import (
	"context"
	"encoding/json"

	"github.com/golang/protobuf/proto"
	pb_pkg "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/pool-incentives/types"
	"github.com/lavanet/lava/utils"
)

type implementedOsmosisPoolincentivesV1beta1 struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedOsmosisPoolincentivesV1beta1

func (is *implementedOsmosisPoolincentivesV1beta1) DistrInfo(ctx context.Context, req *pb_pkg.QueryDistrInfoRequest) (*pb_pkg.QueryDistrInfoResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "osmosis.poolincentives.v1beta1.Query/DistrInfo", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryDistrInfoResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// func (is *implementedOsmosisPoolincentivesV1beta1) ExternalIncentiveGauges(ctx context.Context, req *pb_pkg.QueryExternalIncentiveGaugesRequest) (*pb_pkg.QueryExternalIncentiveGaugesResponse, error) {
// 	reqMarshaled, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
// 	}
// 	res, err := is.cb(ctx, "osmosis.poolincentives.v1beta1.Query/ExternalIncentiveGauges", reqMarshaled)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
// 	}
// 	result := &pb_pkg.QueryExternalIncentiveGaugesResponse{}
// 	err = proto.Unmarshal(res, result)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
// 	}
// 	return result, nil
// }

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisPoolincentivesV1beta1) GaugeIds(ctx context.Context, req *pb_pkg.QueryGaugeIdsRequest) (*pb_pkg.QueryGaugeIdsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "osmosis.poolincentives.v1beta1.Query/GaugeIds", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryGaugeIdsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisPoolincentivesV1beta1) IncentivizedPools(ctx context.Context, req *pb_pkg.QueryIncentivizedPoolsRequest) (*pb_pkg.QueryIncentivizedPoolsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "osmosis.poolincentives.v1beta1.Query/IncentivizedPools", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryIncentivizedPoolsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisPoolincentivesV1beta1) LockableDurations(ctx context.Context, req *pb_pkg.QueryLockableDurationsRequest) (*pb_pkg.QueryLockableDurationsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "osmosis.poolincentives.v1beta1.Query/LockableDurations", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryLockableDurationsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisPoolincentivesV1beta1) Params(ctx context.Context, req *pb_pkg.QueryParamsRequest) (*pb_pkg.QueryParamsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "osmosis.poolincentives.v1beta1.Query/Params", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryParamsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
