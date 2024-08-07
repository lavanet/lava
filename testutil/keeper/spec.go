package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/v2/x/spec/client/utils"
	"github.com/lavanet/lava/v2/x/spec/keeper"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
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
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
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
		nil,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, spectypes.DefaultParams())

	return k, ctx, nil
}

func GetASpec(specIndex, getToTopMostPath string, ctxArg *sdk.Context, keeper *keeper.Keeper) (specRet spectypes.Spec, err error) {
	var ctx sdk.Context
	if keeper == nil || ctxArg == nil {
		keeper, ctx, err = specKeeper()
		if err != nil {
			return spectypes.Spec{}, err
		}
	} else {
		ctx = *ctxArg
	}
	proposalDirectory := "cookbook/specs/"
	proposalFiles := []string{
		"ibc.json", "cosmoswasm.json", "tendermint.json", "cosmossdk.json", "cosmossdk_full.json",
		"ethereum.json", "cosmoshub.json", "lava.json", "osmosis.json", "fantom.json", "celo.json",
		"optimism.json", "arbitrum.json", "starknet.json", "aptos.json", "juno.json", "polygon.json",
		"evmos.json", "base.json", "canto.json", "sui.json", "solana.json", "bsc.json", "axelar.json",
		"avalanche.json", "fvm.json", "near.json",
	}
	for _, fileName := range proposalFiles {
		proposal := utils.SpecAddProposalJSON{}

		contents, err := os.ReadFile(getToTopMostPath + proposalDirectory + fileName)
		if err != nil {
			return spectypes.Spec{}, err
		}
		decoder := json.NewDecoder(bytes.NewReader(contents))
		decoder.DisallowUnknownFields() // This will make the unmarshal fail if there are unused fields

		if err := decoder.Decode(&proposal); err != nil {
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
