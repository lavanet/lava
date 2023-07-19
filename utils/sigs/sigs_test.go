package sigs

import (
	"testing"

	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestExtractSignerAddressFromBadge(t *testing.T) {
	pkey, addr := GenerateFloatingKey()

	tests := []struct {
		name          string
		predefinedSig []byte
	}{
		{"badge_with_nil_sig", nil},
		{"badge_with_dummy_sig", []byte{0x12, 0x34}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			badge := pairingtypes.CreateBadge(100, 0, addr, "lava", tt.predefinedSig)
			sig, err := Sign(pkey, *badge)
			require.Nil(t, err)
			badge.ProjectSig = sig

			extractedAddr, err := ExtractSignerAddressFromBadge(*badge)
			require.Nil(t, err)
			require.Equal(t, addr, extractedAddr)
		})
	}
}
