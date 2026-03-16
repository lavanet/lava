package chainlib

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestIsArchiveRequest(t *testing.T) {
	t.Run("no extensions returns false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		msg := NewMockChainMessage(ctrl)
		msg.EXPECT().GetExtensions().Return(nil)
		require.False(t, IsArchiveRequest(msg))
	})

	t.Run("archive extension returns true", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		msg := NewMockChainMessage(ctrl)
		msg.EXPECT().GetExtensions().Return([]*spectypes.Extension{{Name: extensionslib.ArchiveExtension}})
		require.True(t, IsArchiveRequest(msg))
	})

	t.Run("unrelated extension returns false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		msg := NewMockChainMessage(ctrl)
		msg.EXPECT().GetExtensions().Return([]*spectypes.Extension{{Name: "other"}})
		require.False(t, IsArchiveRequest(msg))
	})

	t.Run("archive among multiple extensions returns true", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		msg := NewMockChainMessage(ctrl)
		msg.EXPECT().GetExtensions().Return([]*spectypes.Extension{
			{Name: "other"},
			{Name: extensionslib.ArchiveExtension},
		})
		require.True(t, IsArchiveRequest(msg))
	})
}

func TestIsDebugOrTraceRequest(t *testing.T) {
	cases := []struct {
		addon    string
		expected bool
	}{
		{"debug", true},
		{"trace", true},
		{"", false},
		{"eth", false},
		{"debugx", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("addon="+tc.addon, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			msg := NewMockChainMessage(ctrl)
			msg.EXPECT().GetApiCollection().Return(&spectypes.ApiCollection{
				CollectionData: spectypes.CollectionData{AddOn: tc.addon},
			})
			require.Equal(t, tc.expected, IsDebugOrTraceRequest(msg))
		})
	}
}

func TestIsBatchRequest(t *testing.T) {
	t.Run("batch returns true", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		msg := NewMockChainMessage(ctrl)
		msg.EXPECT().IsBatch().Return(true)
		require.True(t, IsBatchRequest(msg))
	})

	t.Run("non-batch returns false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		msg := NewMockChainMessage(ctrl)
		msg.EXPECT().IsBatch().Return(false)
		require.False(t, IsBatchRequest(msg))
	})
}
