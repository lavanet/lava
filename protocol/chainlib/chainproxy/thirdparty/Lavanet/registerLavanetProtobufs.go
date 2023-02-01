package lavanet_thirdparty

import (
	"context"

	conflict "github.com/lavanet/lava/x/conflict/types"
	epochstorage "github.com/lavanet/lava/x/epochstorage/types"
	pairing "github.com/lavanet/lava/x/pairing/types"
	spec "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
)

func RegisterLavaProtobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {
	lavanetlavaconflict := &implementedLavanetLavaConflict{cb: cb}
	conflict.RegisterQueryServer(s, lavanetlavaconflict)

	lavanetlavaepochstorage := &implementedLavanetLavaEpochstorage{cb: cb}
	epochstorage.RegisterQueryServer(s, lavanetlavaepochstorage)

	lavanetlavapairing := &implementedLavanetLavaPairing{cb: cb}
	pairing.RegisterQueryServer(s, lavanetlavapairing)

	lavanetlavaspec := &implementedLavanetLavaSpec{cb: cb}
	spec.RegisterQueryServer(s, lavanetlavaspec)

	// this line is used by grpc_scaffolder #Register
}

// this line is used by grpc_scaffolder #Registration
