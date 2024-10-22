package updaters

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v4/utils/rand"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	gomock "go.uber.org/mock/gomock"
)

func TestFreezeJailMetricsOnEpochUpdate(t *testing.T) {
	rand.InitRandomSeed()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	specID := "test-spec"
	address := "initial1"
	epoch := uint64(100)
	stakeAppliedBlock := uint64(10)

	stakeEntryList := []epochstoragetypes.StakeEntry{
		{
			Address: address,
			Chain:   specID,
			Endpoints: []epochstoragetypes.Endpoint{
				{
					IPPORT:        "1234567",
					Geolocation:   1,
					Addons:        []string{},
					ApiInterfaces: []string{"banana"},
					Extensions:    []string{},
				},
			},
			StakeAppliedBlock: stakeAppliedBlock,
		},
	}

	response := &pairingtypes.QueryProviderResponse{StakeEntries: stakeEntryList}

	stateQuery := NewMockProviderPairingStatusStateQueryInf(ctrl)
	metricManager := NewMockProviderMetricsManagerInf(ctrl)

	freezeUpdater := NewProviderFreezeJailUpdater(stateQuery, address, metricManager)

	expectAndRun := func(stakeAppliedBlock, jailedCount uint64, frozen bool, jailed bool) {
		stakeEntryList[0].StakeAppliedBlock = stakeAppliedBlock
		stakeEntryList[0].Jails = jailedCount
		if jailed {
			stakeEntryList[0].JailEndTime = time.Now().Add(time.Hour).UTC().Unix()
		}
		response = &pairingtypes.QueryProviderResponse{StakeEntries: stakeEntryList}
		stateQuery.
			EXPECT().
			Provider(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(response, nil).
			AnyTimes()

		metricManager.
			EXPECT().
			SetJailStatus(specID, jailed).
			Times(1)

		metricManager.
			EXPECT().
			SetFrozenStatus(specID, frozen).
			Times(1)

		metricManager.
			EXPECT().
			SetJailedCount(specID, jailedCount).
			Times(1)

		freezeUpdater.UpdateEpoch(epoch)
	}

	// Normal - no freeze, no jail
	expectAndRun(stakeAppliedBlock, 0, false, false)

	// StakeAppliedBlock > epoch - frozen
	expectAndRun(epoch+1, 0, true, false)

	// Jail status changed + jail count
	expectAndRun(epoch-1, 1, false, true)
}
