package sigs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type MockSignable struct {
	data string
	sig  []byte
}

func (ms MockSignable) GetSignature() []byte {
	return ms.sig
}

func (ms MockSignable) DataToSign() []byte {
	return []byte(ms.data)
}

func (ms MockSignable) HashRounds() int {
	return 1
}

func TestSignAndExtract(t *testing.T) {
	sk, addr := GenerateFloatingKey()
	mock := MockSignable{data: "hello"}
	sig, err := Sign(sk, mock)
	require.Nil(t, err)
	mock.sig = sig
	extractedAddr, err := ExtractSignerAddress(mock)
	require.Nil(t, err)
	require.Equal(t, addr, extractedAddr)
}
