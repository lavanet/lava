package lavasession

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type QoSInfo struct {
	LastQoSReport      *pairingtypes.QualityOfServiceReport
	LatencyScoreList   []sdk.Dec
	SyncScoreSum       int64
	TotalSyncScore     int64
	TotalRelays        uint64
	AnsweredRelays     uint64
	ConsecutiveTimeOut uint64
}

type ClientSession struct {
	CuSum                 uint64
	QoSInfo               QoSInfo
	SessionId             int64
	Client                *RelayerClientWrapper
	Lock                  utils.LavaMutex
	RelayNum              uint64
	LatestBlock           int64
	FinalizedBlocksHashes map[int64]string
	Endpoint              *Endpoint
	blocklisted           bool // if session lost sync we blacklist it.
}

type Endpoint struct {
	Addr               string
	Enabled            bool
	Client             *pairingtypes.RelayerClient
	ConnectionRefusals uint64
}

type RelayerClientWrapper struct {
	Lock             utils.LavaMutex
	Acc              string //public lava address
	Endpoints        []*Endpoint
	Sessions         map[int64]*ClientSession
	MaxComputeUnits  uint64
	UsedComputeUnits uint64
	ReliabilitySent  bool
	PairingEpoch     uint64
}
