package ibc_thirdparty

import (
	"context"
	"encoding/json"

	// add protobuf here as pb_pkg
	"github.com/lavanet/lava/utils"
)

type implementedIbcCoreClientV1 struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedIbcCoreClientV1

func (is *implementedIbcCoreClientV1) ClientParams(ctx context.Context, req *pb_pkg.QueryClientParamsRequest) (*pb_pkg.QueryClientParamsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.ClientParams", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryClientParamsResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) ClientState(ctx context.Context, req *pb_pkg.QueryClientStateRequest) (*pb_pkg.QueryClientStateResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.ClientState", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryClientStateResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) ClientStates(ctx context.Context, req *pb_pkg.QueryClientStatesRequest) (*pb_pkg.QueryClientStatesResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.ClientStates", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryClientStatesResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) ClientStatus(ctx context.Context, req *pb_pkg.QueryClientStatusRequest) (*pb_pkg.QueryClientStatusResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.ClientStatus", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryClientStatusResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) ConsensusState(ctx context.Context, req *pb_pkg.QueryConsensusStateRequest) (*pb_pkg.QueryConsensusStateResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.ConsensusState", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryConsensusStateResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) ConsensusStates(ctx context.Context, req *pb_pkg.QueryConsensusStatesRequest) (*pb_pkg.QueryConsensusStatesResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.ConsensusStates", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryConsensusStatesResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) UpgradedClientState(ctx context.Context, req *pb_pkg.QueryUpgradedClientStateRequest) (*pb_pkg.QueryUpgradedClientStateResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.UpgradedClientState", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryUpgradedClientStateResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) UpgradedConsensusState(ctx context.Context, req *pb_pkg.QueryUpgradedConsensusStateRequest) (*pb_pkg.QueryUpgradedConsensusStateResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.UpgradedConsensusState", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryUpgradedConsensusStateResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedIbcCoreClientV1) ConsensusStateHeights(ctx context.Context, req *pb_pkg.QueryConsensusStateHeightsRequest) (*pb_pkg.QueryConsensusStateHeightsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "ibc.core.client.v1.Query.ConsensusStateHeights", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryConsensusStateHeightsResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
