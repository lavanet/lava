
package osmosis_thirdparty

import (
	"context"
	"encoding/json"

	// add protobuf here as pb_pkg
	"github.com/lavanet/lava/utils"
	"google.golang.org/protobuf/proto"
)

type implementedOsmosisSuperfluid struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedOsmosisSuperfluid



func (is *implementedOsmosisSuperfluid) TotalDelegationByValidatorForDenom(ctx context.Context, req *pb_pkg.QueryTotalDelegationByValidatorForDenomRequest) (*pb_pkg.QueryTotalDelegationByValidatorForDenomResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.TotalDelegationByValidatorForDenom", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryTotalDelegationByValidatorForDenomResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) AllAssets(ctx context.Context, req *pb_pkg.AllAssetsRequest) (*pb_pkg.AllAssetsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.AllAssets", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.AllAssetsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) AllIntermediaryAccounts(ctx context.Context, req *pb_pkg.AllIntermediaryAccountsRequest) (*pb_pkg.AllIntermediaryAccountsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.AllIntermediaryAccounts", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.AllIntermediaryAccountsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) AssetMultiplier(ctx context.Context, req *pb_pkg.AssetMultiplierRequest) (*pb_pkg.AssetMultiplierResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.AssetMultiplier", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.AssetMultiplierResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) AssetType(ctx context.Context, req *pb_pkg.AssetTypeRequest) (*pb_pkg.AssetTypeResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.AssetType", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.AssetTypeResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) ConnectedIntermediaryAccount(ctx context.Context, req *pb_pkg.ConnectedIntermediaryAccountRequest) (*pb_pkg.ConnectedIntermediaryAccountResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.ConnectedIntermediaryAccount", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.ConnectedIntermediaryAccountResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) EstimateSuperfluidDelegatedAmountByValidatorDenom(ctx context.Context, req *pb_pkg.EstimateSuperfluidDelegatedAmountByValidatorDenomRequest) (*pb_pkg.EstimateSuperfluidDelegatedAmountByValidatorDenomResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.EstimateSuperfluidDelegatedAmountByValidatorDenom", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.EstimateSuperfluidDelegatedAmountByValidatorDenomResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) Params(ctx context.Context, req *pb_pkg.QueryParamsRequest) (*pb_pkg.QueryParamsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.Params", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryParamsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) SuperfluidDelegationAmount(ctx context.Context, req *pb_pkg.SuperfluidDelegationAmountRequest) (*pb_pkg.SuperfluidDelegationAmountResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.SuperfluidDelegationAmount", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.SuperfluidDelegationAmountResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) SuperfluidDelegationsByDelegator(ctx context.Context, req *pb_pkg.SuperfluidDelegationsByDelegatorRequest) (*pb_pkg.SuperfluidDelegationsByDelegatorResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.SuperfluidDelegationsByDelegator", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.SuperfluidDelegationsByDelegatorResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) SuperfluidDelegationsByValidatorDenom(ctx context.Context, req *pb_pkg.SuperfluidDelegationsByValidatorDenomRequest) (*pb_pkg.SuperfluidDelegationsByValidatorDenomResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.SuperfluidDelegationsByValidatorDenom", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.SuperfluidDelegationsByValidatorDenomResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) SuperfluidUndelegationsByDelegator(ctx context.Context, req *pb_pkg.SuperfluidUndelegationsByDelegatorRequest) (*pb_pkg.SuperfluidUndelegationsByDelegatorResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.SuperfluidUndelegationsByDelegator", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.SuperfluidUndelegationsByDelegatorResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) TotalDelegationByDelegator(ctx context.Context, req *pb_pkg.QueryTotalDelegationByDelegatorRequest) (*pb_pkg.QueryTotalDelegationByDelegatorResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.TotalDelegationByDelegator", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryTotalDelegationByDelegatorResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method



func (is *implementedOsmosisSuperfluid) TotalSuperfluidDelegations(ctx context.Context, req *pb_pkg.TotalSuperfluidDelegationsRequest) (*pb_pkg.TotalSuperfluidDelegationsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.superfluid.Query.TotalSuperfluidDelegations", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.TotalSuperfluidDelegationsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}
// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
