package rpcprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	types "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

type relaySenderMock struct {
	numberOfTimesHitSendNodeMsg int
}

func (rs *relaySenderMock) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage chainlib.ChainMessageForSend, extensions []string) (relayReply *chainlib.RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, proxyUrl common.NodeUrl, chainId string, err error) {
	rs.numberOfTimesHitSendNodeMsg++
	return &chainlib.RelayReplyWrapper{RelayReply: &types.RelayReply{}}, "", nil, common.NodeUrl{}, "", nil
}

func TestStateMachineHappyFlow(t *testing.T) {
	relaySender := &relaySenderMock{}
	stateMachine := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), relaySender, numberOfRetriesAllowedOnNodeErrors, nil)
	chainMsgMock := chainlib.NewMockChainMessage(gomock.NewController(t))
	chainMsgMock.
		EXPECT().
		GetRawRequestHash().
		Return([]byte{1, 2, 3}, nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApi().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApiCollection().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		CheckResponseError(gomock.Any(), gomock.Any()).
		DoAndReturn(func(msg interface{}, msg2 interface{}) (interface{}, interface{}) {
			if relaySender.numberOfTimesHitSendNodeMsg < numberOfRetriesAllowedOnNodeErrors {
				return true, ""
			}
			return false, ""
		}).
		AnyTimes()
	stateMachine.SendNodeMessage(context.Background(), chainMsgMock, &types.RelayRequest{RelayData: &types.RelayPrivateData{Extensions: []string{}}})
	hash, _ := chainMsgMock.GetRawRequestHash()
	require.Equal(t, relaySender.numberOfTimesHitSendNodeMsg, numberOfRetriesAllowedOnNodeErrors)
	require.False(t, stateMachine.relayRetriesManager.CheckHashInCache(string(hash)))
}

func TestStateMachineAllFailureFlows(t *testing.T) {
	relaySender := &relaySenderMock{}
	stateMachine := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), relaySender, numberOfRetriesAllowedOnNodeErrors, nil)
	chainMsgMock := chainlib.NewMockChainMessage(gomock.NewController(t))
	returnFalse := false
	chainMsgMock.
		EXPECT().
		GetRawRequestHash().
		Return([]byte{1, 2, 3}, nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApi().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApiCollection().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		CheckResponseError(gomock.Any(), gomock.Any()).
		DoAndReturn(func(msg interface{}, msg2 interface{}) (interface{}, interface{}) {
			if returnFalse {
				return false, ""
			}
			return true, "some node error"
		}).
		AnyTimes()
	stateMachine.SendNodeMessage(context.Background(), chainMsgMock, &types.RelayRequest{RelayData: &types.RelayPrivateData{Extensions: []string{}}})
	hash, _ := chainMsgMock.GetRawRequestHash()
	require.Equal(t, numberOfRetriesAllowedOnNodeErrors+1, relaySender.numberOfTimesHitSendNodeMsg)
	for i := 0; i < 10; i++ {
		// wait for routine to end and cache to become consistent
		if stateMachine.relayRetriesManager.CheckHashInCache(string(hash)) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, stateMachine.relayRetriesManager.CheckHashInCache(string(hash)))

	// send second relay with same hash.
	stateMachine.SendNodeMessage(context.Background(), chainMsgMock, &types.RelayRequest{RelayData: &types.RelayPrivateData{Extensions: []string{}}})
	require.Equal(t, 4, relaySender.numberOfTimesHitSendNodeMsg) // no retries.
}

