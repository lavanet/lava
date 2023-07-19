package sigs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

func GetKeyName(clientCtx client.Context) (string, error) {
	_, name, _, err := client.GetFromFields(clientCtx, clientCtx.Keyring, clientCtx.From)
	if err != nil {
		return "", err
	}

	return name, nil
}

func GetPrivKey(clientCtx client.Context, keyName string) (*btcSecp256k1.PrivateKey, error) {
	//
	// get private key
	armor, err := clientCtx.Keyring.ExportPrivKeyArmor(keyName, "")
	if err != nil {
		return nil, err
	}

	privKey, algo, err := crypto.UnarmorDecryptPrivKey(armor, "")
	if err != nil {
		return nil, err
	}
	if algo != "secp256k1" {
		return nil, errors.New("incompatible private key algorithm")
	}

	priv, _ := btcSecp256k1.PrivKeyFromBytes(btcSecp256k1.S256(), privKey.Bytes())
	return priv, nil
}

func HashMsg(msgData []byte) []byte {
	return tendermintcrypto.Sha256(msgData)
}

type Signable interface {
	GetSignature() []byte
	PrepareForSignature() []byte
}

type PrepareFunc func(interface{})

// Sign creates a signature for a struct. The prepareFunc prepares the struct before extracting the data for the signature
func Sign(pkey *btcSecp256k1.PrivateKey, data Signable) ([]byte, error) {
	msgData := data.PrepareForSignature()
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, HashMsg(msgData), false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func ExtractStructSignerAddress(pkey *btcSecp256k1.PrivateKey, data interface{}, prepareFunc PrepareFunc) (sdk.AccAddress, error) {
	if prepareFunc != nil {
		prepareFunc(data)
	}

	// Convert struct to string representation
	msgData := []byte(reflect.ValueOf(data).String())

	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, HashMsg(msgData), false)
	if err != nil {
		return nil, err
	}

	pubKey, err := RecoverPubKey(sig, HashMsg(msgData))
	if err != nil {
		return nil, err
	}

	extractedConsumerAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
	if err != nil {
		return nil, fmt.Errorf("get relay consumer address: %s", err.Error())
	}

	return extractedConsumerAddress, nil
}

func ExtractSignerAddress(in *pairingtypes.RelaySession) (sdk.AccAddress, error) {
	pubKey, err := RecoverPubKeyFromRelay(*in)
	if err != nil {
		return nil, err
	}
	extractedConsumerAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
	if err != nil {
		return nil, utils.LavaFormatError("get relay consumer address", err)
	}
	return extractedConsumerAddress, nil
}

func ExtractSignerAddressFromBadge(badge pairingtypes.Badge) (sdk.AccAddress, error) {
	sig := badge.ProjectSig
	badge.ProjectSig = nil
	hash := HashMsg([]byte(badge.String()))
	pubKey, err := RecoverPubKey(sig, hash)
	if err != nil {
		return nil, err
	}

	extractedConsumerAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
	if err != nil {
		return nil, fmt.Errorf("get relay consumer address %s", err.Error())
	}

	return extractedConsumerAddress, nil
}

func AllDataHash(relayResponse *pairingtypes.RelayReply, relayData pairingtypes.RelayPrivateData) (data_hash []byte) {
	metadataBytes := make([]byte, 0)
	for _, metadata := range relayResponse.GetMetadata() {
		data, err := metadata.Marshal()
		if err != nil {
			utils.LavaFormatError("metadata can't be marshaled to bytes", err)
		}
		metadataBytes = append(metadataBytes, data...)
	}
	// we remove the salt from the signature because it can be different
	relayData.Salt = []byte{}
	data_hash = HashMsg(bytes.Join([][]byte{relayResponse.GetData(), []byte(relayData.String()), metadataBytes}, nil))
	return
}

func DataToSignRequestResponse(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (dataToSign []byte) {
	// we do another hash here so we can verify sig without revealing the allDataHash in conflict detection tx
	return HashMsg(AllDataHash(relayResponse, *relayReq.RelayData))
}

func DataToSignResponseFinalizationData(latestBlock int64, finalizedBlocksHashes []byte, relaySession *pairingtypes.RelaySession, clientAddress sdk.AccAddress) (dataToSign []byte) {
	// sign latest_block+finalized_blocks_hashes+session_id+block_height+relay_num
	relaySessionHash := CalculateRelaySessionHashForFinalization(relaySession)
	return DataToSignResponseFinalizationDataInner(latestBlock, finalizedBlocksHashes, clientAddress, relaySessionHash)
}

func CalculateRelaySessionHashForFinalization(relaySession *pairingtypes.RelaySession) (hash []byte) {
	sessionIdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionIdBytes, relaySession.SessionId)
	blockHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(blockHeightBytes, uint64(relaySession.Epoch))
	relayNumBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(relayNumBytes, relaySession.RelayNum)
	return HashMsg(bytes.Join([][]byte{sessionIdBytes, blockHeightBytes, relayNumBytes}, nil))
}

