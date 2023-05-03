package utils

import (
	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

func CreateBadge(cuAllocation uint64, epoch uint64, address string, chainID string, sig []byte) *pairingtypes.Badge {
	badge := pairingtypes.Badge{
		CuAllocation: cuAllocation,
		Epoch:        epoch,
		Address:      address,
		SpecId:       chainID,
		ProjectSig:   sig,
	}

	return &badge
}

// check badge's basic attributes compared to the same traits from the relay request
func IsBadgeValid(badge pairingtypes.Badge, clientAddr string, chainID string, epoch uint64) bool {
	if badge.GetAddress() != clientAddr || badge.GetSpecId() != chainID || badge.GetEpoch() != epoch {
		return false
	}
	return true
}

func ExtractSignerAddressFromBadge(badge pairingtypes.Badge) (sdk.AccAddress, error) {
	hash := HashMsg([]byte(badge.String()))
	pubKey, err := RecoverPubKey(badge.ProjectSig, hash)
	if err != nil {
		return nil, err
	}

	extractedConsumerAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
	if err != nil {
		return nil, LavaFormatError("get relay consumer address", err)
	}

	return extractedConsumerAddress, nil
}

func HashMsg(msgData []byte) []byte {
	return tendermintcrypto.Sha256(msgData)
}

func RecoverPubKey(sig []byte, msgHash []byte) (secp256k1.PubKey, error) {
	// Recover public key from signature
	recPub, _, err := btcSecp256k1.RecoverCompact(btcSecp256k1.S256(), sig, msgHash)
	if err != nil {
		return nil, LavaFormatError("RecoverCompact", err, Attribute{
			Key: "sigLen", Value: len(sig),
		})
	}
	pk := recPub.SerializeCompressed()

	return (secp256k1.PubKey)(pk), nil
}
