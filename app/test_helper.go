package app

import (
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

// Setup a new App for testing purposes
func TestSetup() (*LavaApp, sdk.Context) {
	db := tmdb.NewMemDB()
	encoding := MakeEncodingConfig()
	app := New(log.NewNopLogger(), db, nil, true, map[int64]bool{}, DefaultNodeHome, 5, encoding, simapp.EmptyAppOptions{})
	return app, app.NewContext(true, tmproto.Header{})
}