func DataToSignResponseFinalizationDataInner(latestBlock int64, finalizedBlockHashes []byte, clientAddress sdk.AccAddress, relaySessionHash []byte) (dataToSign []byte) {
	latestBlockBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(latestBlockBytes, uint64(latestBlock))
	return HashMsg(bytes.Join([][]byte{latestBlockBytes, finalizedBlockHashes, clientAddress, relaySessionHash}, nil))
}

func SignRelayResponse(pkey *btcSecp256k1.PrivateKey, relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) ([]byte, error) {
	relayResponse.Sig = []byte{}
	dataToSign := DataToSignRequestResponse(relayResponse, relayReq)
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, dataToSign, false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func SignResponseFinalizationData(pkey *btcSecp256k1.PrivateKey, relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest, clientAddress sdk.AccAddress) ([]byte, error) {
	dataToSign := DataToSignResponseFinalizationData(relayResponse.LatestBlock, relayResponse.FinalizedBlocksHashes, relayReq.RelaySession, clientAddress)
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, dataToSign, false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func RecoverPubKey(sig []byte, msgHash []byte) (secp256k1.PubKey, error) {
	//
	// Recover public key from signature
	recPub, _, err := btcSecp256k1.RecoverCompact(btcSecp256k1.S256(), sig, msgHash)
	if err != nil {
		return nil, utils.LavaFormatError("RecoverCompact", err, utils.Attribute{
			Key: "sigLen", Value: len(sig),
		})
	}
	pk := recPub.SerializeCompressed()

	return (secp256k1.PubKey)(pk), nil
}

func RecoverPubKeyFromRelay(relay pairingtypes.RelaySession) (secp256k1.PubKey, error) {
	signature := relay.Sig // save sig
	msgData := relay.PrepareForSignature()
	hash := HashMsg(msgData)

	pubKey, err := RecoverPubKey(signature, hash)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func RecoverPubKeyFromRelayReply(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (secp256k1.PubKey, error) {
	dataToSign := DataToSignRequestResponse(relayResponse, relayReq)
	pubKey, err := RecoverPubKey(relayResponse.Sig, dataToSign)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func RecoverPubKeyFromReplyMetadata(relayResponse *conflicttypes.ReplyMetadata) (secp256k1.PubKey, error) {
	dataToSign := relayResponse.HashAllDataHash
	pubKey, err := RecoverPubKey(relayResponse.Sig, dataToSign)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func RecoverPubKeyFromResponseFinalizationData(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest, consumerAddr sdk.AccAddress) (secp256k1.PubKey, error) {
	dataToSign := DataToSignResponseFinalizationData(relayResponse.LatestBlock, relayResponse.FinalizedBlocksHashes, relayReq.RelaySession, consumerAddr)
	pubKey, err := RecoverPubKey(relayResponse.SigBlocks, dataToSign)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func RecoverPubKeyFromReplyMetadataFinalizationData(relayResponse *conflicttypes.ReplyMetadata, relayReq *pairingtypes.RelayRequest, addr sdk.AccAddress) (secp256k1.PubKey, error) {
	dataToSign := DataToSignResponseFinalizationData(relayResponse.LatestBlock, relayResponse.FinalizedBlocksHashes, relayReq.RelaySession, addr)
	pubKey, err := RecoverPubKey(relayResponse.SigBlocks, dataToSign)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func GenerateFloatingKey() (secretKey *btcSecp256k1.PrivateKey, addr sdk.AccAddress) {
	secretKey, _ = btcSecp256k1.NewPrivateKey(btcSecp256k1.S256())
	publicBytes := (secp256k1.PubKey)(secretKey.PubKey().SerializeCompressed())
	addr, _ = sdk.AccAddressFromHex(publicBytes.Address().String())
	return
}

func CalculateContentHashForRelayData(relayRequestData *pairingtypes.RelayPrivateData) []byte {
	requestBlockBytes := make([]byte, 8)
	metadataBytes := make([]byte, 0)
	metadata := relayRequestData.Metadata
	for _, metadataEntry := range metadata {
		metadataBytes = append(metadataBytes, []byte(metadataEntry.Name+metadataEntry.Value)...)
	}
	binary.LittleEndian.PutUint64(requestBlockBytes, uint64(relayRequestData.RequestBlock))
	addon := relayRequestData.Addon
	if addon == nil {
		addon = []string{}
	}
	msgData := bytes.Join([][]byte{metadataBytes, []byte(strings.Join(addon, "")), []byte(relayRequestData.ApiInterface), []byte(relayRequestData.ConnectionType), []byte(relayRequestData.ApiUrl), relayRequestData.Data, requestBlockBytes, relayRequestData.Salt}, nil)
	return HashMsg(msgData)
}