func TestStateMachineFailureAndRecoveryFlow(t *testing.T) {
	relaySender := &relaySenderMock{}
	stateMachine := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), relaySender, numberOfRetriesAllowedOnNodeErrors, nil)
	chainMsgMock := chainlib.NewMockChainMessage(gomock.NewController(t))
	returnFalse := false
	chainMsgMock.
		EXPECT().
		GetRawRequestHash().
		Return([]byte{1, 2, 3}, nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApi().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApiCollection().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		CheckResponseError(gomock.Any(), gomock.Any()).
		DoAndReturn(func(msg interface{}, msg2 interface{}) (interface{}, interface{}) {
			if returnFalse {
				return false, ""
			}
			return true, "some node error"
		}).
		AnyTimes()
	stateMachine.SendNodeMessage(context.Background(), chainMsgMock, &types.RelayRequest{RelayData: &types.RelayPrivateData{Extensions: []string{}}})
	hash, _ := chainMsgMock.GetRawRequestHash()
	require.Equal(t, numberOfRetriesAllowedOnNodeErrors+1, relaySender.numberOfTimesHitSendNodeMsg)
	for i := 0; i < 10; i++ {
		// wait for routine to end and cache to become consistent
		if stateMachine.relayRetriesManager.CheckHashInCache(string(hash)) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, stateMachine.relayRetriesManager.CheckHashInCache(string(hash)))

	// send second relay with same hash.
	returnFalse = true
	stateMachine.SendNodeMessage(context.Background(), chainMsgMock, &types.RelayRequest{RelayData: &types.RelayPrivateData{Extensions: []string{}}})
	require.Equal(t, 4, relaySender.numberOfTimesHitSendNodeMsg) // no retries, first success.
	// wait for routine to end..
	for i := 0; i < 10; i++ {
		if !stateMachine.relayRetriesManager.CheckHashInCache(string(hash)) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.False(t, stateMachine.relayRetriesManager.CheckHashInCache(string(hash)))
}

type unsupportedMethodRelaySenderMock struct {
	numberOfTimesHitSendNodeMsg int
}

func (rs *unsupportedMethodRelaySenderMock) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage chainlib.ChainMessageForSend, extensions []string) (relayReply *chainlib.RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, proxyUrl common.NodeUrl, chainId string, err error) {
	rs.numberOfTimesHitSendNodeMsg++
	// Return a response that contains an unsupported method error
	return &chainlib.RelayReplyWrapper{
		RelayReply: &types.RelayReply{
			Data: []byte(`{"error":{"code":-32601,"message":"method not found"},"id":1}`),
		},
		StatusCode: 200,
	}, "", nil, common.NodeUrl{}, "", nil
}

func TestStateMachineUnsupportedMethodError(t *testing.T) {
	relaySender := &unsupportedMethodRelaySenderMock{}
	stateMachine := NewProviderStateMachine("test", lavaprotocol.NewRelayRetriesManager(), relaySender, numberOfRetriesAllowedOnNodeErrors, nil)
	chainMsgMock := chainlib.NewMockChainMessage(gomock.NewController(t))

	// Mock chain message behavior
	chainMsgMock.
		EXPECT().
		GetRawRequestHash().
		Return([]byte{1, 2, 3}, nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApi().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		GetApiCollection().
		Return(nil).
		AnyTimes()
	chainMsgMock.
		EXPECT().
		CheckResponseError(gomock.Any(), gomock.Any()).
		Return(true, "method not found").
		AnyTimes()

	// Execute the test
	ctx := context.Background()
	result, err := stateMachine.SendNodeMessage(ctx, chainMsgMock, &types.RelayRequest{
		RelayData: &types.RelayPrivateData{Extensions: []string{}},
		RelaySession: &types.RelaySession{
			SessionId: 123,
			RelayNum:  1,
		},
	})

	// Verify that an error is returned for unsupported methods
	require.Error(t, err)
	require.Nil(t, result)

	// Verify that the error is an UnsupportedMethodError
	var unsupportedError *chainlib.UnsupportedMethodError
	require.True(t, errors.As(err, &unsupportedError))
	require.Contains(t, err.Error(), "unsupported method")

	// Verify that we only hit the relay sender once (no retries for unsupported methods)
	require.Equal(t, 1, relaySender.numberOfTimesHitSendNodeMsg)
}
