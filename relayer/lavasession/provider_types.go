package lavasession

import (
	"sync/atomic"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type voteData struct {
	RelayDataHash []byte
	Nonce         int64
	CommitHash    []byte
}

type ProviderSessionsEpochData struct {
	UsedComputeUnits uint64
	MaxComputeUnits  uint64
	DataReliability  *pairingtypes.VRFData
	VrfPk            utils.VrfPubKey
}

type ProviderSessionsWithConsumer struct {
	Sessions      map[uint64]*SingleProviderSession
	IsBlockListed bool
	user          string
	dataByEpoch   map[uint64]*ProviderSessionsEpochData
	Lock          utils.LavaMutex
}

type SingleProviderSession struct {
	userSessionsParent *ProviderSessionsWithConsumer
	CuSum              uint64
	UniqueIdentifier   uint64
	Lock               utils.LavaMutex
	Proof              *pairingtypes.RelayRequest // saves last relay request of a session as proof
	RelayNum           uint64
	PairingEpoch       uint64
}

func (r *SingleProviderSession) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&r.PairingEpoch)
}

func (r *SingleProviderSession) SetPairingEpoch(epoch uint64) {
	atomic.StoreUint64(&r.PairingEpoch, epoch)
}
