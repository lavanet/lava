package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/v4/x/spec/client/utils"
	"github.com/lavanet/lava/v4/x/spec/keeper"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

func SpecKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	keep, ctx, err := specKeeper()
	require.NoError(t, err)
	return keep, ctx
}

func specKeeper() (*keeper.Keeper, sdk.Context, error) {
	storeKey := storetypes.NewKVStoreKey(spectypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(spectypes.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
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

func decodeProposal(path string) (utils.SpecAddProposalJSON, error) {
	proposal := utils.SpecAddProposalJSON{}
	contents, err := os.ReadFile(path)
	if err != nil {
		return proposal, err
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields() // This will make the unmarshal fail if there are unused fields

	err = decoder.Decode(&proposal)
	return proposal, err
}

func GetSpecsFromPath(path string, specIndex string, ctxArg *sdk.Context, keeper *keeper.Keeper) (specRet spectypes.Spec, err error) {
	var ctx sdk.Context
	if keeper == nil || ctxArg == nil {
		keeper, ctx, err = specKeeper()
		if err != nil {
			return spectypes.Spec{}, err
		}
	} else {
		ctx = *ctxArg
	}

	// Split the string by "," if we have a spec with dependencies we need to first load the dependencies.. for example:
	// ibc.json, cosmossdk.json, lava.json.
	files := strings.Split(path, ",")
	for _, fileName := range files {
		trimmedFileName := strings.TrimSpace(fileName)
		spec, err := GetSpecFromPath(trimmedFileName, specIndex, &ctx, keeper)
		if err == nil {
			return spec, nil
		}
	}
	return spectypes.Spec{}, fmt.Errorf("spec not found %s", specIndex)
}

func GetSpecFromPath(path string, specIndex string, ctxArg *sdk.Context, keeper *keeper.Keeper) (specRet spectypes.Spec, err error) {
	var ctx sdk.Context
	if keeper == nil || ctxArg == nil {
		keeper, ctx, err = specKeeper()
		if err != nil {
			return spectypes.Spec{}, err
		}
	} else {
		ctx = *ctxArg
	}

	proposal, err := decodeProposal(path)
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
	return spectypes.Spec{}, fmt.Errorf("spec not found %s", path)
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
		spec, err := GetSpecFromPath(getToTopMostPath+proposalDirectory+fileName, specIndex, &ctx, keeper)
		if err == nil {
			return spec, nil
		}
	}
	return spectypes.Spec{}, fmt.Errorf("spec not found %s", specIndex)
}
