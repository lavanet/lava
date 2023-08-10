package app

import (
	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Setup a new App for testing purposes
func TestSetup() (*LavaApp, sdk.Context) {
	db := tmdb.NewMemDB()
	encoding := MakeEncodingConfig()
	app := New(log.NewNopLogger(), db, nil, true, map[int64]bool{}, DefaultNodeHome, 5, encoding, sims.EmptyAppOptions{})
	return app, app.NewContext(true, tmproto.Header{})
}
