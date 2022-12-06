package lava_thirdparty

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type implementedQueryServer struct {
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

func (qs *implementedQueryServer) Params(ctx context.Context, req *pairingtypes.QueryParamsRequest) (*pairingtypes.QueryParamsResponse, error) {
	reqMarshaled, err := proto.Marshal(req)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.cb(ctx, "lavanet.lava.pairing.Query/Params", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	result := &pairingtypes.QueryParamsResponse{}
	err = proto.Unmarshal(res, result)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
	}
	return result, nil
}

func (qs *implementedQueryServer) Providers(ctx context.Context, req *pairingtypes.QueryProvidersRequest) (*pairingtypes.QueryProvidersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Providers not implemented")
}
func (qs *implementedQueryServer) Clients(ctx context.Context, req *pairingtypes.QueryClientsRequest) (*pairingtypes.QueryClientsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Clients not implemented")
}
func (qs *implementedQueryServer) GetPairing(ctx context.Context, req *pairingtypes.QueryGetPairingRequest) (*pairingtypes.QueryGetPairingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPairing not implemented")
}
func (qs *implementedQueryServer) VerifyPairing(ctx context.Context, req *pairingtypes.QueryVerifyPairingRequest) (*pairingtypes.QueryVerifyPairingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyPairing not implemented")
}
func (qs *implementedQueryServer) UniquePaymentStorageClientProvider(ctx context.Context, req *pairingtypes.QueryGetUniquePaymentStorageClientProviderRequest) (*pairingtypes.QueryGetUniquePaymentStorageClientProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UniquePaymentStorageClientProvider not implemented")
}
func (qs *implementedQueryServer) UniquePaymentStorageClientProviderAll(ctx context.Context, req *pairingtypes.QueryAllUniquePaymentStorageClientProviderRequest) (*pairingtypes.QueryAllUniquePaymentStorageClientProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UniquePaymentStorageClientProviderAll not implemented")
}
func (qs *implementedQueryServer) ProviderPaymentStorage(ctx context.Context, req *pairingtypes.QueryGetProviderPaymentStorageRequest) (*pairingtypes.QueryGetProviderPaymentStorageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProviderPaymentStorage not implemented")
}
func (qs *implementedQueryServer) ProviderPaymentStorageAll(ctx context.Context, req *pairingtypes.QueryAllProviderPaymentStorageRequest) (*pairingtypes.QueryAllProviderPaymentStorageResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProviderPaymentStorageAll not implemented")
}
func (qs *implementedQueryServer) EpochPayments(ctx context.Context, req *pairingtypes.QueryGetEpochPaymentsRequest) (*pairingtypes.QueryGetEpochPaymentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EpochPayments not implemented")
}
func (qs *implementedQueryServer) EpochPaymentsAll(ctx context.Context, req *pairingtypes.QueryAllEpochPaymentsRequest) (*pairingtypes.QueryAllEpochPaymentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EpochPaymentsAll not implemented")
}
func (qs *implementedQueryServer) UserEntry(ctx context.Context, req *pairingtypes.QueryUserEntryRequest) (*pairingtypes.QueryUserEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserEntry not implemented")
}
