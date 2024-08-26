package reliabilitymanager

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"

	terderminttypes "github.com/cometbft/cometbft/abci/types"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/protocol/chaintracker"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/rand"
	"github.com/lavanet/lava/v2/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"golang.org/x/exp/slices"
)

const (
	DetectionVoteType = 0
	RevealVoteType    = 1
	CloseVoteType     = 2
)

type TxSender interface {
	SendVoteReveal(voteID string, vote *VoteData, specID string) error
	SendVoteCommitment(voteID string, vote *VoteData, specID string) error
}

type ChainTrackerInf interface {
	GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error)
	GetLatestBlockNum() (int64, time.Time)
}

type ReliabilityManager struct {
	chainTracker  ChainTrackerInf
	votes_mutex   sync.Mutex
	votes         map[string]*VoteData
	txSender      TxSender
	publicAddress string
	chainRouter   chainlib.ChainRouter
	chainParser   chainlib.ChainParser
}

func (rm *ReliabilityManager) VoteHandler(voteParams *VoteParams, nodeHeight uint64) error {
	// got a vote event, handle the cases here
	voteID := voteParams.VoteID
	voteDeadline := voteParams.VoteDeadline
	if !voteParams.GetCloseVote() {
		// meaning we dont close a vote, so we should check stuff
		if voteDeadline < nodeHeight {
			// its too late to vote
			return utils.LavaFormatError("Vote Event received but it's too late to vote", nil,
				utils.Attribute{Key: "deadline", Value: voteDeadline},
				utils.Attribute{Key: "nodeHeight", Value: nodeHeight})
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
					utils.Attribute{Key: "voteID", Value: voteID})
				delete(rm.votes, voteID)
				return nil
			}
			// expected to start a new vote but found an existing one
			return utils.LavaFormatError("new vote Request for vote had existing entry", nil,
				utils.Attribute{Key: "voteParams", Value: voteParams}, utils.Attribute{Key: "voteID", Value: voteID}, utils.Attribute{Key: "voteData", Value: vote})
		}
		utils.LavaFormatInfo(" Received Vote Reveal for vote, sending Reveal for result",
			utils.Attribute{Key: "voteID", Value: voteID}, utils.Attribute{Key: "voteData", Value: vote})
		rm.txSender.SendVoteReveal(voteID, vote, voteParams.ChainID)
		return nil
	} else {
		// new vote
		if voteParams == nil {
			return utils.LavaFormatError("vote commit Request didn't have a vote entry", nil,
				utils.Attribute{Key: "voteID", Value: voteID})
		}
		if voteParams.GetCloseVote() {
			return utils.LavaFormatError("vote closing received but didn't have a vote entry", nil,
				utils.Attribute{Key: "voteID", Value: voteID})
		}
		if voteParams.ParamsType != DetectionVoteType {
			return utils.LavaFormatError("new voteID without DetectionVoteType", nil,
				utils.Attribute{Key: "voteParams", Value: voteParams})
		}
		// try to find this provider in the jury
		found := slices.Contains(voteParams.Voters, rm.publicAddress)
		if !found {
			utils.LavaFormatInfo("new vote initiated but not for this provider to vote")
			// this is a new vote but not for us
			return nil
		}
		// we need to send a commit, first we need to use the chainProxy and get the response
		// TODO: implement code that verified the requested block is finalized and if its not waits and tries again
		ctx := context.Background()
		chainMessage, err := rm.chainParser.ParseMsg(voteParams.ApiURL, voteParams.RequestData, voteParams.ConnectionType, nil, extensionslib.ExtensionInfo{LatestBlock: 0}) // TODO: do we have the latest block?, also we need to add extensions to vote params to report the right extension
		if err != nil {
			return utils.LavaFormatError("vote Request did not pass the api check on chain proxy", err,
				utils.Attribute{Key: "voteID", Value: voteID}, utils.Attribute{Key: "chainID", Value: voteParams.ChainID})
		}
		// TODO: get extensions and addons from the request
		replyWrapper, _, _, _, _, err := rm.chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
		if err != nil {
			return utils.LavaFormatError("vote relay send has failed", err,
				utils.Attribute{Key: "ApiURL", Value: voteParams.ApiURL}, utils.Attribute{Key: "RequestData", Value: voteParams.RequestData})
		}
		if replyWrapper == nil || replyWrapper.RelayReply == nil {
			return utils.LavaFormatError("vote relay send has failed, relayWrapper is nil", nil, utils.Attribute{Key: "ApiURL", Value: voteParams.ApiURL}, utils.Attribute{Key: "RequestData", Value: voteParams.RequestData})
		}
		reply := replyWrapper.RelayReply
		reply.Metadata, _, _ = rm.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
		nonce := rand.Int63()
		relayData := BuildRelayDataFromVoteParams(voteParams)
		relayExchange := pairingtypes.NewRelayExchange(pairingtypes.RelayRequest{RelayData: relayData}, *reply)
		replyDataHash := sigs.HashMsg(relayExchange.DataToSign())
		commitHash := conflicttypes.CommitVoteData(nonce, replyDataHash, rm.publicAddress)

		vote = &VoteData{RelayDataHash: replyDataHash, Nonce: nonce, CommitHash: commitHash}
		rm.votes[voteID] = vote
		utils.LavaFormatInfo("Received Vote start, sending commitment for result", utils.Attribute{Key: "voteID", Value: voteID}, utils.Attribute{Key: "voteData", Value: vote})
		rm.txSender.SendVoteCommitment(voteID, vote, voteParams.ChainID)
		return nil
	}
}

