package types

import (
	"bytes"
	"encoding/binary"
)

func (rs RelaySession) GetSignature() []byte {
	return rs.Sig
}

func (rs RelaySession) DataToSign() []byte {
	rs.Badge = nil // its not a part of the signature, its a separate part
	rs.Sig = []byte{}
	return []byte(rs.String())
}

func (rs RelaySession) HashRounds() int {
	return 1
}

func (rs RelaySession) CalculateHashForFinalization() []byte {
	sessionIdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionIdBytes, rs.SessionId)
	blockHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(blockHeightBytes, uint64(rs.Epoch))
	relayNumBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(relayNumBytes, rs.RelayNum)
	return bytes.Join([][]byte{sessionIdBytes, blockHeightBytes, relayNumBytes}, nil)
}
