package types

import (
	"testing"

	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/stretchr/testify/require"
)

func TestExtractSignerAddressFromBadge(t *testing.T) {
	pkey, addr := sigs.GenerateFloatingKey()

	tests := []struct {
		name          string
		predefinedSig []byte
	}{
		{"badge_with_nil_sig", nil},
		{"badge_with_dummy_sig", []byte{0x12, 0x34}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			badge := CreateBadge(100, 0, addr, "lava", tt.predefinedSig)
			sig, err := sigs.Sign(pkey, *badge)
			require.NoError(t, err)
			badge.ProjectSig = sig

			extractedAddr, err := sigs.ExtractSignerAddress(*badge)
			require.NoError(t, err)
			require.Equal(t, addr, extractedAddr)
		})
	}
}
