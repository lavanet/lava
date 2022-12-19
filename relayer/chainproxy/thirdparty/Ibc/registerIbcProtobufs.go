package ibc_thirdparty

import (
	"context"

	pb_pkg6 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/ibc/apps/interchain-accounts/controller/types"
	pb_pkg5 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/ibc/apps/interchain-accounts/host/types"
	pb_pkg1 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/ibc/apps/transfer/types"
	pb_pkg2 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/ibc/core/channel/types"
	pb_pkg3 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/ibc/core/client/types"
	pb_pkg4 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/ibc/core/connection/types"
	"google.golang.org/grpc"
)

func RegisterLavaProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pb_pkg1.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pb_pkg2.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pb_pkg3.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pb_pkg4.RegisterQueryServer(s, ibccoreconnectionv1)

	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pb_pkg5.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationsinterchain_accountscontrollerv1 := &implementedIbcApplicationsInterchain_accountsControllerV1{cb: cb}
	pb_pkg6.RegisterQueryServer(s, ibcapplicationsinterchain_accountscontrollerv1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterOsmosisProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pb_pkg5.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pb_pkg1.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pb_pkg2.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pb_pkg3.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pb_pkg4.RegisterQueryServer(s, ibccoreconnectionv1)

	ibcapplicationsinterchain_accountscontrollerv1 := &implementedIbcApplicationsInterchain_accountsControllerV1{cb: cb}
	pb_pkg6.RegisterQueryServer(s, ibcapplicationsinterchain_accountscontrollerv1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterCosmosProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	ibcapplicationsinterchain_accountscontrollerv1 := &implementedIbcApplicationsInterchain_accountsControllerV1{cb: cb}
	pb_pkg6.RegisterQueryServer(s, ibcapplicationsinterchain_accountscontrollerv1)

	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pb_pkg5.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pb_pkg1.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pb_pkg2.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pb_pkg3.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pb_pkg4.RegisterQueryServer(s, ibccoreconnectionv1)

	// this line is used by grpc_scaffolder #Register
}

func RegisterJunoProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	ibcapplicationsinterchain_accountshostv1 := &implementedIbcApplicationsInterchain_accountsHostV1{cb: cb}
	pb_pkg5.RegisterQueryServer(s, ibcapplicationsinterchain_accountshostv1)

	ibcapplicationstransferv1 := &implementedIbcApplicationsTransferV1{cb: cb}
	pb_pkg1.RegisterQueryServer(s, ibcapplicationstransferv1)

	ibccorechannelv1 := &implementedIbcCoreChannelV1{cb: cb}
	pb_pkg2.RegisterQueryServer(s, ibccorechannelv1)

	ibccoreclientv1 := &implementedIbcCoreClientV1{cb: cb}
	pb_pkg3.RegisterQueryServer(s, ibccoreclientv1)

	ibccoreconnectionv1 := &implementedIbcCoreConnectionV1{cb: cb}
	pb_pkg4.RegisterQueryServer(s, ibccoreconnectionv1)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
