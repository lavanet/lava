package app

import (
	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Setup a new App for testing purposes
func TestSetup() (*LavaApp, sdk.Context) {
	db := dbm.NewMemDB()
	encoding := MakeEncodingConfig()
	app := New(log.NewNopLogger(), db, nil, true, map[int64]bool{}, DefaultNodeHome, 5, encoding, sims.EmptyAppOptions{})
	return app, app.NewContext(true)
}
