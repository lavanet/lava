package reliabilitymanager

import (
	"strconv"
	"strings"

	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/utils"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
)

type ReliabilityManager struct {
	chainTracker *chaintracker.ChainTracker
}

func (rm *ReliabilityManager) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error) {
	return rm.chainTracker.GetLatestBlockData(fromBlock, toBlock, specificBlock)
}

func (rm *ReliabilityManager) GetLatestBlockNum() int64 {
	return rm.chainTracker.GetLatestBlockNum()
}

func NewReliabilityManager(chainTracker *chaintracker.ChainTracker) *ReliabilityManager {
	rm := &ReliabilityManager{}
	rm.chainTracker = chainTracker
	return rm
}

type VoteParams struct {
	CloseVote      bool
	ChainID        string
	ApiURL         string
	RequestData    []byte
	RequestBlock   uint64
	Voters         []string
	ConnectionType string
	ApiInterface   string
	VoteDeadline   uint64
	VoteID         string
}

func (vp *VoteParams) GetCloseVote() bool {
	if vp == nil {
		// default returns false
		return false
	}
	return vp.CloseVote
}

func BuildVoteParamsFromDetectionEvent(event terderminttypes.Event) (*VoteParams, error) {
	attributes := map[string]string{}
	for _, attribute := range event.Attributes {
		attributes[string(attribute.Key)] = string(attribute.Value)
	}
	voteID, ok := attributes["voteID"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	chainID, ok := attributes["chainID"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	apiURL, ok := attributes["apiURL"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	requestData_str, ok := attributes["requestData"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	requestData := []byte(requestData_str)

	connectionType, ok := attributes["connectionType"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	apiInterface, ok := attributes["apiInterface"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	num_str, ok := attributes["requestBlock"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	requestBlock, err := strconv.ParseUint(num_str, 10, 64)
	if err != nil {
		return nil, utils.LavaFormatError("vote requested block could not be parsed", err, &map[string]string{"requested block": num_str, "voteID": voteID})

	}
	num_str, ok = attributes["voteDeadline"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	voteDeadline, err := strconv.ParseUint(num_str, 10, 64)
	if err != nil {
		return nil, utils.LavaFormatError("vote deadline could not be parsed", err, &map[string]string{"deadline": num_str, "voteID": voteID})
	}
	voters_st, ok := attributes["voters"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	voters := strings.Split(voters_st, ",")
	voteParams := &VoteParams{
		ChainID:        chainID,
		ApiURL:         apiURL,
		RequestData:    requestData,
		RequestBlock:   requestBlock,
		Voters:         voters,
		CloseVote:      false,
		ConnectionType: connectionType,
		ApiInterface:   apiInterface,
		VoteDeadline:   voteDeadline,
		VoteID:         voteID,
	}
	return voteParams, nil
}
