package updaters

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v4/utils/rand"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

func testFreezeAndJailsMetricsOnEpochUpdate(t *testing.T, freezeStatus FrozenStatus, jailsAmount uint64) {
	// setup expectations
	rand.InitRandomSeed()
	ctrlStateQuery := gomock.NewController(t)
	ctrlMetrics := gomock.NewController(t)

	defer ctrlStateQuery.Finish()
	defer ctrlMetrics.Finish()
	specID := "test-spec"
	address := "initial1"
	mockEpoch := uint64(1)
	stakeAppliedBlock := 0
	if freezeStatus == FROZEN {
		stakeAppliedBlock = 1000
	}

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
			StakeAppliedBlock: uint64(stakeAppliedBlock),
			Jails:             jailsAmount,
		},
	}
	response := &pairingtypes.QueryProviderResponse{StakeEntries: stakeEntryList}

	// Create a new mock objects
	stateQuery := NewMockProviderPairingStatusStateQueryInf(ctrlStateQuery)
	stateQuery.EXPECT().Provider(gomock.Any(), gomock.Any(), gomock.Any()).Return(response, nil).AnyTimes()
	metricManager := NewMockProviderMetricsManagerInf(ctrlMetrics)
	// set expect for correct metric calls
	metricManager.EXPECT().SetFrozenStatus(float64(freezeStatus), specID, address).Return().AnyTimes()
	metricManager.EXPECT().SetJailedStatus(jailsAmount, specID, address).Return().AnyTimes()

	// create and call freeze updater
	pau := NewProviderFreezeUpdater(stateQuery, specID, address, metricManager)
	pau.UpdateEpoch(mockEpoch)
}

func TestFrozenAvailability(t *testing.T) {
	testFreezeAndJailsMetricsOnEpochUpdate(t, FROZEN, 0)
}

func TestFrozenAvailabilityWithJails(t *testing.T) {
	testFreezeAndJailsMetricsOnEpochUpdate(t, FROZEN, 2)
}

func TestNonFrozenAvailability(t *testing.T) {
	testFreezeAndJailsMetricsOnEpochUpdate(t, AVAILABLE, 0)
}

func TestStakeEntryReplyOfDifferentAddress(t *testing.T) {
	// setup expectations
	rand.InitRandomSeed()
	ctrlStateQuery := gomock.NewController(t)
	ctrlMetrics := gomock.NewController(t)

	defer ctrlStateQuery.Finish()
	defer ctrlMetrics.Finish()
	specID := "test-spec"
	address := "initial1"
	mockEpoch := uint64(1)

	stakeEntryList := []epochstoragetypes.StakeEntry{
		{
			Address: address + "-test",
			Chain:   specID,
		},
	}
	response := &pairingtypes.QueryProviderResponse{StakeEntries: stakeEntryList}

	// Create a new mock objects
	stateQuery := NewMockProviderPairingStatusStateQueryInf(ctrlStateQuery)
	stateQuery.EXPECT().Provider(gomock.Any(), gomock.Any(), gomock.Any()).Return(response, nil).AnyTimes()
	metricManager := NewMockProviderMetricsManagerInf(ctrlMetrics)
	// set expect for correct metric calls
	metricManager.EXPECT().SetFrozenStatus(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)
	metricManager.EXPECT().SetJailedStatus(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(0)

	// create and call freeze updater
	pau := NewProviderFreezeUpdater(stateQuery, specID, address, metricManager)
	pau.UpdateEpoch(mockEpoch)
}
