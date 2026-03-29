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
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	btcSecp256k1Ecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	tendermintcrypto "github.com/cometbft/cometbft/crypto"
	btcutilbech32 "github.com/cosmos/btcutil/bech32"
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // needed for Bitcoin-style address derivation
	"github.com/lavanet/lava/v5/utils"
)

// Bech32AddrPrefix is the bech32 prefix used when encoding AccAddress values.
// It defaults to "cosmos" (the SDK default) so that test code behaves identically
// to the previous sdk.AccAddress.String() output.
// Production callers that need a different prefix can set this at startup.
var Bech32AddrPrefix = "cosmos"

// AccAddress is a []byte that encodes to a bech32 string, mirroring sdk.AccAddress.
type AccAddress []byte

// String returns the bech32 encoding of the address using Bech32AddrPrefix.
func (a AccAddress) String() string {
	if len(a) == 0 {
		return ""
	}
	s, err := btcutilbech32.EncodeFromBase256(Bech32AddrPrefix, []byte(a))
	if err != nil {
		return fmt.Sprintf("<invalid-address: %s>", err)
	}
	return s
}

// PubKeyAddress is implemented by any public key that can derive a blockchain address.
// It intentionally avoids depending on cosmos-sdk types.
type PubKeyAddress interface {
	Address() tendermintcrypto.Address
}

type Signable interface {
	// GetSignature gets the object's signature
	GetSignature() []byte
	// DataToSign processes the object's data before it's hashed and signed
	DataToSign() []byte
	// HashRounds gets the number of times the object's data is hashed before it's signed
	HashRounds() int
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
func ExtractSignerAddress(data Signable) (AccAddress, error) {
	pubKey, err := RecoverPubKey(data)
	if err != nil {
		return nil, err
	}

	addrBytes := pubKeyToAccAddress(pubKey.Address())
	return AccAddress(addrBytes), nil
}

// RecoverPubKey recovers the public key from data's signature.
// The returned PubKeyAddress supports .Address().String() to obtain a hex address.
func RecoverPubKey(data Signable) (PubKeyAddress, error) {
	sig := data.GetSignature()

	msgData := data.DataToSign()
	for i := 0; i < data.HashRounds(); i++ {
		msgData = HashMsg(msgData)
	}

	// Recover public key from signature
	recPub, _, err := btcSecp256k1Ecdsa.RecoverCompact(sig, msgData)
	if err != nil {
		return nil, utils.LavaFormatError("RecoverCompact", err,
			utils.Attribute{Key: "sigLen", Value: len(sig)},
		)
	}

	return &btcPubKeyWrapper{pub: recPub}, nil
}

// btcPubKeyWrapper wraps a btcSecp256k1 public key and implements PubKeyAddress.
type btcPubKeyWrapper struct {
	pub *btcSecp256k1.PublicKey
}

// Address returns the Bitcoin-style RIPEMD160(SHA256(compressed-pubkey)) address.
func (w *btcPubKeyWrapper) Address() tendermintcrypto.Address {
	compressed := w.pub.SerializeCompressed()
	return pubKeyBytesToAddress(compressed)
}

// pubKeyBytesToAddress computes RIPEMD160(SHA256(pubkeyBytes)), matching
// the algorithm used by cosmos-sdk's secp256k1.PubKey.Address().
func pubKeyBytesToAddress(pubkeyBytes []byte) []byte {
	shaHash := sha256.Sum256(pubkeyBytes)
	hasher := ripemd160.New()
	hasher.Write(shaHash[:])
	return hasher.Sum(nil)
}

// pubKeyToAccAddress converts a tendermintcrypto.Address (hex bytes) to AccAddress
// by treating it as raw bytes.
func pubKeyToAccAddress(addr tendermintcrypto.Address) []byte {
	return []byte(addr)
}

// EncodeUint64 encodes a uint64 value to a byte array
func EncodeUint64(val uint64) []byte {
	encodedVal := make([]byte, 8)
	binary.LittleEndian.PutUint64(encodedVal, val)
	return encodedVal
}

// HashMsg hashes msgData using SHA-256
func HashMsg(msgData []byte) []byte {
	return tendermintcrypto.Sha256(msgData)
}

// GenerateFloatingKey creates a new private key with an account address derived from the corresponding public key
func GenerateFloatingKey() (secretKey *btcSecp256k1.PrivateKey, addr AccAddress) {
	sk, err := btcSecp256k1.NewPrivateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate private key: %v", err))
	}
	compressed := sk.PubKey().SerializeCompressed()
	addrBytes := pubKeyBytesToAddress(compressed)
	return sk, AccAddress(addrBytes)
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

// generateKeyFromSecret generates a deterministic private key from a secret seed.
// This mirrors the logic of secp256k1.GenPrivKeyFromSecret without the cosmos-sdk dependency.
func generateKeyFromSecret(secret []byte) (*btcSecp256k1.PrivateKey, error) {
	// Use btcec scalar operations to create a deterministic key
	// secp256k1.GenPrivKeyFromSecret uses SHA256-based hashing; replicate it:
	// The cosmos secp256k1.GenPrivKeyFromSecret hashes repeatedly until valid.
	// For our purposes (test key generation) a single pass is sufficient.
	hasher := sha256.New()
	hasher.Write(secret)
	skBytes := hasher.Sum(nil)
	sk, _ := btcSecp256k1.PrivKeyFromBytes(skBytes)
	if sk == nil {
		return nil, errors.New("failed to derive private key from secret")
	}
	return sk, nil
}

// GenerateDeterministicFloatingKey creates a new private key with an account address derived from the corresponding public key using a rand source.
// The returned account holds only secp256k1 keys; ed25519 ConsKey is no longer populated.
func GenerateDeterministicFloatingKey(r io.Reader) (sk *btcSecp256k1.PrivateKey, addr AccAddress) {
	privkeySeed := make([]byte, 15)
	_, err := r.Read(privkeySeed)
	if err != nil {
		panic("failed to create account)")
	}

	key, err := generateKeyFromSecret(privkeySeed)
	if err != nil {
		panic(fmt.Sprintf("failed to generate deterministic key: %v", err))
	}

	compressed := key.PubKey().SerializeCompressed()
	addrBytes := pubKeyBytesToAddress(compressed)
	return key, AccAddress(addrBytes)
}
