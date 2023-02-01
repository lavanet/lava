package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/99designs/keyring"
	vrf "github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

const (
	bechPrefix    = "vrf"
	pk_vrf_prefix = "vrf-pk-"
	sk_vrf_prefix = "vrf-sk-"
)

var VRFValueAboveReliabilityThresholdError = sdkerrors.New("VRFValueAboveReliabilityThreshold Error", 1, "calculated vrf does not result in a smaller value than threshold") // client could'nt connect to any provider.

func GetIndexForVrf(vrf []byte, providersCount uint32, reliabilityThreshold uint32) (index int64, err error) {
	vrf_num := binary.LittleEndian.Uint32(vrf)
	if vrf_num <= reliabilityThreshold {
		// need to send relay with VRF
		modulo := providersCount
		index = int64(vrf_num % modulo)
	} else {
		index = -1
		err = VRFValueAboveReliabilityThresholdError.Wrapf("Vrf Does not meet threshold: %d VS threshold: %d", vrf_num, reliabilityThreshold)
	}
	return
}

func verifyVRF(queryHash []byte, reliabilityData *pairingtypes.VRFData, vrf_pk VrfPubKey, relayEpochStart uint64) (valid bool) {
	providerSig := reliabilityData.ProviderSig
	differentiator := []uint8{0}
	if reliabilityData.Differentiator {
		differentiator = []uint8{1}
	}
	relayEpochStartBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(relayEpochStartBytes, relayEpochStart)
	vrf_data := bytes.Join([][]byte{queryHash, relayEpochStartBytes, providerSig, differentiator}, nil)
	return vrf_pk.pk.Verify(vrf_data, reliabilityData.VrfValue, reliabilityData.VrfProof)
}

func VerifyVrfProofFromVRFData(reliabilityData *pairingtypes.VRFData, vrf_pk VrfPubKey, relayEpochStart uint64) (valid bool) {
	queryHash := reliabilityData.QueryHash
	return verifyVRF(queryHash, reliabilityData, vrf_pk, relayEpochStart)
}

func VerifyVrfProof(request *pairingtypes.RelayRequest, vrf_pk VrfPubKey, relayEpochStart uint64) (valid bool) {
	queryHash := CalculateQueryHash(*request)
	return verifyVRF(queryHash, request.DataReliability, vrf_pk, relayEpochStart)
}

func CalculateVrfOnRelay(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply, vrf_sk vrf.PrivateKey, currentEpoch uint64) ([]byte, []byte) {
	vrfData0 := FormatDataForVrf(request, response, false, currentEpoch)
	vrfData1 := FormatDataForVrf(request, response, true, currentEpoch)
	return vrf_sk.Compute(vrfData0), vrf_sk.Compute(vrfData1)
}

func ProveVrfOnRelay(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply, vrf_sk vrf.PrivateKey, differentiator bool, currentEpoch uint64) (vrf_res []byte, proof []byte) {
	vrfData := FormatDataForVrf(request, response, differentiator, currentEpoch)
	return vrf_sk.Prove(vrfData)
}

func CalculateQueryHash(relayReq pairingtypes.RelayRequest) (queryHash []byte) {
	relayReq.CuSum = 0
	relayReq.Provider = ""
	relayReq.RelayNum = 0
	relayReq.SessionId = 0
	relayReq.Sig = nil
	relayReq.QoSReport = nil
	relayReq.DataReliability = nil
	relayReq.UnresponsiveProviders = nil
	queryHash = tendermintcrypto.Sha256([]byte(relayReq.String()))
	return
}

func FormatDataForVrf(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply, differentiator bool, currentEpoch uint64) (data []byte) {
	// vrf is calculated on: query hash, relayer signature and 0/1 byte
	queryHash := CalculateQueryHash(*request)
	currentEpochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(currentEpochBytes, currentEpoch)
	if differentiator {
		data = bytes.Join([][]byte{queryHash, currentEpochBytes, response.Sig, []uint8{1}}, nil)
	} else {
		data = bytes.Join([][]byte{queryHash, currentEpochBytes, response.Sig, []uint8{0}}, nil)
	}
	return
}

func VerifyVRF(vrfpk string) error {
	// everything is okay
	if vrfpk == "" {
		return fmt.Errorf("can't stake with an empty vrf pk bech32 string")
	}
	return nil
}

