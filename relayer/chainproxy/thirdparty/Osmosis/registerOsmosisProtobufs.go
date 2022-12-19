package osmosis_thirdparty

import (
	"context"

	pb_pkg2 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/epochs/types"
	pb_pkg3 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/gamm/types"
	pb_pkg4 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/incentives/types"
	pb_pkg5 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/lockup/types"
	pb_pkg6 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/mint/types"
	pb_pkg7 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/pool-incentives/types"
	pkg1 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/superfluid/types"
	pb_pkg8 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/tokenfactory/types"
	pb_pkg9 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/twap/types"
	pb_pkg10 "github.com/lavanet/lava/relayer/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs/txfees/types"
	"google.golang.org/grpc"
)

func RegisterOsmosisProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	osmosissuperfluid := &implementedOsmosisSuperfluid{cb: cb}
	pkg1.RegisterQueryServer(s, osmosissuperfluid)

	osmosisepochsv1beta1 := &implementedOsmosisEpochsV1beta1{cb: cb}
	pb_pkg2.RegisterQueryServer(s, osmosisepochsv1beta1)

	osmosisgammv1beta1 := &implementedOsmosisGammV1beta1{cb: cb}
	pb_pkg3.RegisterQueryServer(s, osmosisgammv1beta1)

	osmosisincentives := &implementedOsmosisIncentives{cb: cb}
	pb_pkg4.RegisterQueryServer(s, osmosisincentives)

	osmosislockup := &implementedOsmosisLockup{cb: cb}
	pb_pkg5.RegisterQueryServer(s, osmosislockup)

	osmosismintv1beta1 := &implementedOsmosisMintV1beta1{cb: cb}
	pb_pkg6.RegisterQueryServer(s, osmosismintv1beta1)

	osmosispoolincentivesv1beta1 := &implementedOsmosisPoolincentivesV1beta1{cb: cb}
	pb_pkg7.RegisterQueryServer(s, osmosispoolincentivesv1beta1)

	osmosistokenfactoryv1beta1 := &implementedOsmosisTokenfactoryV1beta1{cb: cb}
	pb_pkg8.RegisterQueryServer(s, osmosistokenfactoryv1beta1)

	osmosistwapv1beta1 := &implementedOsmosisTwapV1beta1{cb: cb}
	pb_pkg9.RegisterQueryServer(s, osmosistwapv1beta1)

	osmosistxfeesv1beta1 := &implementedOsmosisTxfeesV1beta1{cb: cb}
	pb_pkg10.RegisterQueryServer(s, osmosistxfeesv1beta1)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
