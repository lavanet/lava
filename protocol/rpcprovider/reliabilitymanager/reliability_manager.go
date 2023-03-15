package reliabilitymanager

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/exp/slices"
)

const (
	DetectionVoteType = 0
	RevealVoteType    = 1
	CloseVoteType     = 2
)

type TxSender interface {
	SendVoteReveal(voteID string, vote *VoteData) error
	SendVoteCommitment(voteID string, vote *VoteData) error
}

type ReliabilityManager struct {
	chainTracker  *chaintracker.ChainTracker
	votes_mutex   sync.Mutex
	votes         map[string]*VoteData
	txSender      TxSender
	publicAddress string
	chainProxy    chainlib.ChainProxy
	chainParser   chainlib.ChainParser
}

func (rm *ReliabilityManager) VoteHandler(voteParams *VoteParams, nodeHeight uint64) {
	// got a vote event, handle the cases here
	voteID := voteParams.VoteID
	voteDeadline := voteParams.VoteDeadline
	if !voteParams.GetCloseVote() {
		// meaning we dont close a vote, so we should check stuff
		if voteDeadline < nodeHeight {
			// its too late to vote
			utils.LavaFormatError("Vote Event received but it's too late to vote", nil,
				&map[string]string{"deadline": strconv.FormatUint(voteDeadline, 10), "nodeHeight": strconv.FormatUint(nodeHeight, 10)})
			return
		}
	}
	rm.votes_mutex.Lock()
	defer rm.votes_mutex.Unlock()
	vote, ok := rm.votes[voteID]
	if ok {
		// we have an existing vote with this ID
		if voteParams.ParamsType == CloseVoteType {
			if voteParams.GetCloseVote() {
				// we are closing the vote, so its okay we have this voteID
				utils.LavaFormatInfo("Received Vote termination event for vote, cleared entry",
					&map[string]string{"voteID": voteID})
				delete(rm.votes, voteID)
				return
			}
			// expected to start a new vote but found an existing one
			utils.LavaFormatError("new vote Request for vote had existing entry", nil,
				&map[string]string{"voteParams": fmt.Sprintf("%+v", voteParams), "voteID": voteID, "voteData": fmt.Sprintf("%+v", vote)})
			return
		}
		utils.LavaFormatInfo(" Received Vote Reveal for vote, sending Reveal for result",
			&map[string]string{"voteID": voteID, "voteData": fmt.Sprintf("%+v", vote)})
		rm.txSender.SendVoteReveal(voteID, vote)
		return
	} else {
		// new vote
		if voteParams == nil {
			utils.LavaFormatError("vote commit Request didn't have a vote entry", nil,
				&map[string]string{"voteID": voteID})
			return
		}
		if voteParams.GetCloseVote() {
			utils.LavaFormatError("vote closing received but didn't have a vote entry", nil,
				&map[string]string{"voteID": voteID})
			return
		}
		if voteParams.ParamsType != DetectionVoteType {
			utils.LavaFormatError("new voteID without DetectionVoteType", nil,
				&map[string]string{"voteParams": fmt.Sprintf("%v", voteParams)})
			return
		}
		// try to find this provider in the jury
		found := slices.Contains(voteParams.Voters, rm.publicAddress)
		if !found {
			utils.LavaFormatInfo("new vote initiated but not for this provider to vote", nil)
			// this is a new vote but not for us
			return
		}
		// we need to send a commit, first we need to use the chainProxy and get the response
		// TODO: implement code that verified the requested block is finalized and if its not waits and tries again
		ctx := context.Background()
		chainMessage, err := rm.chainParser.ParseMsg(voteParams.ApiURL, voteParams.RequestData, voteParams.ConnectionType)
		if err != nil {
			utils.LavaFormatError("vote Request did not pass the api check on chain proxy", err,
				&map[string]string{"voteID": voteID, "chainID": voteParams.ChainID})
			return
		}
		reply, _, _, err := rm.chainProxy.SendNodeMsg(ctx, nil, chainMessage)
		if err != nil {
			utils.LavaFormatError("vote relay send has failed", err,
				&map[string]string{"ApiURL": voteParams.ApiURL, "RequestData": string(voteParams.RequestData)})
			return
		}
		nonce := rand.Int63()
		replyDataHash := sigs.HashMsg(reply.Data)
		commitHash := conflicttypes.CommitVoteData(nonce, replyDataHash)

		vote = &VoteData{RelayDataHash: replyDataHash, Nonce: nonce, CommitHash: commitHash}
		rm.votes[voteID] = vote
		utils.LavaFormatInfo("Received Vote start, sending commitment for result", &map[string]string{"voteID": voteID, "voteData": fmt.Sprintf("%+v", vote)})
		rm.txSender.SendVoteCommitment(voteID, vote)
		return
	}
}

func (rm *ReliabilityManager) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error) {
	return rm.chainTracker.GetLatestBlockData(fromBlock, toBlock, specificBlock)
}

func (rm *ReliabilityManager) GetLatestBlockNum() int64 {
	return rm.chainTracker.GetLatestBlockNum()
}

func NewReliabilityManager(chainTracker *chaintracker.ChainTracker, txSender TxSender, publicAddress string, chainProxy chainlib.ChainProxy, chainParser chainlib.ChainParser) *ReliabilityManager {
	rm := &ReliabilityManager{
		votes:         map[string]*VoteData{},
		txSender:      txSender,
		publicAddress: publicAddress,
		chainTracker:  chainTracker,
		chainProxy:    chainProxy,
		chainParser:   chainParser,
	}

	return rm
}

type VoteData struct {
	RelayDataHash []byte
	Nonce         int64
	CommitHash    []byte
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
	ParamsType     uint
}

func (vp *VoteParams) GetCloseVote() bool {
	if vp == nil {
		// default returns false
		return false
	}
	return vp.CloseVote
}

func BuildBaseVoteDataFromEvent(event terderminttypes.Event) (voteID string, voteDeadline uint64, err error) {
	attributes := map[string]string{}
	for _, attribute := range event.Attributes {
		attributes[string(attribute.Key)] = string(attribute.Value)
	}
	voteID, ok := attributes["voteID"]
	if !ok {
		return "", 0, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, &attributes)
	}
	num_str, ok := attributes["voteDeadline"]
	if !ok {
		return voteID, 0, utils.LavaFormatError("no attribute deadline", NoVoteDeadline, nil)
	}
	voteDeadline, err = strconv.ParseUint(num_str, 10, 64)
	if err != nil {
		return "", 0, utils.LavaFormatError("vote deadline could not be parsed", err, &map[string]string{"deadline": num_str, "voteID": voteID})
	}
	return voteID, voteDeadline, nil
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
		ParamsType:     DetectionVoteType,
	}
	return voteParams, nil
}
