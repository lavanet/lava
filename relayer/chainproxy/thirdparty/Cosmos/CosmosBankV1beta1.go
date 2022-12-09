package cosmos_thirdparty

import (
	"context"
	"encoding/json"

	"github.com/golang/protobuf/proto"
	pb_pkg "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/cosmos/bank/types"
	"github.com/lavanet/lava/utils"
)

type implementedCosmosBankV1beta1 struct {
	pb_pkg.UnimplementedQueryServer
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

// this line is used by grpc_scaffolder #implementedCosmosBankV1beta1

func (is *implementedCosmosBankV1beta1) AllBalances(ctx context.Context, req *pb_pkg.QueryAllBalancesRequest) (*pb_pkg.QueryAllBalancesResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.AllBalances", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryAllBalancesResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) DenomMetadata(ctx context.Context, req *pb_pkg.QueryDenomMetadataRequest) (*pb_pkg.QueryDenomMetadataResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.DenomMetadata", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryDenomMetadataResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) DenomsMetadata(ctx context.Context, req *pb_pkg.QueryDenomsMetadataRequest) (*pb_pkg.QueryDenomsMetadataResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.DenomsMetadata", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryDenomsMetadataResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) Params(ctx context.Context, req *pb_pkg.QueryParamsRequest) (*pb_pkg.QueryParamsResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.Params", reqMarshaled)
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

func (is *implementedCosmosBankV1beta1) SpendableBalances(ctx context.Context, req *pb_pkg.QuerySpendableBalancesRequest) (*pb_pkg.QuerySpendableBalancesResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.SpendableBalances", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QuerySpendableBalancesResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) SupplyOf(ctx context.Context, req *pb_pkg.QuerySupplyOfRequest) (*pb_pkg.QuerySupplyOfResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.SupplyOf", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QuerySupplyOfResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) TotalSupply(ctx context.Context, req *pb_pkg.QueryTotalSupplyRequest) (*pb_pkg.QueryTotalSupplyResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.TotalSupply", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryTotalSupplyResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) DenomOwners(ctx context.Context, req *pb_pkg.QueryDenomOwnersRequest) (*pb_pkg.QueryDenomOwnersResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.DenomOwners", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryDenomOwnersResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// // this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) SendEnabled(ctx context.Context, req *pb_pkg.QuerySendEnabledRequest) (*pb_pkg.QuerySendEnabledResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.SendEnabled", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QuerySendEnabledResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// // this line is used by grpc_scaffolder #Method

func (is *implementedCosmosBankV1beta1) Balance(ctx context.Context, req *pb_pkg.QueryBalanceRequest) (*pb_pkg.QueryBalanceResponse, error) {
	reqMarshaled, err := json.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.Balance", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pb_pkg.QueryBalanceResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

// // this line is used by grpc_scaffolder #Method
// func (is *implementedCosmosBankV1beta1) SupplyOfWithoutOffset(ctx context.Context, req *pb_pkg.QuerySupplyOfWithoutOffsetRequest) (*pb_pkg.QuerySupplyOfWithoutOffsetResponse, error) {
// 	reqMarshaled, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
// 	}
// 	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.SupplyOfWithoutOffset", reqMarshaled)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
// 	}
// 	result := &pb_pkg.QuerySupplyOfWithoutOffsetResponse{}
// 	err = proto.Unmarshal(res, result)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
// 	}
// 	return result, nil
// }

// // this line is used by grpc_scaffolder #Method

// func (is *implementedCosmosBankV1beta1) TotalSupplyWithoutOffset(ctx context.Context, req *pb_pkg.QueryTotalSupplyWithoutOffsetRequest) (*pb_pkg.QueryTotalSupplyWithoutOffsetResponse, error) {
// 	reqMarshaled, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
// 	}
// 	res, err := is.cb(ctx, "cosmos.bank.v1beta1.Query.TotalSupplyWithoutOffset", reqMarshaled)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
// 	}
// 	result := &pb_pkg.QueryTotalSupplyWithoutOffsetResponse{}
// 	err = proto.Unmarshal(res, result)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
// 	}
// 	return result, nil
// }

// this line is used by grpc_scaffolder #Method

// this line is used by grpc_scaffolder #Methods
