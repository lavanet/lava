package ibc_thirdparty

import (
	"context"

	"google.golang.org/grpc"
)

func RegisterLavaProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreconnectionv1)

	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationsinterchain_accountscontrollerv1 := &implementedIbcApplicationsInterchain_accountsControllerV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationsinterchain_accountscontrollerv1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterOsmosisProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreconnectionv1)

	ibcapplicationsinterchain_accountscontrollerv1 := &implementedIbcApplicationsInterchain_accountsControllerV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationsinterchain_accountscontrollerv1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterCosmosProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	ibcapplicationsinterchain_accountscontrollerv1 := &implementedIbcApplicationsInterchain_accountsControllerV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationsinterchain_accountscontrollerv1)

	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreconnectionv1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterJunoProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {

	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pkg.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pkg.RegisterQueryServer(s, ibccoreconnectionv1)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
