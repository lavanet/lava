package specfetcher

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/x/spec/types"
)

// expandSpec expands a spec by resolving all its dependencies (inherited specs).
func expandSpec(specs map[string]types.Spec, index string) (*types.Spec, error) {
	spec, ok := specs[index]
	if !ok {
		availableSpecs := make([]string, 0, len(specs))
		for id := range specs {
			availableSpecs = append(availableSpecs, id)
		}
		return nil, fmt.Errorf("spec not found for chainId: %s (available specs: %v)", index, availableSpecs)
	}

	getBaseSpec := func(_ sdk.Context, idx string) (types.Spec, bool) {
		s, found := specs[idx]
		return s, found
	}

	depends := map[string]bool{index: true}
	inherit := map[string]bool{}

	ctx := sdk.Context{}
	_, err := types.DoExpandSpec(ctx, &spec, depends, &inherit, spec.Index, getBaseSpec)
	if err != nil {
		return nil, fmt.Errorf("spec expand failed: %w", err)
	}

	return &spec, nil
}
