package spec

import (
	"context"
	"os"

	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
	specpb "github.com/lavanet/lava/x/spec/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type SpecManager interface {
	Fetch(ctx context.Context) (pbSpec *specpb.Spec, err error)
	Save(pbSpec *specpb.Spec, path string) (err error)
}

type specManager struct {
	conn *grpc.ClientConn
}

func NewSpecManager(conn *grpc.ClientConn) SpecManager {
	return &specManager{
		conn: conn,
	}
}

func (m *specManager) Fetch(ctx context.Context) (pbSpec *specpb.Spec, err error) {
	var serviceDescSlice []*grpc.ServiceDesc
	if serviceDescSlice, err = grpcutil.GetServiceDescSlice(ctx, m.conn); err != nil {
		return nil, errors.Wrap(err, "grpcutil.GetServiceDescSlice()")
	}

	serviceAPISlice := make([]specpb.ServiceApi, 0)
	for _, serviceDesc := range serviceDescSlice {
		for _, methodDesc := range serviceDesc.Methods {
			serviceAPISlice = append(serviceAPISlice, specpb.ServiceApi{
				Name:         serviceDesc.ServiceName + "/" + methodDesc.MethodName,
				BlockParsing: specpb.BlockParser{}, // TODO
				ComputeUnits: 10,
				Enabled:      true,
				ApiInterfaces: []specpb.ApiInterface{
					{
						Interface:         "grpc",
						Type:              "", // TODO
						ExtraComputeUnits: 0,
					},
				},
				Category: &specpb.SpecCategory{
					Deterministic: true,
					Local:         false,
					Subscription:  false,
					Stateful:      0,
				},
				Parsing: specpb.Parsing{}, // TODO
			})
		}
	}

	return &specpb.Spec{
		Index:                     "",
		Name:                      "", // TODO
		Apis:                      serviceAPISlice,
		Enabled:                   true,
		ReliabilityThreshold:      268435455,
		ComparesHashes:            true,
		FinalizationCriteria:      0,
		SavedBlocks:               1,
		AverageBlockTime:          6500,
		AllowedBlockLagForQosSync: 2,
	}, nil
}

func (m *specManager) Save(pbSpec *specpb.Spec, path string) (err error) {
	var b []byte
	if b, err = pbSpec.Marshal(); err != nil {
		return errors.Wrap(err, "pbSpec.Marshal()")
	}

	var f *os.File
	if f, err = os.Create(path); err != nil {
		return errors.Wrap(err, "os.Create()")
	}
	defer func() { _ = f.Close() }()

	if _, err = f.Write(b); err != nil {
		return errors.Wrap(err, "f.Write()")
	}

	return nil
}
