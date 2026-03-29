package specfetcher

import (
	"context"
	"fmt"

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

	getBaseSpec := func(_ context.Context, idx string) (types.Spec, bool) {
		s, found := specs[idx]
		return s, found
	}

	depends := map[string]bool{index: true}
	inherit := map[string]bool{}

	_, err := types.DoExpandSpec(context.Background(), &spec, depends, &inherit, spec.Index, getBaseSpec)
	if err != nil {
		return nil, fmt.Errorf("spec expand failed: %w", err)
	}

	return &spec, nil
}
