package osmosis_thirdparty

import (
	"context"
	"encoding/json"

	"github.com/golang/protobuf/proto"
	pb_pkg "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/twap/types"
	"github.com/lavanet/lava/utils"
)

type implementedOsmosisTwapV1beta1 struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedOsmosisTwapV1beta1

func (is *implementedOsmosisTwapV1beta1) ArithmeticTwap(ctx context.Context, req *pb_pkg.ArithmeticTwapRequest) (*pb_pkg.ArithmeticTwapResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.twap.v1beta1.Query.ArithmeticTwap", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.ArithmeticTwapResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisTwapV1beta1) ArithmeticTwapToNow(ctx context.Context, req *pb_pkg.ArithmeticTwapToNowRequest) (*pb_pkg.ArithmeticTwapToNowResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.twap.v1beta1.Query.ArithmeticTwapToNow", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.ArithmeticTwapToNowResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedOsmosisTwapV1beta1) Params(ctx context.Context, req *pb_pkg.ParamsRequest) (*pb_pkg.ParamsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "osmosis.twap.v1beta1.Query.Params", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.ParamsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
