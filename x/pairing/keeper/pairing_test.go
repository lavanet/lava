package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/stretchr/testify/require"
)

func TestPairingUniqueness(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	var balance int64 = 10000
	stake := balance / 10

	consumer1 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer1, spec, stake, false)
	consumer2 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer2, spec, stake, false)

	providers := []common.Account{}
	for i := 1; i <= 1000; i++ {
		provider := common.CreateNewAccount(ctx, *keepers, balance)
		common.StakeAccount(t, ctx, *keepers, *servers, provider, spec, stake, true)
		providers = append(providers, provider)
	}

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	providers1, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)

	providers2, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer2.Addr)
	require.Nil(t, err)

	require.Equal(t, len(providers1), len(providers2))

	diffrent := false

	for _, provider := range providers1 {
		found := false
		for _, provider2 := range providers2 {
			if provider.Address == provider2.Address {
				found = true
			}
		}
		if !found {
			diffrent = true
		}
	}

	require.True(t, diffrent)

}