func (rm *ReliabilityManager) GetLatestBlockData(fromBlock, toBlock, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	return rm.chainTracker.GetLatestBlockData(fromBlock, toBlock, specificBlock)
}

func (rm *ReliabilityManager) GetLatestBlockNum() (int64, time.Time) {
	return rm.chainTracker.GetLatestBlockNum()
}

func NewReliabilityManager(chainTracker ChainTrackerInf, txSender TxSender, publicAddress string, chainRouter chainlib.ChainRouter, chainParser chainlib.ChainParser) *ReliabilityManager {
	rm := &ReliabilityManager{
		votes:         map[string]*VoteData{},
		txSender:      txSender,
		publicAddress: publicAddress,
		chainTracker:  chainTracker,
		chainRouter:   chainRouter,
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
	Metadata       []pairingtypes.Metadata
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
		attributes[attribute.Key] = attribute.Value
	}
	voteID, ok := attributes["voteID"]
	if !ok {
		return "", 0, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	num_str, ok := attributes["voteDeadline"]
	if !ok {
		return voteID, 0, utils.LavaFormatError("no attribute deadline", NoVoteDeadline)
	}
	voteDeadline, err = strconv.ParseUint(num_str, 10, 64)
	if err != nil {
		return "", 0, utils.LavaFormatError("vote deadline could not be parsed", err, utils.Attribute{Key: "deadline", Value: num_str}, utils.Attribute{Key: "voteID", Value: voteID})
	}
	return voteID, voteDeadline, nil
}

func BuildVoteParamsFromDetectionEvent(event terderminttypes.Event) (*VoteParams, error) {
	attributes := map[string]string{}
	for _, attribute := range event.Attributes {
		attributes[attribute.Key] = attribute.Value
	}
	voteID, ok := attributes["voteID"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	chainID, ok := attributes["chainID"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	apiURL, ok := attributes["apiURL"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	requestData_str, ok := attributes["requestData"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	requestData := []byte(requestData_str)

	connectionType, ok := attributes["connectionType"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	apiInterface, ok := attributes["apiInterface"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	num_str, ok := attributes["requestBlock"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}
	requestBlock, err := strconv.ParseUint(num_str, 10, 64)
	if err != nil {
		return nil, utils.LavaFormatError("vote requested block could not be parsed", err, utils.Attribute{Key: "requested block", Value: num_str}, utils.Attribute{Key: "voteID", Value: voteID})
	}

	num_str, ok = attributes["voteDeadline"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}

	metadataStr, ok := attributes["metadata"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
	}

	metadata := make([]pairingtypes.Metadata, 0)
	err = json.Unmarshal([]byte(metadataStr), &metadata)
	if err != nil {
		return nil, utils.LavaFormatError("failed unmarshaling metadata json", err, utils.Attribute{Key: "metadataStr", Value: metadataStr}, utils.Attribute{Key: "voteID", Value: voteID})
	}

	voteDeadline, err := strconv.ParseUint(num_str, 10, 64)
	if err != nil {
		return nil, utils.LavaFormatError("vote deadline could not be parsed", err, utils.Attribute{Key: "deadline", Value: num_str}, utils.Attribute{Key: "voteID", Value: voteID})
	}

	voters_st, ok := attributes["voters"]
	if !ok {
		return nil, utils.LavaFormatError("failed building BuildVoteParamsFromRevealEvent", nil, utils.Attribute{Key: "attributes", Value: attributes})
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

func BuildRelayDataFromVoteParams(voteParams *VoteParams) *pairingtypes.RelayPrivateData {
	reply := pairingtypes.RelayPrivateData{
		ConnectionType: voteParams.ConnectionType,
		ApiUrl:         voteParams.ApiURL,
		Data:           voteParams.RequestData,
		RequestBlock:   int64(voteParams.RequestBlock),
		ApiInterface:   voteParams.ApiInterface,
		Salt:           []byte{},
		Metadata:       voteParams.Metadata,
	}
	return &reply
}
