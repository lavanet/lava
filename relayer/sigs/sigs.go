package sigs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

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

func SignVRFData(pkey *btcSecp256k1.PrivateKey, vrfData *pairingtypes.VRFData) ([]byte, error) {
	msgData := []byte(vrfData.String())
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, HashMsg(msgData), false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func SignRelay(pkey *btcSecp256k1.PrivateKey, request pairingtypes.RelayRequest) ([]byte, error) {
	//
	request.DataReliability = nil // its not a part of the signature, its a separate part
	request.Sig = []byte{}
	msgData := []byte(request.String())
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, HashMsg(msgData), false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func AllDataHash(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (data_hash []byte) {
	nonceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceBytes, relayResponse.Nonce)
	data_hash = HashMsg(bytes.Join([][]byte{relayResponse.Data, nonceBytes, []byte(relayReq.String())}, nil))
	return
}

func DataToSignRelayResponse(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (dataToSign []byte) {
	// sign the data hash+query hash+nonce
	queryHash := utils.CalculateQueryHash(*relayReq)
	data_hash := AllDataHash(relayResponse, relayReq)
	dataToSign = bytes.Join([][]byte{data_hash, queryHash}, nil)
	dataToSign = HashMsg(dataToSign)
	return
}

func DataToVerifyProviderSig(request *pairingtypes.RelayRequest, data_hash []byte) (dataToSign []byte) {
	queryHash := utils.CalculateQueryHash(*request)
	dataToSign = bytes.Join([][]byte{data_hash, queryHash}, nil)
	dataToSign = HashMsg(dataToSign)
	return
}

func DataToSignResponseFinalizationData(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest, clientAddress sdk.AccAddress) (dataToSign []byte) {
	// sign latest_block+finalized_blocks_hashes+session_id+block_height+relay_num
	return DataToSignResponseFinalizationDataInner(relayResponse.LatestBlock, relayReq.SessionId, relayReq.BlockHeight, relayReq.RelayNum, relayResponse.FinalizedBlocksHashes, clientAddress)
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
		return nil, utils.LavaFormatError("RecoverCompact", err, &map[string]string{
			"sigLen": strconv.FormatInt(int64(len(sig)), 10),
		})
	}
	pk := recPub.SerializeCompressed()

	return (secp256k1.PubKey)(pk), nil
}

func RecoverPubKeyFromVRFData(vrfData pairingtypes.VRFData) (secp256k1.PubKey, error) {
	signature := vrfData.Sig
	vrfData.Sig = nil
	msgDataHash := HashMsg([]byte(vrfData.String()))
	pubKey, err := RecoverPubKey(signature, msgDataHash)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func ValidateSignerOnVRFData(signer sdk.AccAddress, dataReliability pairingtypes.VRFData) (valid bool, err error) {
	pubKey, err := RecoverPubKeyFromVRFData(dataReliability)
	if err != nil {
		return false, fmt.Errorf("RecoverPubKeyFromVRFData: %w", err)
	}
	signerAccAddress, err := sdk.AccAddressFromHex(pubKey.Address().String()) // signer
	if err != nil {
		return false, fmt.Errorf("AccAddressFromHex : %w", err)
	}
	if !signerAccAddress.Equals(signer) {
		return false, fmt.Errorf("signer on VRFData is not the same as on the original relay request %s, %s", signerAccAddress.String(), signer.String())
	}
	return true, nil
}

func RecoverProviderPubKeyFromVrfDataOnly(dataReliability *pairingtypes.VRFData) (providerAccAddress sdk.AccAddress, err error) {
	queryHash := dataReliability.QueryHash
	data_hash := dataReliability.AllDataHash
	dataToSign := bytes.Join([][]byte{data_hash, queryHash}, nil)
	dataToSign = HashMsg(dataToSign)
	pubKey, err := RecoverPubKey(dataReliability.ProviderSig, dataToSign)
	if err != nil {
		return nil, fmt.Errorf("err: %w DataReliability: %+v", err, dataReliability)
	}
	providerAccAddress, err = sdk.AccAddressFromHex(pubKey.Address().String()) // consumer signer
	return
}

func RecoverProviderPubKeyFromQueryAndAllDataHash(request *pairingtypes.RelayRequest, allDataHash []byte, providerSig []byte) (secp256k1.PubKey, error) {
	dataToSign := DataToVerifyProviderSig(request, allDataHash)
	pubKey, err := RecoverPubKey(providerSig, dataToSign)
	if err != nil {
		return nil, fmt.Errorf("err: %w DataReliability: %+v", err, request.DataReliability)
	}
	return pubKey, nil
}

func RecoverProviderPubKeyFromVrfDataAndQuery(request *pairingtypes.RelayRequest) (secp256k1.PubKey, error) {
	// we take the all data hash from reliability
	return RecoverProviderPubKeyFromQueryAndAllDataHash(request, request.DataReliability.AllDataHash, request.DataReliability.ProviderSig)
}

func RecoverPubKeyFromRelay(in pairingtypes.RelayRequest) (secp256k1.PubKey, error) {
	signature := in.Sig
	in.Sig = []byte{}
	in.DataReliability = nil
	hash := HashMsg([]byte(in.String()))
	pubKey, err := RecoverPubKey(signature, hash)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
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
