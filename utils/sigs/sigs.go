package sigs

import (
	"errors"
	"fmt"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

// Signable is an interface for objects that are signed
type Signable interface {
	// GetSignature gets the object's signature
	GetSignature() []byte
	// DataToSign process the object's data before it's hashed and signed
	PrepareForSignature() []byte
	// HashRounds gets the number of times the object's data is hashed before it's signed
	HashCount() int
}

// Sign creates a signature for a struct. The prepareFunc prepares the struct before extracting the data for the signature
func Sign(pkey *btcSecp256k1.PrivateKey, data Signable) ([]byte, error) {
	msgData := data.PrepareForSignature()
	for i := 0; i < data.HashCount(); i++ {
		msgData = HashMsg(msgData)
	}

	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), pkey, msgData, false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func ExtractSignerAddress(data Signable) (sdk.AccAddress, error) {
	pubKey, err := RecoverPubKey(data)
	if err != nil {
		return nil, err
	}

	extractedConsumerAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
	if err != nil {
		return nil, fmt.Errorf("get relay consumer address: %s", err.Error())
	}

	return extractedConsumerAddress, nil
}

func RecoverPubKey(data Signable) (secp256k1.PubKey, error) {
	sig := data.GetSignature()

	msgData := data.PrepareForSignature()
	for i := 0; i < data.HashCount(); i++ {
		msgData = HashMsg(msgData)
	}

	// Recover public key from signature
	recPub, _, err := btcSecp256k1.RecoverCompact(btcSecp256k1.S256(), sig, msgData)
	if err != nil {
		return nil, utils.LavaFormatError("RecoverCompact", err,
			utils.Attribute{Key: "sigLen", Value: len(sig)},
		)
	}
	pk := recPub.SerializeCompressed()

	return (secp256k1.PubKey)(pk), nil
}

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

// GenerateFloatingKey creates a new private key with an account address derived from the corresponding public key
func GenerateFloatingKey() (secretKey *btcSecp256k1.PrivateKey, addr sdk.AccAddress) {
	secretKey, _ = btcSecp256k1.NewPrivateKey(btcSecp256k1.S256())
	publicBytes := (secp256k1.PubKey)(secretKey.PubKey().SerializeCompressed())
	addr, _ = sdk.AccAddressFromHex(publicBytes.Address().String())
	return
}
