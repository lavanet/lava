package sigs

import (
	"encoding/binary"
	"errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

func GetKeyName(clientCtx client.Context) (string, error) {
	_, name, _, err := client.GetFromFields(clientCtx.Keyring, clientCtx.From, false)
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

func SignRelay(pkey *btcSecp256k1.PrivateKey, msgData []byte) ([]byte, error) {
	//
	// Sign
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, HashMsg(msgData), false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func SignRelayResponse(pkey *btcSecp256k1.PrivateKey, relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) ([]byte, error) {
	//
	queryHash := HashMsg([]byte(relayReq.String()))
	data_hash := HashMsg(relayResponse.Data)
	nonceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceBytes, relayResponse.Nonce)
	dataToSign := append(data_hash, queryHash...)
	dataToSign = append(dataToSign, nonceBytes...)
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
		return nil, err
	}

	pk := recPub.SerializeCompressed()

	return (secp256k1.PubKey)(pk), nil
}

func RecoverPubKeyFromRelay(in *pairingtypes.RelayRequest) (secp256k1.PubKey, error) {
	tmp := in.Sig
	in.Sig = []byte{}
	hash := HashMsg([]byte(in.String()))
	in.Sig = tmp

	pubKey, err := RecoverPubKey(in.Sig, hash)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func RecoverPubKeyFromRelayReply(relayResponse *pairingtypes.RelayReply, relayReq *pairingtypes.RelayRequest) (secp256k1.PubKey, error) {
	// relay reply signature signs: sign the data hash+query hash+nonce
	queryHash := HashMsg([]byte(relayReq.String()))
	data_hash := HashMsg(relayResponse.Data)
	nonceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceBytes, relayResponse.Nonce)
	dataToSign := append(data_hash, queryHash...)
	dataToSign = append(dataToSign, nonceBytes...)
	pubKey, err := RecoverPubKey(relayResponse.Sig, dataToSign)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}
