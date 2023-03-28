package lavanet_thirdparty

import (
	"context"
	"encoding/json"

	// add protobuf here as pb_pkg
	"github.com/golang/protobuf/proto"
	"github.com/lavanet/lava/utils"
	pb_pkg "github.com/lavanet/lava/x/spec/types"
)

type implementedLavanetLavaSpec struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedLavanetLavaSpec

func (is *implementedLavanetLavaSpec) Params(ctx context.Context, req *pb_pkg.QueryParamsRequest) (*pb_pkg.QueryParamsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "lavanet.lava.spec.Query/Params", reqMarshaled)
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

func (is *implementedLavanetLavaSpec) ShowAllChains(ctx context.Context, req *pb_pkg.QueryShowAllChainsRequest) (*pb_pkg.QueryShowAllChainsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "lavanet.lava.spec.Query/ShowAllChains", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryShowAllChainsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedLavanetLavaSpec) ShowChainInfo(ctx context.Context, req *pb_pkg.QueryShowChainInfoRequest) (*pb_pkg.QueryShowChainInfoResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "lavanet.lava.spec.Query/ShowChainInfo", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryShowChainInfoResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedLavanetLavaSpec) SpecAll(ctx context.Context, req *pb_pkg.QueryAllSpecRequest) (*pb_pkg.QueryAllSpecResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	}
	res, err := is.cb(ctx, "lavanet.lava.spec.Query/SpecAll", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	}
	result := &pb_pkg.QueryAllSpecResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
