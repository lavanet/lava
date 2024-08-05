package rpcprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	testcommon "github.com/lavanet/lava/v2/testutil/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSetSpecWithoutChainFetchersNorProviderListenerNoErrors(t *testing.T) {
	spec := testcommon.CreateMockSpec()

	specValidator := NewSpecValidator()
	require.NotPanics(t, func() { specValidator.VerifySpec(spec) })
}

func TestSetSpecWithoutRelayReceiversNoErrors(t *testing.T) {
	spec := testcommon.CreateMockSpec()

	providerListener := NewProviderListener(context.Background(), lavasession.NetworkAddressData{}, "")
	providerListener.RegisterReceiver(&RPCProviderServer{}, &lavasession.RPCProviderEndpoint{})

	specValidator := NewSpecValidator()
	specValidator.AddRPCProviderListener("", providerListener)
	require.NotPanics(t, func() { specValidator.VerifySpec(spec) })
}

func TestAddChainFetcherAndSetSpecCallsValidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	spec := testcommon.CreateMockSpec()
	specValidator := NewSpecValidator()

	specName := "LAV1"
	spec.Index = specName

	ctx := context.Background()

	chainFetcher := chainlib.NewMockChainFetcherIf(ctrl)
	var chainFetcherIf chainlib.ChainFetcherIf = chainFetcher

	chainFetcher.EXPECT().FetchEndpoint().AnyTimes()

	firstCall := chainFetcher.EXPECT().Validate(gomock.Any()).Times(1)
	specValidator.AddChainFetcher(ctx, &chainFetcherIf, specName)

	chainFetcher.EXPECT().Validate(gomock.Any()).Times(1).After(firstCall)
	specValidator.VerifySpec(spec)
}

func TestStartCallsAllValidateFunctions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	spec := testcommon.CreateMockSpec()
	specValidator := NewSpecValidator()

	specName := "LAV1"
	spec.Index = specName

	ctx := context.Background()

	done := make(chan bool)
	calls := 0

	raiseCallCount := func(data interface{}) {
		calls++
		if calls == 10 {
			done <- true
		}
	}

	for i := 0; i < 10; i++ {
		chainFetcher := chainlib.NewMockChainFetcherIf(ctrl)

		chainFetcher.EXPECT().FetchEndpoint().AnyTimes()
		firstCall := chainFetcher.EXPECT().Validate(gomock.Any()).Times(1)

		var chainFetcherIf chainlib.ChainFetcherIf = chainFetcher
		specValidator.AddChainFetcher(ctx, &chainFetcherIf, specName)

		chainFetcher.EXPECT().Validate(gomock.Any()).Times(1).After(firstCall).Do(raiseCallCount)
	}
	SpecValidationInterval = 1 * time.Second
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	specValidator.Start(ctx)

	select {
	case <-done:
		t.Log("Done validations")
	case <-ctx.Done():
		t.Fatal("The operation timed out")
	}
}

func TestFailedThenSuccessVerificationDisablesThenEnablesReceiver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	spec := testcommon.CreateMockSpec()
	specValidator := NewSpecValidator()

	specName := "LAV1"
	spec.Index = specName

	ctx := context.Background()

	chainFetcher := chainlib.NewMockChainFetcherIf(ctrl)
	chainFetcher.EXPECT().FetchEndpoint().AnyTimes()

	firstCall := chainFetcher.EXPECT().Validate(gomock.Any()).Times(1)

	var chainFetcherIf chainlib.ChainFetcherIf = chainFetcher
	specValidator.AddChainFetcher(ctx, &chainFetcherIf, specName)

	secondCall := chainFetcher.EXPECT().Validate(gomock.Any()).Times(1).After(firstCall).Return(errors.New(""))

	addressData := lavasession.NetworkAddressData{}
	providerListener := NewProviderListener(context.Background(), addressData, "")

	relayReceiver := NewMockRelayReceiver(ctrl)
	rpcProviderEndpoint := &lavasession.RPCProviderEndpoint{}
	providerListener.RegisterReceiver(relayReceiver, rpcProviderEndpoint)
	specValidator.AddRPCProviderListener(addressData.Address, providerListener)

	// Check that the receiver is first enabled
	rpcEndpoint := specValidator.getRpcProviderEndpointFromChainFetcher(&chainFetcherIf)
	require.True(t, providerListener.relayServer.relayReceivers[rpcEndpoint.Key()].enabled)

	specValidator.validateChain(ctx, specName)

	// Check that the receiver is disabled
	rpcEndpoint = specValidator.getRpcProviderEndpointFromChainFetcher(&chainFetcherIf)
	require.False(t, providerListener.relayServer.relayReceivers[rpcEndpoint.Key()].enabled)

	chainFetcher.EXPECT().Validate(gomock.Any()).Times(1).After(secondCall).Return(nil)

	specValidator.validateChain(ctx, specName)

	// Check that the receiver is enabled
	rpcEndpoint = specValidator.getRpcProviderEndpointFromChainFetcher(&chainFetcherIf)
	require.True(t, providerListener.relayServer.relayReceivers[rpcEndpoint.Key()].enabled)
}