func GeneratePrivateVRFKey() (vrf.PrivateKey, vrf.PublicKey, error) {
	privateKey, err := vrf.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}
	pk, success := privateKey.Public()
	if !success {
		return nil, nil, err
	}
	return privateKey, pk, nil
}

func GetOrCreateVRFKey(clientCtx client.Context) (sk vrf.PrivateKey, pk *VrfPubKey, err error) {
	sk, pk, err = LoadVRFKey(clientCtx)
	if err != nil {
		sk, pk, err = GenerateVRFKey(clientCtx)
		fmt.Printf("Generated New VRF Key: {%X}\n", pk)
	}
	return
}

func GenerateVRFKey(clientCtx client.Context) (vrf.PrivateKey, *VrfPubKey, error) {
	kr, err := OpenKeyring(clientCtx)
	if err != nil {
		return nil, nil, err
	}
	sk, pk, err := GeneratePrivateVRFKey()
	if err != nil {
		return nil, nil, err
	}
	key := keyring.Item{Key: pk_vrf_prefix + clientCtx.FromName, Data: pk, Label: "pk", Description: "the vrf public key"}
	err = kr.Set(key)
	if err != nil {
		return nil, nil, err
	}
	key = keyring.Item{Key: sk_vrf_prefix + clientCtx.FromName, Data: sk, Label: "sk", Description: "the vrf secret key"}
	err = kr.Set(key)
	if err != nil {
		return nil, nil, err
	}
	return sk, &VrfPubKey{pk: pk}, nil
}

func OpenKeyring(clientCtx client.Context) (keyring.Keyring, error) {
	keyringConfig := keyring.Config{
		AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		ServiceName:     "vrf",
		KeychainName:    "vrf",
		FileDir:         clientCtx.KeyringDir,
		FilePasswordFunc: func(_ string) (string, error) {
			return "test", nil
		},
	}
	kr, err := keyring.Open(keyringConfig)
	if err != nil {
		return nil, err
	}
	return kr, nil
}

func LoadVRFKey(clientCtx client.Context) (vrf.PrivateKey, *VrfPubKey, error) {
	kr, err := OpenKeyring(clientCtx)
	if err != nil {
		return nil, nil, err
	}
	pkItem, err := kr.Get(pk_vrf_prefix + clientCtx.FromName)
	if err != nil {
		return nil, nil, err
	}
	skItem, err := kr.Get(sk_vrf_prefix + clientCtx.FromName)
	return skItem.Data, &VrfPubKey{pk: pkItem.Data}, err
}

// type PubKey interface {
// 	proto.Message

// 	Address() Address
// 	Bytes() []byte
// 	VerifySignature(msg []byte, sig []byte) bool
// 	Equals(PubKey) bool
// 	Type() string
// }

type VrfPubKey struct {
	pk vrf.PublicKey
}

func (pk *VrfPubKey) Bytes() []byte {
	if pk == nil {
		return nil
	}
	return pk.pk
}

func (pk *VrfPubKey) DecodeFromBech32(bech32str string) (*VrfPubKey, error) {
	hrp, bz, err := bech32.DecodeAndConvert(bech32str)
	if hrp != bechPrefix {
		return nil, fmt.Errorf("invalid prefix for bech string: %s", hrp)
	}
	pk.pk = bz
	return pk, err
}

func (pk *VrfPubKey) EncodeBech32() (string, error) {
	return bech32.ConvertAndEncode(bechPrefix, pk.Bytes())
}

func (pk *VrfPubKey) Equals(pk2 VrfPubKey) bool {
	return bytes.Equal(pk.Bytes(), pk2.Bytes())
}

func (pk *VrfPubKey) VerifySignature(m []byte, vrfBytes []byte, proof []byte) bool {
	return pk.pk.Verify(m, vrfBytes, proof)
}

// String returns a string representation of the public key
func (pk *VrfPubKey) String() string {
	st, err := pk.EncodeBech32()
	if err != nil {
		return fmt.Sprintf("{%X}", pk.Bytes())
	}
	return st
}

func (pk *VrfPubKey) Reset() { *pk = VrfPubKey{} }

// // **** Proto Marshaler ****

// // MarshalTo implements proto.Marshaler interface.
func (pk *VrfPubKey) MarshalTo(dAtA []byte) (int, error) {
	bz := pk.Bytes()
	copy(dAtA, bz)
	return len(bz), nil
}

// // Unmarshal implements proto.Marshaler interface.
func (pk *VrfPubKey) Unmarshal(bz []byte) error {
	pk.pk = bz
	return nil
}
