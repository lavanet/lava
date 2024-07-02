package types_test

import (
	"testing"

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
