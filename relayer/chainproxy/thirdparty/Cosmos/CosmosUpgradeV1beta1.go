
package cosmos_thirdparty

import (
	"context"
	"encoding/json"

	// add protobuf here as pb_pkg
	"github.com/lavanet/lava/utils"
	"google.golang.org/protobuf/proto"
)

type implementedCosmosUpgradeV1beta1 struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedCosmosUpgradeV1beta1



func (is *implementedCosmosUpgradeV1beta1) AppliedPlan(ctx context.Context, req *pb_pkg.QueryAppliedPlanRequest) (*pb_pkg.QueryAppliedPlanResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.upgrade.v1beta1.Query.AppliedPlan", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryAppliedPlanResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedCosmosUpgradeV1beta1) CurrentPlan(ctx context.Context, req *pb_pkg.QueryCurrentPlanRequest) (*pb_pkg.QueryCurrentPlanResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.upgrade.v1beta1.Query.CurrentPlan", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryCurrentPlanResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedCosmosUpgradeV1beta1) ModuleVersions(ctx context.Context, req *pb_pkg.QueryModuleVersionsRequest) (*pb_pkg.QueryModuleVersionsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.upgrade.v1beta1.Query.ModuleVersions", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryModuleVersionsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedCosmosUpgradeV1beta1) UpgradedConsensusState(ctx context.Context, req *pb_pkg.QueryUpgradedConsensusStateRequest) (*pb_pkg.QueryUpgradedConsensusStateResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.upgrade.v1beta1.Query.UpgradedConsensusState", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryUpgradedConsensusStateResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
