// Signable is an interface for objects that should be signed. For example, relay requests
// are signed by the provider so it can prove that it's their relay.
//
// To create an object that satisfies the Signable interface, use relay_exchange.go as a reference
//
// A Signable object can use the Sign() to be signed, ExtractSignerAddress() to get the object
// that signed it, and RecoverPubKey() to get the public key that corresponds to the object's
// private key

package sigs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	btcSecp256k1Ecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	tendermintcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
)

type Account struct {
	sk      cryptotypes.PrivKey
	SK      *btcSecp256k1.PrivateKey
	PubKey  cryptotypes.PubKey
	Addr    sdk.AccAddress
	ConsKey cryptotypes.PrivKey
	Vault   *Account // provider vault account (only for provider)
}

type Signable interface {
	// GetSignature gets the object's signature
	GetSignature() []byte
	// DataToSign processes the object's data before it's hashed and signed
	DataToSign() []byte
	// HashRounds gets the number of times the object's data is hashed before it's signed
	HashRounds() int
}

func (acc Account) GetVaultAddr() string {
	if acc.Vault != nil {
		return acc.Vault.Addr.String()
	}

	return ""
}

// Sign creates a signature for a struct. The prepareFunc prepares the struct before extracting the data for the signature
func Sign(pkey *btcSecp256k1.PrivateKey, data Signable) ([]byte, error) {
	msgData := data.DataToSign()
	for i := 0; i < data.HashRounds(); i++ {
		msgData = HashMsg(msgData)
	}

	sig, err := btcSecp256k1Ecdsa.SignCompact(pkey, msgData, false)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

// ExtractSignerAddress extracts the signer address of data
func ExtractSignerAddress(data Signable) (sdk.AccAddress, error) {
	pubKey, err := RecoverPubKey(data)
	if err != nil {
		return nil, err
	}

	extractedConsumerAddress, err := sdk.AccAddressFromHexUnsafe(pubKey.Address().String())
	if err != nil {
		return nil, fmt.Errorf("get relay consumer address: %s", err.Error())
	}

	return extractedConsumerAddress, nil
}

// RecoverPubKey recovers the public key from data's signature
func RecoverPubKey(data Signable) (secp256k1.PubKey, error) {
	sig := data.GetSignature()

	msgData := data.DataToSign()
	for i := 0; i < data.HashRounds(); i++ {
		msgData = HashMsg(msgData)
	}

	// Recover public key from signature
	recPub, _, err := btcSecp256k1Ecdsa.RecoverCompact(sig, msgData)
	if err != nil {
		return secp256k1.PubKey{}, utils.LavaFormatError("RecoverCompact", err,
			utils.Attribute{Key: "sigLen", Value: len(sig)},
		)
	}
	pk := recPub.SerializeCompressed()

	return secp256k1.PubKey{Key: pk}, nil
}

// EncodeUint64 encodes a uint64 value to a byte array
func EncodeUint64(val uint64) []byte {
	encodedVal := make([]byte, 8)
	binary.LittleEndian.PutUint64(encodedVal, val)
	return encodedVal
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

	priv, _ := btcSecp256k1.PrivKeyFromBytes(privKey.Bytes())
	return priv, nil
}

// HashMsg hashes msgData using SHA-256
func HashMsg(msgData []byte) []byte {
	return tendermintcrypto.Sha256(msgData)
}

// GenerateFloatingKey creates a new private key with an account address derived from the corresponding public key
func GenerateFloatingKey() (secretKey *btcSecp256k1.PrivateKey, addr sdk.AccAddress) {
	sk := secp256k1.GenPrivKey()
	PubKey := sk.PubKey()
	addr = sdk.AccAddress(PubKey.Address())
	secretKey, _ = btcSecp256k1.PrivKeyFromBytes(sk.Bytes())
	return
}

type ZeroReader struct {
	Seed byte
	rand *rand.Rand
}

func NewZeroReader(seed int64) *ZeroReader {
	return &ZeroReader{
		Seed: 1,
		rand: rand.New(rand.NewSource(seed)),
	}
}

func (z ZeroReader) Read(p []byte) (n int, err error) {
	// fool the non determinism mechanism of crypto
	if len(p) == 1 {
		p[0] = z.Seed
		return len(p), nil
	}
	return z.rand.Read(p)
}

func (z *ZeroReader) Inc() {
	z.Seed++
	if z.Seed == 0 {
		z.Seed++
	}
}

// GenerateDeterministicFloatingKey creates a new private key with an account address derived from the corresponding public key using a rand source
func GenerateDeterministicFloatingKey(rand io.Reader) (acc Account) {
	privkeySeed := make([]byte, 15)
	_, err := rand.Read(privkeySeed)
	if err != nil {
		panic("failed to create account)")
	}

	acc.sk = secp256k1.GenPrivKeyFromSecret(privkeySeed)
	acc.PubKey = acc.sk.PubKey()
	acc.Addr = sdk.AccAddress(acc.PubKey.Address())
	acc.ConsKey = ed25519.GenPrivKeyFromSecret(privkeySeed)
	acc.SK, _ = btcSecp256k1.PrivKeyFromBytes(acc.sk.Bytes())

	return
}
