package score

import (
	sdkerrors "cosmossdk.io/errors"
)

var TimeConflictingScoresError = sdkerrors.New("TimeConflictingScoreStoreError", 5183, "ScoreStore has a more recent sample than the one provided")
