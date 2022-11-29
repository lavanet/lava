package grpcutil

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Dial(ctx context.Context, host string) (conn *grpc.ClientConn, err error) {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	return grpc.DialContext(tCtx, host, []grpc.DialOption{
		grpc.WithAuthority(host),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		}),
	}...)
}

func MustDial(ctx context.Context, host string) (conn *grpc.ClientConn) {
	var err error
	if conn, err = Dial(ctx, host); err != nil {
		panic(err)
	}

	return conn
}

func Marshal(m proto.Message) (b []byte, err error) {
	marshalOptions := protojson.MarshalOptions{
		Multiline:      true,
		Indent:         "  ",
		AllowPartial:   true,
		UseProtoNames:  true,
		UseEnumNumbers: false,
	}
	return marshalOptions.Marshal(m)
}

func Unmarshal(b []byte, m proto.Message) (err error) {
	unmarshalOptions := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	return unmarshalOptions.Unmarshal(b, m)
}

func Log(m proto.Message) {
	b, _ := Marshal(m)
	log.Println(string(b))
}
