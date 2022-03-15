package relayer

import (
	"context"
	"errors"

	"github.com/lavanet/lava/x/spec/types"
)

func getSpec(ctx context.Context, queryClient types.QueryClient, specId int) (*types.Spec, map[string]types.ServiceApi, error) {
	//
	// SPEC/CU
	// TODO: move to service that checks for updates every block
	allSpecs, err := queryClient.SpecAll(ctx, &types.QueryAllSpecRequest{})
	if err != nil {
		return nil, nil, err
	}
	if len(allSpecs.Spec) == 0 || len(allSpecs.Spec) <= specId {
		return nil, nil, errors.New("bad specId or no specs found")
	}
	curSpec := allSpecs.Spec[specId]
	serverApis := map[string]types.ServiceApi{}
	for _, api := range curSpec.Apis {
		serverApis[api.Name] = api
	}

	return &curSpec, serverApis, nil
}
