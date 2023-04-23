package chainproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"time"

	"github.com/lavanet/lava/protocol/parser"
)

const (
	LavaErrorCode       = 555
	InternalErrorString = "Internal Error"
)

type CustomParsingMessage interface {
	NewParsableRPCInput(input json.RawMessage) (parser.RPCInput, error)
}

type DefaultRPCInput struct {
	Result json.RawMessage
}

func (dri DefaultRPCInput) GetParams() interface{} {
	return nil
}

func (dri DefaultRPCInput) GetResult() json.RawMessage {
	return dri.Result
}

func (dri DefaultRPCInput) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func DefaultParsableRPCInput(input json.RawMessage) parser.RPCInput {
	return DefaultRPCInput{Result: input}
}

func generateSelfSignedCertificate() (tls.Certificate, error) {
	// Generate private key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Generate certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Return certificate and private key as a tls.Certificate
	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  key,
	}, nil
}
