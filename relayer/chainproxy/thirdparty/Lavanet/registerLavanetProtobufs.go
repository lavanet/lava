
package lavanet_thirdparty

import (
	"context"

	"google.golang.org/grpc"
)

func RegisterLavaProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	
lavanetlavaconflict := &implementedLavanetLavaConflict{cb: cb}
pkg.RegisterQueryServer(s, lavanetlavaconflict)


lavanetlavaepochstorage := &implementedLavanetLavaEpochstorage{cb: cb}
pkg.RegisterQueryServer(s, lavanetlavaepochstorage)


lavanetlavapairing := &implementedLavanetLavaPairing{cb: cb}
pkg.RegisterQueryServer(s, lavanetlavapairing)


lavanetlavaspec := &implementedLavanetLavaSpec{cb: cb}
pkg.RegisterQueryServer(s, lavanetlavaspec)

// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
