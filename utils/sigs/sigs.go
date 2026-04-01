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

	tendermintcrypto "github.com/cometbft/cometbft/crypto"
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // needed for Bitcoin-style address derivation
)

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

// pubKeyBytesToAddress computes RIPEMD160(SHA256(pubkeyBytes)), matching
// the algorithm used by cosmos-sdk's secp256k1.PubKey.Address().
func pubKeyBytesToAddress(pubkeyBytes []byte) []byte {
	shaHash := sha256.Sum256(pubkeyBytes)
	hasher := ripemd160.New()
	hasher.Write(shaHash[:])
	return hasher.Sum(nil)
}
