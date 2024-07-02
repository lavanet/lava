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
	"github.com/lavanet/lava/x/spec/client/utils"
	"github.com/lavanet/lava/x/spec/keeper"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
		"",
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
	proposalFile := "./cookbook/specs/ibc.json,./cookbook/specs/cosmoswasm.json,./cookbook/specs/tendermint.json,./cookbook/specs/cosmossdk.json,./cookbook/specs/cosmossdk_full.json,./cookbook/specs/ethereum.json,./cookbook/specs/cosmoshub.json,./cookbook/specs/lava.json,./cookbook/specs/osmosis.json,./cookbook/specs/fantom.json,./cookbook/specs/celo.json,./cookbook/specs/optimism.json,./cookbook/specs/arbitrum.json,./cookbook/specs/starknet.json,./cookbook/specs/aptos.json,./cookbook/specs/juno.json,./cookbook/specs/polygon.json,./cookbook/specs/evmos.json,./cookbook/specs/base.json,./cookbook/specs/canto.json,./cookbook/specs/sui.json,./cookbook/specs/solana.json,./cookbook/specs/bsc.json,./cookbook/specs/axelar.json,./cookbook/specs/avalanche.json,./cookbook/specs/fvm.json"
	for _, fileName := range strings.Split(proposalFile, ",") {
		proposal := utils.SpecAddProposalWithDepositJSON{}

		contents, err := os.ReadFile(getToTopMostPath + fileName)
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
