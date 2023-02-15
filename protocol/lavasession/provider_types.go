package lavasession

import (
	"context"
	"fmt"
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

type RPCProviderEndpoint struct {
	NetworkAddress string `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address,omitempty"` // IP:PORT
	ChainID        string `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"`                                // spec chain identifier
	ApiInterface   string `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64 `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
	NodeUrl        string `yaml:"node-url,omitempty" json:"node-url,omitempty" mapstructure:"node-url"`
}

func (rpcpe *RPCProviderEndpoint) Key() string {
	return rpcpe.ChainID + rpcpe.ApiInterface + rpcpe.NodeUrl
}

const (
	notBlockListedConsumer = 0
	blockListedConsumer    = 1
)

// holds all of the data for a consumer for a certain epoch
type ProviderSessionsWithConsumer struct {
	Sessions      map[uint64]*SingleProviderSession
	isBlockListed uint32
	consumer      string
	epochData     *ProviderSessionsEpochData
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
	LatestRelayCu      uint64
	UniqueIdentifier   uint64
	Lock               sync.RWMutex
	Proof              *pairingtypes.RelayRequest // saves last relay request of a session as proof
	RelayNum           uint64
	PairingEpoch       uint64
}

func (sps *SingleProviderSession) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&sps.PairingEpoch)
}

func (sps *SingleProviderSession) SetPairingEpoch(epoch uint64) {
	atomic.StoreUint64(&sps.PairingEpoch, epoch)
}

func (sps *SingleProviderSession) PrepareSessionForUsage(cu uint64) error {
	// verify locked
	// verify total cu in the parent (atomic read)
	// set LatestRelayCu (verify it's 0)
	// add to parent with atomic - make sure there is no race to corrupt the total cu in the parent
	return fmt.Errorf("not implemented")
}

func (pswc *ProviderSessionsWithConsumer) GetExistingSession(sessionId uint64) (session *SingleProviderSession, err error) {
	pswc.Lock.RLock()
	defer pswc.Lock.RUnlock()
	if session, ok := pswc.Sessions[sessionId]; ok {
		return session, nil
	}
	return nil, fmt.Errorf("session does not exist")
}

type StateQuery interface {
	QueryVerifyPairing(ctx context.Context, consumer string, blockHeight uint64)
}
