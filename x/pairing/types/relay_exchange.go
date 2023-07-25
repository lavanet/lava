package types

import (
	"bytes"
	"strings"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
)

// RelayExchange consists a relay request and its corresponding response
type RelayExchange struct {
	Request RelayRequest
	Reply   RelayReply
}

func NewRelayExchange(req RelayRequest, res RelayReply) RelayExchange {
	return RelayExchange{Request: req, Reply: res}
}

func (re RelayExchange) GetSignature() []byte {
	return re.Reply.Sig
}

func (re RelayExchange) DataToSign() []byte {
	re.Reply.Sig = nil
	var metadataBytes []byte
	for _, metadata := range re.Reply.GetMetadata() {
		data, err := metadata.Marshal()
		if err != nil {
			utils.LavaFormatError("metadata can't be marshaled to bytes", err)
		}
		metadataBytes = append(metadataBytes, data...)
	}
	// we remove the salt from the signature because it can be different
	re.Request.RelayData.Salt = nil
	msgParts := [][]byte{
		re.Reply.GetData(),
		[]byte(re.Request.RelayData.String()),
		metadataBytes,
	}
	return bytes.Join(msgParts, nil)
}

func (re RelayExchange) HashRounds() int {
	return 2
}

func (rp RelayPrivateData) GetContentHashData() []byte {
	var metadataBytes []byte
	for _, metadataEntry := range rp.Metadata {
		metadataBytes = append(metadataBytes, []byte(metadataEntry.Name+metadataEntry.Value)...)
	}
	requestBlockBytes := sigs.Encode(uint64(rp.RequestBlock))
	msgParts := [][]byte{
		metadataBytes,
		[]byte(strings.Join(rp.Addon, "")),
		[]byte(rp.ApiInterface),
		[]byte(rp.ConnectionType),
		[]byte(rp.ApiUrl),
		rp.Data,
		requestBlockBytes,
		rp.Salt,
	}
	msgData := bytes.Join(msgParts, nil)
	return msgData
}
