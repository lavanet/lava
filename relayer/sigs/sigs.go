package sigs

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto"
	servicertypes "github.com/lavanet/lava/x/servicer/types"

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

func RecoverPubKeyFromRelay(in *servicertypes.RelayRequest) (secp256k1.PubKey, error) {
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

func RecoverPubKeyFromRelayReply(in *servicertypes.RelayReply) (secp256k1.PubKey, error) {
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
