package lavasession

import (
	"sync"
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

const (
	notBlockListedConsumer = 0
	blockListedConsumer    = 1
)

type ProviderSessionsWithConsumer struct {
	Sessions      map[uint64]*SingleProviderSession
	isBlockListed uint32
	user          string
	dataByEpoch   map[uint64]*ProviderSessionsEpochData
	Lock          sync.RWMutex
}

// reads cs.BlockedEpoch atomically
func (pswc *ProviderSessionsWithConsumer) atomicWriteBlockedEpoch(blockStatus uint32) {
	atomic.StoreUint32(&pswc.isBlockListed, blockStatus)
}

// reads cs.BlockedEpoch atomically
func (pswc *ProviderSessionsWithConsumer) atomicReadBlockedEpoch() (blockStatus uint32) {
	return atomic.LoadUint32(&pswc.isBlockListed)
}

func (pswc *ProviderSessionsWithConsumer) readBlockListedAtomic() {
}

type SingleProviderSession struct {
	userSessionsParent *ProviderSessionsWithConsumer
	CuSum              uint64
	UniqueIdentifier   uint64
	Lock               sync.RWMutex
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
