package chainsentry

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var ErrorFailedToFetchLatestBlock = sdkerrors.New("Error FailedToFetchLatestBlock", 1001, "Failed to fetch latest block from node")
