package osmosis_thirdparty

import (
	"context"
	"encoding/json"

	"github.com/golang/protobuf/proto"
	pb_pkg "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/incentives/types"
	"github.com/lavanet/lava/utils"
)

type implementedOsmosisIncentives struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedOsmosisIncentives

func (is *implementedOsmosisIncentives) ActiveGauges(ctx context.Context, req *pb_pkg.ActiveGaugesRequest) (*pb_pkg.ActiveGaugesResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/ActiveGauges", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.ActiveGaugesResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisIncentives) ActiveGaugesPerDenom(ctx context.Context, req *pb_pkg.ActiveGaugesPerDenomRequest) (*pb_pkg.ActiveGaugesPerDenomResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/ActiveGaugesPerDenom", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.ActiveGaugesPerDenomResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisIncentives) GaugeByID(ctx context.Context, req *pb_pkg.GaugeByIDRequest) (*pb_pkg.GaugeByIDResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/GaugeByID", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GaugeByIDResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisIncentives) LockableDurations(ctx context.Context, req *pb_pkg.QueryLockableDurationsRequest) (*pb_pkg.QueryLockableDurationsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/LockableDurations", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryLockableDurationsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// func (is *implementedOsmosisIncentives) ModuleDistributedCoins(ctx context.Context, req *pb_pkg.ModuleDistributedCoinsRequest) (*pb_pkg.ModuleDistributedCoinsResponse, error) {
// 	reqMarshaled, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
// 	}
// 	res, err := is.cb(ctx, "osmosis.incentives.Query/ModuleDistributedCoins", reqMarshaled)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
// 	}
// 	result := &pb_pkg.ModuleDistributedCoinsResponse{}
// 	err = proto.Unmarshal(res, result)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
// 	}
// 	return result, nil
// }

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisIncentives) ModuleToDistributeCoins(ctx context.Context, req *pb_pkg.ModuleToDistributeCoinsRequest) (*pb_pkg.ModuleToDistributeCoinsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/ModuleToDistributeCoins", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.ModuleToDistributeCoinsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisIncentives) RewardsEst(ctx context.Context, req *pb_pkg.RewardsEstRequest) (*pb_pkg.RewardsEstResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/RewardsEst", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.RewardsEstResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisIncentives) UpcomingGauges(ctx context.Context, req *pb_pkg.UpcomingGaugesRequest) (*pb_pkg.UpcomingGaugesResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/UpcomingGauges", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.UpcomingGaugesResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisIncentives) UpcomingGaugesPerDenom(ctx context.Context, req *pb_pkg.UpcomingGaugesPerDenomRequest) (*pb_pkg.UpcomingGaugesPerDenomResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.incentives.Query/UpcomingGaugesPerDenom", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.UpcomingGaugesPerDenomResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
