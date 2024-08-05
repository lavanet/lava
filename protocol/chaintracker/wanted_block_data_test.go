package chaintracker_test

import (
	fmt "fmt"
	"testing"

	chaintracker "github.com/lavanet/lava/v2/protocol/chaintracker"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestWantedBlockData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		earliestBlock    int64
		latestBlock      int64
		fromBlock        int64
		toBlock          int64
		specificBlock    int64
		valid            bool
		expectedElements int
	}{
		{name: "only one saved block range 1", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: 1000, specificBlock: spectypes.NOT_APPLICABLE, valid: true, expectedElements: 1},
		{name: "only one saved block overlap", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: 1000, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "only one saved block specific from N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "only one saved block specific to N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "only one saved block specific both N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "only one saved block specific from N/A other is latest", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "only one saved block specific from N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "only one saved block specific to N/A other is latest", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.LATEST_BLOCK, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "only one saved block specific to N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.LATEST_BLOCK - 5, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "latest only one saved block specific from N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest only one saved block specific to N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest only one saved block specific both N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest only one saved block specific from N/A other is latest", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest only one saved block specific from N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest only one saved block specific to N/A other is latest", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.LATEST_BLOCK, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest only one saved block specific to N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.LATEST_BLOCK - 5, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},

		{name: "ten saved blocks range 1", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1000, specificBlock: spectypes.NOT_APPLICABLE, valid: true, expectedElements: 1},
		{name: "ten saved blocks overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1000, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "ten saved blocks specific from N/A", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "ten saved blocks specific to N/A", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "ten saved blocks specific both N/A", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "ten saved blocks specific from N/A other is latest", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "ten saved blocks specific from N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "ten saved blocks specific to N/A other is latest", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "ten saved blocks specific to N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 5, toBlock: spectypes.NOT_APPLICABLE, specificBlock: 1000, valid: true, expectedElements: 1},
		{name: "latest ten saved blocks specific from N/A", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest ten saved blocks specific to N/A", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest ten saved blocks specific both N/A", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest ten saved blocks specific from N/A other is latest", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest ten saved blocks specific from N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest ten saved blocks specific to N/A other is latest", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},
		{name: "latest ten saved blocks specific to N/A other is latest with distance", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 5, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 1},

		{name: "ten saved blocks range 5", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1004, specificBlock: spectypes.NOT_APPLICABLE, valid: true, expectedElements: 5},
		{name: "ten saved blocks range 5 the end of the list", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1005, toBlock: 1009, specificBlock: spectypes.NOT_APPLICABLE, valid: true, expectedElements: 5},
		{name: "ten saved blocks range 5 overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1004, specificBlock: 1003, valid: true, expectedElements: 5},
		{name: "ten saved blocks range 5 w specific next", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1004, specificBlock: 1006, valid: true, expectedElements: 6},
		{name: "ten saved blocks range 5 w specific", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1004, specificBlock: 1007, valid: true, expectedElements: 6},
		{name: "ten saved blocks range 5 w specific before", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1002, toBlock: 1006, specificBlock: 1000, valid: true, expectedElements: 6},
		{name: "ten saved blocks range 5 w specific prev", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1002, toBlock: 1006, specificBlock: 1001, valid: true, expectedElements: 6},
		{name: "ten saved blocks range 10", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1009, specificBlock: spectypes.NOT_APPLICABLE, valid: true, expectedElements: 10},

		{name: "latest ten saved blocks range 5", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 9, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: spectypes.NOT_APPLICABLE, valid: true, expectedElements: 5},
		{name: "latest ten saved blocks range 5 the end of the list", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 4, toBlock: spectypes.LATEST_BLOCK - 1, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 5},
		{name: "latest ten saved blocks range 5 overlap the end of the list", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 4, toBlock: spectypes.LATEST_BLOCK - 1, specificBlock: spectypes.LATEST_BLOCK - 2, valid: true, expectedElements: 4},
		{name: "latest ten saved blocks range 5 overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 5, toBlock: spectypes.LATEST_BLOCK - 1, specificBlock: spectypes.LATEST_BLOCK - 2, valid: true, expectedElements: 5},
		{name: "latest ten saved blocks range 5 w specific next", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 9, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: spectypes.LATEST_BLOCK - 4, valid: true, expectedElements: 6},
		{name: "latest ten saved blocks range 5 w specific", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 9, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: spectypes.LATEST_BLOCK - 2, valid: true, expectedElements: 6},
		{name: "latest ten saved blocks range 5 w specific before", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 6, toBlock: spectypes.LATEST_BLOCK - 2, specificBlock: spectypes.LATEST_BLOCK - 8, valid: true, expectedElements: 6},
		{name: "latest ten saved blocks range 5 w specific prev", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 6, toBlock: spectypes.LATEST_BLOCK - 2, specificBlock: spectypes.LATEST_BLOCK - 7, valid: true, expectedElements: 6},
		{name: "latest ten saved blocks range 9 w specific", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 9, toBlock: spectypes.LATEST_BLOCK - 1, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 10},
		{name: "latest ten saved blocks range 10 no specific", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 9, toBlock: spectypes.LATEST_BLOCK, specificBlock: spectypes.NOT_APPLICABLE, valid: true, expectedElements: 10},
		{name: "latest ten saved blocks range 10 w specific overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 9, toBlock: spectypes.LATEST_BLOCK, specificBlock: spectypes.LATEST_BLOCK, valid: true, expectedElements: 10},

		// test invalid cases
		{name: "invalid only one saved block all N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid only one saved block from/specific N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid only one saved block to/specific", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: spectypes.NOT_APPLICABLE, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid only one saved block invalid range bigger", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1001, toBlock: 1002, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid only one saved block invalid range size", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1001, toBlock: 1000, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid only one saved block invalid range smaller", earliestBlock: 1000, latestBlock: 1000, fromBlock: 999, toBlock: 1000, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid only one saved block invalid specific bigger", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: 1000, specificBlock: 1001, valid: false, expectedElements: 0},
		{name: "invalid only one saved block invalid specific smaller", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: 1000, specificBlock: 999, valid: false, expectedElements: 0},
		{name: "invalid only one saved block invalid specific bigger range N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: 1001, valid: false, expectedElements: 0},
		{name: "invalid only one saved block invalid specific smaller range N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: 999, valid: false, expectedElements: 0},
		{name: "invalid latest only one saved block invalid range smaller", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.LATEST_BLOCK - 1, toBlock: spectypes.LATEST_BLOCK, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid latest only one saved block invalid range size", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.LATEST_BLOCK - 2, toBlock: spectypes.LATEST_BLOCK, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid latest only one saved block invalid specific smaller", earliestBlock: 1000, latestBlock: 1000, fromBlock: 1000, toBlock: 1001, specificBlock: spectypes.LATEST_BLOCK - 1, valid: false, expectedElements: 0},
		{name: "invalid latest only one saved block invalid specific smaller range N/A", earliestBlock: 1000, latestBlock: 1000, fromBlock: spectypes.NOT_APPLICABLE, toBlock: 1000, specificBlock: spectypes.LATEST_BLOCK - 1, valid: false, expectedElements: 0},

		{name: "invalid specific ten saved blocks range 5", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1004, specificBlock: 999, valid: false, expectedElements: 0},
		{name: "invalid specific ten saved blocks range 5 the end of the list", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1005, toBlock: 1009, specificBlock: 999, valid: false, expectedElements: 0},
		{name: "invalid specific ten saved blocks range 10", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1009, specificBlock: 999, valid: false, expectedElements: 0},
		{name: "invalid specific latest ten saved blocks range 5", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1004, specificBlock: spectypes.LATEST_BLOCK - 10, valid: false, expectedElements: 0},
		{name: "invalid specific latest ten saved blocks range 5 the end of the list", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1005, toBlock: 1009, specificBlock: spectypes.LATEST_BLOCK - 10, valid: false, expectedElements: 0},
		{name: "invalid specific latest ten saved blocks range 10", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1000, toBlock: 1009, specificBlock: spectypes.LATEST_BLOCK - 10, valid: false, expectedElements: 0},
		{name: "invalid range ten saved blocks smaller no range overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: 900, toBlock: 905, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid range ten saved blocks smaller with range overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: 999, toBlock: 1004, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid range ten saved blocks bigger no range overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1010, toBlock: 1015, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid range ten saved blocks bigger with range overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: 1007, toBlock: 1012, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid range latest ten saved blocks smaller no range overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 20, toBlock: spectypes.LATEST_BLOCK - 15, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
		{name: "invalid range latest ten saved blocks smaller with range overlap", earliestBlock: 1000, latestBlock: 1009, fromBlock: spectypes.LATEST_BLOCK - 10, toBlock: spectypes.LATEST_BLOCK - 5, specificBlock: spectypes.NOT_APPLICABLE, valid: false, expectedElements: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantedBlocksData := chaintracker.WantedBlocksData{}
			err := wantedBlocksData.New(tt.fromBlock, tt.toBlock, tt.specificBlock, tt.latestBlock, tt.earliestBlock)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			iterationIndexes := wantedBlocksData.IterationIndexes()
			if tt.valid {
				require.Greater(t, len(iterationIndexes), 0)
				require.GreaterOrEqual(t, tt.latestBlock+1-tt.earliestBlock, int64(len(iterationIndexes)), fmt.Sprintf("%+v", iterationIndexes))
				require.GreaterOrEqual(t, tt.latestBlock+1-tt.earliestBlock, int64(iterationIndexes[len(iterationIndexes)-1]), fmt.Sprintf("%+v", iterationIndexes))
				require.Equal(t, len(iterationIndexes), tt.expectedElements, fmt.Sprintf("%+v", iterationIndexes))
			}
			for i := 0; i < len(iterationIndexes)-2; i++ {
				require.Less(t, iterationIndexes[i], iterationIndexes[i+1], fmt.Sprintf("%+v", iterationIndexes))
			}
		})
	}
}
