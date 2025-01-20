package lavaprotocol

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestSignAndExtract(t *testing.T) {
	ctx := context.Background()
	sk, address := sigs.GenerateFloatingKey()
	specId := "LAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:              20,
		LatestRelayCu:      10, // set by GetSessions cuNeededForSession
		SessionId:          123,
		Parent:             nil,
		RelayNum:           1,
		LatestBlock:        epoch,
		EndpointConnection: nil,
		BlockListed:        false, // if session lost sync we blacklist it.
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	qosManagerMock := NewMockQoSManager(ctrl)
	qosManagerMock.
		EXPECT().
		GetLastReputationQoSReportRaw(gomock.Any(), gomock.Any()).
		AnyTimes()

	qosManagerMock.
		EXPECT().
		GetLastQoSReport(gomock.Any(), gomock.Any()).
		AnyTimes()

	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "x-cosmos-block-height:",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, 10, "tendermintrpc", metadataValue, "test", nil)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := ConstructRelayRequest(ctx, sk, "lava", specId, relayRequestData, "lava@stubProviderAddress", singleConsumerSession, epoch, unresponsiveProviderStub(), qosManagerMock)
	require.NoError(t, err)

	// check signature
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.NoError(t, err)
	require.Equal(t, extractedConsumerAddress, address)
}
