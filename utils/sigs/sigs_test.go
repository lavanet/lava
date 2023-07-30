package sigs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type MockSignable struct {
	data       string
	sig        []byte
	hashRounds int
}

func (ms MockSignable) GetSignature() []byte {
	return ms.sig
}

func (ms MockSignable) DataToSign() []byte {
	return []byte(ms.data)
}

func (ms MockSignable) HashRounds() int {
	return ms.hashRounds
}

func NewMockSignable(data string, hashRounds int) MockSignable {
	return MockSignable{data: data, hashRounds: hashRounds}
}

func TestSignAndExtract(t *testing.T) {
	sk, addr := GenerateFloatingKey()
	mock := NewMockSignable("hello", 1)
	sig, err := Sign(sk, mock)
	require.Nil(t, err)
	mock.sig = sig
	extractedAddr, err := ExtractSignerAddress(mock)
	require.Nil(t, err)
	require.Equal(t, addr, extractedAddr)
}

func TestHashRounds(t *testing.T) {
	sk, _ := GenerateFloatingKey()
	data := "hello"

	mock0 := NewMockSignable(data, 0)
	mock1 := NewMockSignable(data, 1)
	mock2 := NewMockSignable(data, 2)

	// change mock0 and mock1 data (with hashes) so it'll match mock2 data
	mock0.data = string(HashMsg(HashMsg(mock0.DataToSign())))
	mock1.data = string(HashMsg(mock1.DataToSign()))

	sig0, err := Sign(sk, mock0)
	require.Nil(t, err)
	sig1, err := Sign(sk, mock1)
	require.Nil(t, err)
	sig2, err := Sign(sk, mock2)
	require.Nil(t, err)

	// check all sigs are the same
	require.Equal(t, sig0, sig1)
	require.Equal(t, sig0, sig2)
}
