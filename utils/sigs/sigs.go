package sigs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
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

func prepareRelaySessionForSignature(request *pairingtypes.RelaySession) {
	request.Badge = nil // its not a part of the signature, its a separate part
	request.Sig = []byte{}
}

func SignRelay(pkey *btcSecp256k1.PrivateKey, request pairingtypes.RelaySession) ([]byte, error) {
	prepareRelaySessionForSignature(&request)
	msgData := []byte(request.String())
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, HashMsg(msgData), false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func SignBadge(pkey *btcSecp256k1.PrivateKey, badge pairingtypes.Badge) ([]byte, error) {
	badge.ProjectSig = []byte{}
	msgData := []byte(badge.String())
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, HashMsg(msgData), false)
	if err != nil {
		return nil, err
	}

	return sig, nil
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

func AllDataHash(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (data_hash []byte) {
	nonceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceBytes, relayResponse.Nonce)
	data_hash = HashMsg(bytes.Join([][]byte{relayResponse.Data, nonceBytes, []byte(relayReq.String())}, nil))
	return
}

func DataToSignRelayResponse(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (dataToSign []byte) {
	// sign the data hash+query hash+nonce
	queryHash := utils.CalculateQueryHash(*relayReq.RelayData)
	data_hash := AllDataHash(relayResponse, relayReq)
	dataToSign = bytes.Join([][]byte{data_hash, queryHash}, nil)
	dataToSign = HashMsg(dataToSign)
	return
}

func DataToVerifyProviderSig(request *pairingtypes.RelayRequest, data_hash []byte) (dataToSign []byte) {
	queryHash := utils.CalculateQueryHash(*request.RelayData)
	dataToSign = bytes.Join([][]byte{data_hash, queryHash}, nil)
	dataToSign = HashMsg(dataToSign)
	return
}

func DataToSignResponseFinalizationData(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest, clientAddress sdk.AccAddress) (dataToSign []byte) {
	// sign latest_block+finalized_blocks_hashes+session_id+block_height+relay_num
	return DataToSignResponseFinalizationDataInner(relayResponse.LatestBlock, relayReq.RelaySession.SessionId, relayReq.RelaySession.Epoch, relayReq.RelaySession.RelayNum, relayResponse.FinalizedBlocksHashes, clientAddress)
}

func DataToSignResponseFinalizationDataInner(latestBlock int64, sessionID uint64, blockHeight int64, relayNum uint64, finalizedBlockHashes []byte, clientAddress sdk.AccAddress) (dataToSign []byte) {
	latestBlockBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(latestBlockBytes, uint64(latestBlock))
	sessionIdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionIdBytes, sessionID)
	blockHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(blockHeightBytes, uint64(blockHeight))
	relayNumBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(relayNumBytes, relayNum)
	return bytes.Join([][]byte{latestBlockBytes, finalizedBlockHashes, sessionIdBytes, blockHeightBytes, relayNumBytes, clientAddress}, nil)
}

func SignRelayResponse(pkey *btcSecp256k1.PrivateKey, relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) ([]byte, error) {
	relayResponse.Sig = []byte{}
	dataToSign := DataToSignRelayResponse(relayResponse, relayReq)
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, dataToSign, false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func SignResponseFinalizationData(pkey *btcSecp256k1.PrivateKey, relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest, clientAddress sdk.AccAddress) ([]byte, error) {
	dataToSign := DataToSignResponseFinalizationData(relayResponse, relayReq, clientAddress)
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

func RecoverProviderPubKeyFromQueryAndAllDataHash(request *pairingtypes.RelayRequest, allDataHash []byte, providerSig []byte) (secp256k1.PubKey, error) {
	dataToSign := DataToVerifyProviderSig(request, allDataHash)
	pubKey, err := RecoverPubKey(providerSig, dataToSign)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func RecoverPubKeyFromRelay(relay pairingtypes.RelaySession) (secp256k1.PubKey, error) {
	signature := relay.Sig // save sig
	prepareRelaySessionForSignature(&relay)
	hash := HashMsg([]byte(relay.String()))

	pubKey, err := RecoverPubKey(signature, hash)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
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

func RecoverPubKeyFromRelayReply(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (secp256k1.PubKey, error) {
	dataToSign := DataToSignRelayResponse(relayResponse, relayReq)
	pubKey, err := RecoverPubKey(relayResponse.Sig, dataToSign)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func RecoverPubKeyFromResponseFinalizationData(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest, addr sdk.AccAddress) (secp256k1.PubKey, error) {
	dataToSign := DataToSignResponseFinalizationData(relayResponse, relayReq, addr)
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
	binary.LittleEndian.PutUint64(requestBlockBytes, uint64(relayRequestData.RequestBlock))
	msgData := bytes.Join([][]byte{[]byte(relayRequestData.ApiInterface), []byte(relayRequestData.ConnectionType), []byte(relayRequestData.ApiUrl), relayRequestData.Data, requestBlockBytes, relayRequestData.Salt}, nil)
	return HashMsg(msgData)
}
