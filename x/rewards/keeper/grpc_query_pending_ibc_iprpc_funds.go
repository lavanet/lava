package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) PendingIbcIprpcFunds(goCtx context.Context, req *types.QueryPendingIbcIprpcFundsRequest) (*types.QueryPendingIbcIprpcFundsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	piifs := []types.PendingIbcIprpcFund{}
	infos := []types.PendingIbcIprpcFundInfo{}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// get the relevant PendingIbcIprpcFund objects by filter
	if index, err := strconv.ParseUint(req.Filter, 10, 64); err == nil {
		// index filter
		piif, found := k.GetPendingIbcIprpcFund(ctx, index)
		if !found {
			return nil, fmt.Errorf("PendingIbcIprpcFund not found with index %d", index)
		}
		piifs = append(piifs, piif)
	} else if _, found := k.specKeeper.GetSpec(ctx, req.Filter); found {
		// spec filter
		allPiifs := k.GetAllPendingIbcIprpcFund(ctx)
		for _, piif := range allPiifs {
			if piif.Spec == req.Filter {
				piifs = append(piifs, piif)
			}
		}
	} else if req.Filter != "" {
		// creator filter
		allPiifs := k.GetAllPendingIbcIprpcFund(ctx)
		for _, piif := range allPiifs {
			if piif.Creator == req.Filter {
				piifs = append(piifs, piif)
			}
		}
	} else {
		// no filter
		piifs = k.GetAllPendingIbcIprpcFund(ctx)
	}

	// construct PendingIbcIprpcFundInfo list (calculate cost to apply)
	if len(piifs) == 0 {
		return nil, fmt.Errorf("PendingIbcIprpcFund not found with filter %s", req.Filter)
	}

	for _, piif := range piifs {
		cost := k.CalcPendingIbcIprpcFundMinCost(ctx, piif)
		infos = append(infos, types.PendingIbcIprpcFundInfo{PendingIbcIprpcFund: piif, Cost: cost})
	}

	return &types.QueryPendingIbcIprpcFundsResponse{PendingIbcIprpcFundsInfo: infos}, nil
}
