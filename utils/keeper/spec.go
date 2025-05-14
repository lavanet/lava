package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	"github.com/lavanet/lava/v5/x/spec/client/utils"
	"github.com/lavanet/lava/v5/x/spec/keeper"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
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

	proposalDirectories := []string{
		"cookbook/specs/",
	}
	baseProposalFiles := []string{
		"ibc.json", "cosmoswasm.json", "tendermint.json", "cosmossdk.json",
		"cosmossdkv45.json", "cosmossdkv50.json", "ethereum.json", "ethermint.json", "solana.json",
	}

	// Create a map of base files for quick lookup
	baseFiles := make(map[string]struct{})
	for _, f := range baseProposalFiles {
		baseFiles[f] = struct{}{}
	}

	// Try each directory
	for _, proposalDirectory := range proposalDirectories {
		// Try base proposal files first
		for _, fileName := range baseProposalFiles {
			spec, err := GetSpecFromPath(getToTopMostPath+proposalDirectory+fileName, specIndex, &ctx, keeper)
			if err == nil {
				return spec, nil
			}
		}

		// Read all files from the proposal directory
		files, err := os.ReadDir(getToTopMostPath + proposalDirectory)
		if err != nil {
			continue // Skip to next directory if this one fails
		}

		// Try additional JSON files that aren't in baseProposalFiles
		for _, file := range files {
			fileName := file.Name()
			// Skip if not a JSON file or if it's in baseProposalFiles
			if !strings.HasSuffix(fileName, ".json") {
				continue
			}
			if _, exists := baseFiles[fileName]; exists {
				continue
			}

			spec, err := GetSpecFromPath(getToTopMostPath+proposalDirectory+fileName, specIndex, &ctx, keeper)
			if err == nil {
				return spec, nil
			}
		}
	}

	return spectypes.Spec{}, fmt.Errorf("spec not found %s", specIndex)
}
