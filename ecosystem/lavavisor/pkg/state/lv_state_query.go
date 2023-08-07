package lvstatetracker

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

const (
	CacheMaxCost                = 10 * 1024 // 10K cost
	CacheNumCounters            = 100000    // expect 10K items
	DefaultTimeToLiveExpiration = 30 * time.Minute
	PairingRespKey              = "pairing-resp"
	VerifyPairingRespKey        = "verify-pairing-resp"
	MaxCuResponseKey            = "max-cu-resp"
	EffectivePolicyRespKey      = "effective-policy-resp"
)

type StateQuery struct {
	ProtocolClient protocoltypes.QueryClient
}

func NewStateQuery(ctx context.Context, clientCtx client.Context) *StateQuery {
	sq := &StateQuery{}
	sq.ProtocolClient = protocoltypes.NewQueryClient(clientCtx)
	return sq
}

func (csq *StateQuery) GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error) {
	param, err := csq.ProtocolClient.Params(ctx, &protocoltypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}
	return &param.Params.Version, nil
}
