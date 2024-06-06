package keeper_test

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/stretchr/testify/require"
)

// TestParseIprpcOverIbcMemo tests the behavior of OnRecvPacket() for different memos:
// 0. empty memo -> "not an iprpc memo" error
// 1. non-JSON memo -> "not an iprpc memo" error
// 2. JSON memo without "iprpc" tag -> "not an iprpc memo" error
// 3. valid JSON memo with "iprpc" tag -> happy flow
// 4. invalid JSON memo with "iprpc" tag (invalid/missing values) -> returns error (multiple cases)
func TestParseIprpcOverIbcMemo(t *testing.T) {
	ts := newTester(t, false)
	memos := []string{
		"",
		"blabla",
		`{
			"client": "Bruce",
		    "duration": 3	
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "",
			  "spec": "mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "mockspec",
			  "spec": "mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "spec": "mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "other-mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec",
			  "duration": -3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec"
			}
		}`,
	}

	const (
		EMPTY = iota
		NOT_JSON
		JSON_NO_IPRPC
		VALID_JSON_IPRPC
		EMPTY_CREATOR_JSON_IPRPC
		CREATOR_IS_SPEC_JSON_IPRPC
		MISSING_CREATOR_JSON_IPRPC
		INVALID_SPEC_JSON_IPRPC
		MISSING_SPEC_JSON_IPRPC
		INVALID_DURATION_JSON_IPRPC
		MISSING_DURATION_JSON_IPRPC
	)

	testCases := []struct {
		name         string
		memoInd      int
		expectError  *sdkerrors.Error
		expectedMemo types.IprpcMemo
	}{
		{
			name:         "empty memo",
			memoInd:      EMPTY,
			expectError:  types.ErrMemoNotIprpcOverIbc,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "memo not json",
			memoInd:      NOT_JSON,
			expectError:  types.ErrMemoNotIprpcOverIbc,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "memo json that is not iprpc",
			memoInd:      JSON_NO_IPRPC,
			expectError:  types.ErrMemoNotIprpcOverIbc,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "memo iprpc json valid",
			memoInd:      VALID_JSON_IPRPC,
			expectError:  nil,
			expectedMemo: types.IprpcMemo{Creator: "my-moniker", Spec: "mockspec", Duration: 3},
		},
		{
			name:         "invalid memo iprpc json - invalid creator - empty creator",
			memoInd:      EMPTY_CREATOR_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - invalid creator - creator is named like on-chain spec",
			memoInd:      CREATOR_IS_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - missing creator",
			memoInd:      MISSING_CREATOR_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - invalid spec",
			memoInd:      INVALID_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - missing spec",
			memoInd:      MISSING_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - invalid duration",
			memoInd:      INVALID_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - missing duration",
			memoInd:      MISSING_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			data := ts.createIbcTransferPacketData(memos[tt.memoInd])
			memo, err := ts.Keepers.Rewards.ExtractIprpcMemoFromPacket(ts.Ctx, data)
			require.True(t, tt.expectError.Is(err))
			require.True(t, memo.IsEqual(tt.expectedMemo))
		})
	}
}
