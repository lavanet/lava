package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	ibctypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	specs := m.keeper.GetAllSpec(ctx)
	for _, spec := range specs {
		spec.Name = strings.ToLower(spec.Name)
		m.keeper.SetSpec(ctx, spec)
	}
	return nil
}

func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	params := m.keeper.GetParams(ctx)
	params.AllowlistedExpeditedMsgs = append(params.AllowlistedExpeditedMsgs,
		proto.MessageName(&ibctypes.MsgRecoverClient{}),
		proto.MessageName(&ibctypes.MsgUpgradeClient{}),
	)
	m.keeper.SetParams(ctx, params)
	return nil
}
