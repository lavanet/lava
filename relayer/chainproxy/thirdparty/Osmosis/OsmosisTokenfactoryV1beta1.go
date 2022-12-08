package osmosis_thirdparty

import (
	"context"
	"encoding/json"

	pb_pkg "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/tokenfactory/types"
	"github.com/lavanet/lava/utils"
)

type implementedOsmosisTokenfactoryV1beta1 struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedOsmosisTokenfactoryV1beta1

func (is *implementedOsmosisTokenfactoryV1beta1) DenomAuthorityMetadata(ctx context.Context, req *pb_pkg.QueryDenomAuthorityMetadataRequest) (*pb_pkg.QueryDenomAuthorityMetadataResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.tokenfactory.v1beta1.Query.DenomAuthorityMetadata", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryDenomAuthorityMetadataResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisTokenfactoryV1beta1) DenomsFromCreator(ctx context.Context, req *pb_pkg.QueryDenomsFromCreatorRequest) (*pb_pkg.QueryDenomsFromCreatorResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.tokenfactory.v1beta1.Query.DenomsFromCreator", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryDenomsFromCreatorResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisTokenfactoryV1beta1) Params(ctx context.Context, req *pb_pkg.QueryParamsRequest) (*pb_pkg.QueryParamsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.tokenfactory.v1beta1.Query.Params", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryParamsResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
