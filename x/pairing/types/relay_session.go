package types

import (
	"github.com/lavanet/lava/utils/sigs"
)

func (rs RelaySession) GetSignature() []byte {
	return rs.Sig
}

func (rs RelaySession) DataToSign() []byte {
	rs.Badge = nil // its not a part of the signature, its a separate part
	rs.Sig = nil
	return []byte(rs.String())
}

func (rs RelaySession) HashRounds() int {
	return 1
}

func (rs RelaySession) CalculateHashForFinalization() []byte {
	sessionIdBytes := sigs.Encode(rs.SessionId)
	blockHeightBytes := sigs.Encode(uint64(rs.Epoch))
	relayNumBytes := sigs.Encode(rs.RelayNum)
	msgParts := [][]byte{
		sessionIdBytes,
		blockHeightBytes,
		relayNumBytes,
	}
	return sigs.Join(msgParts)
}
