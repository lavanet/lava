package keeper

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/x/spec/client/utils"
	"github.com/lavanet/lava/x/spec/keeper"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

func SpecKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	keep, ctx, err := specKeeper()
	require.NoError(t, err)
	return keep, ctx
}

func specKeeper() (*keeper.Keeper, sdk.Context, error) {
	storeKey := sdk.NewKVStoreKey(spectypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(spectypes.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, sdk.StoreTypeMemory, nil)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		return nil, sdk.Context{}, err
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		spectypes.Amino,
		storeKey,
		memStoreKey,
		"SpecParams",
	)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, spectypes.DefaultParams())

	return k, ctx, nil
}

func GetASpec(specIndex string) (spectypes.Spec, error) {
	keeper, ctx, err := specKeeper()
	if err != nil {
		return spectypes.Spec{}, err
	}
	getToTopMostPath := "../../"
	proposalFile := "./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmoswasm.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_cosmossdk_full.json,./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_cosmoshub.json,./cookbook/specs/spec_add_lava.json,./cookbook/specs/spec_add_osmosis.json,./cookbook/specs/spec_add_fantom.json,./cookbook/specs/spec_add_celo.json,./cookbook/specs/spec_add_optimism.json,./cookbook/specs/spec_add_arbitrum.json,./cookbook/specs/spec_add_starknet.json,./cookbook/specs/spec_add_aptos.json,./cookbook/specs/spec_add_juno.json,./cookbook/specs/spec_add_polygon.json,./cookbook/specs/spec_add_evmos.json,./cookbook/specs/spec_add_base.json,./cookbook/specs/spec_add_canto.json,./cookbook/specs/spec_add_sui.json,./cookbook/specs/spec_add_solana.json,./cookbook/specs/spec_add_bsc.json,./cookbook/specs/spec_add_axelar.json,./cookbook/specs/spec_add_avalanche.json,./cookbook/specs/spec_add_fvm.json"
	for _, fileName := range strings.Split(proposalFile, ",") {
		proposal := utils.SpecAddProposalJSON{}

		contents, err := os.ReadFile(getToTopMostPath + fileName)
		if err != nil {
			return spectypes.Spec{}, err
		}

		err = codec.NewLegacyAmino().UnmarshalJSON(contents, &proposal)
		if err != nil {
			return spectypes.Spec{}, err
		}

		for _, spec := range proposal.Proposal.Specs {
			keeper.SetSpec(ctx, spec)
			if specIndex != spec.Index {
				continue
			}
			fullspec, err := keeper.ExpandSpec(ctx, spec)
			if err != nil {
				return spectypes.Spec{}, err
			}
			return fullspec, nil
		}
	}
	return spectypes.Spec{}, fmt.Errorf("spec not found %s", specIndex)
}
