package lavasession

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/lavanet/lava/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

const (
	MaxConsecutiveConnectionAttempts                 = 5
	TimeoutForEstablishingAConnection                = 1 * time.Second
	MaxSessionsAllowedPerProvider                    = 1000 // Max number of sessions allowed per provider
	MaxAllowedBlockListedSessionPerProvider          = 3
	MaximumNumberOfFailuresAllowedPerConsumerSession = 3
	RelayNumberIncrement                             = 1
	DataReliabilitySessionId                         = 0 // data reliability session id is 0. we can change to more sessions later if needed.
	DataReliabilityRelayNumber                       = 1
	DataReliabilityCuSum                             = 0
	GeolocationFlag                                  = "geolocation"
	TendermintUnsubscribeAll                         = "unsubscribe_all"
	IndexNotFound                                    = -15
	MinValidAddressesForBlockingProbing              = 2
	BACKOFF_TIME_ON_FAILURE                          = 3 * time.Second
	BLOCKING_PROBE_SLEEP_TIME                        = 1000 * time.Millisecond // maximum amount of time to sleep before triggering probe, to scatter probes uniformly across chains
	BLOCKING_PROBE_TIMEOUT                           = time.Minute             // maximum time to wait for probe to complete before updating pairing
)

var AvailabilityPercentage sdk.Dec = sdk.NewDecWithPrec(1, 1) // TODO move to params pairing
const (
	PercentileToCalculateLatency = 0.9
	MinProvidersForSync          = 0.6
	OptimizerPerturbation        = 0.10
	LatencyThresholdStatic       = 1 * time.Second
	LatencyThresholdSlope        = 1 * time.Millisecond
	StaleEpochDistance           = 3 // relays done 3 epochs back are ready to be rewarded

)

func IsSessionSyncLoss(err error) bool {
	code := status.Code(err)
	return code == codes.Code(SessionOutOfSyncError.ABCICode())
}

func ConnectgRPCClient(ctx context.Context, address string, allowInsecure bool) (*grpc.ClientConn, error) {
	var tlsConf tls.Config
	if allowInsecure {
		tlsConf.InsecureSkipVerify = true // this will allow us to use self signed certificates in development.
	}
	credentials := credentials.NewTLS(&tlsConf)
	return grpc.DialContext(ctx, address, grpc.WithBlock(), grpc.WithTransportCredentials(credentials))
}

func GenerateSelfSignedCertificate() (tls.Certificate, error) {
	// Generate a private key
	utils.LavaFormatWarning("Warning: Using Self signed certificate is not recommended, this will not allow https connections to be established", nil)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create a self-signed certificate template
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Generate the self-signed certificate using the private key and template
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create a tls.Certificate using the private key and certificate bytes
	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privateKey,
	}

	return cert, nil
}

func GetCaCertificate(serverCertPath, serverKeyPath string) (*tls.Config, error) {
	serverCert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		ClientAuth:   tls.NoClientCert,
		Certificates: []tls.Certificate{serverCert},
	}, nil
}

func GetTlsConfig(networkAddress NetworkAddressData) *tls.Config {
	var tlsConfig *tls.Config
	var err error
	if networkAddress.CertPem != "" {
		utils.LavaFormatInfo("Running with TLS certificate", utils.Attribute{Key: "cert", Value: networkAddress.CertPem}, utils.Attribute{Key: "key", Value: networkAddress.KeyPem})
		tlsConfig, err = GetCaCertificate(networkAddress.CertPem, networkAddress.KeyPem)
		if err != nil {
			utils.LavaFormatFatal("failed to generate TLS certificate", err)
		}
	} else {
		cert, err := GenerateSelfSignedCertificate()
		if err != nil {
			utils.LavaFormatFatal("failed to generate TLS certificate", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	return tlsConfig
}
