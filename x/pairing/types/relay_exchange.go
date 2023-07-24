package types

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/lavanet/lava/utils"
)

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

func (re RelayExchange) PrepareForSignature() []byte {
	metadataBytes := make([]byte, 0)
	for _, metadata := range re.Reply.GetMetadata() {
		data, err := metadata.Marshal()
		if err != nil {
			utils.LavaFormatError("metadata can't be marshaled to bytes", err)
		}
		metadataBytes = append(metadataBytes, data...)
	}
	// we remove the salt from the signature because it can be different
	re.Request.RelayData.Salt = []byte{}
	return bytes.Join([][]byte{re.Reply.GetData(), []byte(re.Request.RelayData.String()), metadataBytes}, nil)
}

func (re RelayExchange) HashCount() int {
	return 2
}

func (rp RelayPrivateData) GetContentHashData() []byte {
	requestBlockBytes := make([]byte, 8)
	metadataBytes := make([]byte, 0)
	metadata := rp.Metadata
	for _, metadataEntry := range metadata {
		metadataBytes = append(metadataBytes, []byte(metadataEntry.Name+metadataEntry.Value)...)
	}
	binary.LittleEndian.PutUint64(requestBlockBytes, uint64(rp.RequestBlock))
	addon := rp.Addon
	if addon == nil {
		addon = []string{}
	}
	msgData := bytes.Join([][]byte{metadataBytes, []byte(strings.Join(addon, "")), []byte(rp.ApiInterface), []byte(rp.ConnectionType), []byte(rp.ApiUrl), rp.Data, requestBlockBytes, rp.Salt}, nil)
	return msgData
}
