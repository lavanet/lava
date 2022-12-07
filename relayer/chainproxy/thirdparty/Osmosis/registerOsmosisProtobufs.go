
package osmosis_thirdparty

import (
	"context"

	"google.golang.org/grpc"
)

func RegisterOsmosisProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	
osmosissuperfluid := &implementedOsmosisSuperfluid{cb: cb}
pkg.RegisterQueryServer(s, osmosissuperfluid)


osmosisepochsv1beta1 := &implementedOsmosisEpochsV1beta1{cb: cb}
pkg.RegisterQueryServer(s, osmosisepochsv1beta1)


osmosisgammv1beta1 := &implementedOsmosisGammV1beta1{cb: cb}
pkg.RegisterQueryServer(s, osmosisgammv1beta1)


osmosisincentives := &implementedOsmosisIncentives{cb: cb}
pkg.RegisterQueryServer(s, osmosisincentives)


osmosislockup := &implementedOsmosisLockup{cb: cb}
pkg.RegisterQueryServer(s, osmosislockup)


osmosismintv1beta1 := &implementedOsmosisMintV1beta1{cb: cb}
pkg.RegisterQueryServer(s, osmosismintv1beta1)


osmosispoolincentivesv1beta1 := &implementedOsmosisPoolincentivesV1beta1{cb: cb}
pkg.RegisterQueryServer(s, osmosispoolincentivesv1beta1)


osmosistokenfactoryv1beta1 := &implementedOsmosisTokenfactoryV1beta1{cb: cb}
pkg.RegisterQueryServer(s, osmosistokenfactoryv1beta1)


osmosistwapv1beta1 := &implementedOsmosisTwapV1beta1{cb: cb}
pkg.RegisterQueryServer(s, osmosistwapv1beta1)


osmosistxfeesv1beta1 := &implementedOsmosisTxfeesV1beta1{cb: cb}
pkg.RegisterQueryServer(s, osmosistxfeesv1beta1)

// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
