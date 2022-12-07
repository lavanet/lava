package cosmos_thirdparty

import (
	"context"
	"encoding/json"

	// add protobuf here as pb_pkg
	"github.com/lavanet/lava/utils"
)

type implementedCosmosTxV1beta1 struct {
	pb_pkg.UnimplementedServiceServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedCosmosTxV1beta1

func (is *implementedCosmosTxV1beta1) BroadcastTx(ctx context.Context, req *pb_pkg.BroadcastTxRequest) (*pb_pkg.BroadcastTxResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.tx.v1beta1.Service.BroadcastTx", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.BroadcastTxResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosTxV1beta1) GetBlockWithTxs(ctx context.Context, req *pb_pkg.GetBlockWithTxsRequest) (*pb_pkg.GetBlockWithTxsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.tx.v1beta1.Service.GetBlockWithTxs", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetBlockWithTxsResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosTxV1beta1) GetTx(ctx context.Context, req *pb_pkg.GetTxRequest) (*pb_pkg.GetTxResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.tx.v1beta1.Service.GetTx", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetTxResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosTxV1beta1) GetTxsEvent(ctx context.Context, req *pb_pkg.GetTxsEventRequest) (*pb_pkg.GetTxsEventResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.tx.v1beta1.Service.GetTxsEvent", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.GetTxsEventResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosTxV1beta1) Simulate(ctx context.Context, req *pb_pkg.SimulateRequest) (*pb_pkg.SimulateResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.tx.v1beta1.Service.Simulate", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.SimulateResponse{}
	err = json.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
