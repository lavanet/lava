package mock_test

import (
	"context"

	"google.golang.org/grpc"
)

type MockGRPCConnector struct {
	conn *grpc.ClientConn
}

func NewMockGRPCConnector(conn *grpc.ClientConn) *MockGRPCConnector {
	return &MockGRPCConnector{conn: conn}
}

func (mc *MockGRPCConnector) Close() {
	mc.conn.Close()
}

func (mc *MockGRPCConnector) ReturnRpc(rpc *grpc.ClientConn) {
}

func (mc *MockGRPCConnector) GetRpc(ctx context.Context, block bool) (*grpc.ClientConn, error) {
	return mc.conn, nil
}
