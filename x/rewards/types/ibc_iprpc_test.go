package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/utils/common/types"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/stretchr/testify/require"
)

// TestIprpcMemoIsEqual tests IprpcMemo method: IsEqual
func TestIprpcMemoIsEqual(t *testing.T) {
	memo := types.IprpcMemo{Creator: "creator", Spec: "spec", Duration: 3}
	template := []struct {
		name    string
		memo    types.IprpcMemo
		isEqual bool
	}{
		{"equal", types.IprpcMemo{Creator: "creator", Spec: "spec", Duration: 3}, true},
		{"different creator", types.IprpcMemo{Creator: "creator2", Spec: "spec", Duration: 3}, false},
		{"different spec", types.IprpcMemo{Creator: "creator", Spec: "spec2", Duration: 3}, false},
		{"different duration", types.IprpcMemo{Creator: "creator", Spec: "spec", Duration: 2}, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isEqual, memo.IsEqual(tt.memo))
		})
	}
}

// TestPendingIbcIprpcFundIsEqual tests PendingIbcIprpcFund method: IsEqual
func TestPendingIbcIprpcFundIsEqual(t *testing.T) {
	piif := types.PendingIbcIprpcFund{
		Index: 1, Creator: "creator", Spec: "spec", Duration: 3, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Expiry: 10,
	}
	template := []struct {
		name    string
		piif    types.PendingIbcIprpcFund
		isEqual bool
	}{
		{"equal", types.PendingIbcIprpcFund{Index: 1, Creator: "creator", Spec: "spec", Duration: 3, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Expiry: 10}, true},
		{"different index", types.PendingIbcIprpcFund{Index: 2, Creator: "creator", Spec: "spec", Duration: 3, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Expiry: 10}, false},
		{"different creator", types.PendingIbcIprpcFund{Index: 1, Creator: "creator2", Spec: "spec", Duration: 3, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Expiry: 10}, false},
		{"different spec", types.PendingIbcIprpcFund{Index: 1, Creator: "creator", Spec: "spec2", Duration: 3, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Expiry: 10}, false},
		{"different duration", types.PendingIbcIprpcFund{Index: 1, Creator: "creator", Spec: "spec", Duration: 2, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Expiry: 10}, false},
		{"different fund", types.PendingIbcIprpcFund{Index: 1, Creator: "creator", Spec: "spec", Duration: 3, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt().AddRaw(1)), Expiry: 10}, false},
		{"different expiry", types.PendingIbcIprpcFund{Index: 1, Creator: "creator", Spec: "spec", Duration: 3, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Expiry: 12}, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isEqual, piif.IsEqual(tt.piif))
		})
	}
}

// TestPendingIbcIprpcFundIsEmpty tests PendingIbcIprpcFund method: IsEmpty
func TestPendingIbcIprpcFundIsEmpty(t *testing.T) {
	template := []struct {
		name    string
		piif    types.PendingIbcIprpcFund
		isEmpty bool
	}{
		{"empty", types.PendingIbcIprpcFund{}, true},
		{"not empty", types.PendingIbcIprpcFund{Index: 1}, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isEmpty, tt.piif.IsEmpty())
		})
	}
}

// TestPendingIbcIprpcFundIsValid tests PendingIbcIprpcFund method: IsValid
func TestPendingIbcIprpcFundIsValid(t *testing.T) {
	template := []struct {
		name    string
		piif    types.PendingIbcIprpcFund
		isValid bool
	}{
		{"valid", types.PendingIbcIprpcFund{Expiry: 1, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Duration: 1}, true},
		{"empty", types.PendingIbcIprpcFund{}, false},
		{"invalid expiry", types.PendingIbcIprpcFund{Expiry: 0, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Duration: 1}, false},
		{"invalid fund amount", types.PendingIbcIprpcFund{Expiry: 1, Fund: sdk.NewCoin(commontypes.TokenDenom, math.ZeroInt()), Duration: 1}, false},
		{"invalid duration", types.PendingIbcIprpcFund{Expiry: 1, Fund: sdk.NewCoin(commontypes.TokenDenom, math.OneInt()), Duration: 0}, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isValid, tt.piif.IsValid())
		})
	}
}
