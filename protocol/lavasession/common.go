package lavasession

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"golang.org/x/exp/slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/keeper/scores"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

const (
	MaxConsecutiveConnectionAttempts                 = 5
	TimeoutForEstablishingAConnection                = 1500 * time.Millisecond // 1.5 seconds
	MaxSessionsAllowedPerProvider                    = 1000                    // Max number of sessions allowed per provider
	MaxAllowedBlockListedSessionPerProvider          = MaxSessionsAllowedPerProvider / 3
	MaximumNumberOfFailuresAllowedPerConsumerSession = 15
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
	unixPrefix                                       = "unix:"
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
	return code == codes.Code(SessionOutOfSyncError.ABCICode()) || sdkerrors.IsOf(err, SessionOutOfSyncError)
}

func ConnectGRPCClient(ctx context.Context, address string, allowInsecure bool, skipTLS bool, allowCompression bool) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if skipTLS {
		// Skip TLS encryption completely
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		// Use TLS with optional server verification
		var tlsConf tls.Config
		if allowInsecure {
			tlsConf.InsecureSkipVerify = true // Allows self-signed certificates
		}
		credentials := credentials.NewTLS(&tlsConf)
		opts = append(opts, grpc.WithTransportCredentials(credentials))
	}

	opts = append(opts, grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(chainproxy.MaxCallRecvMsgSize)))

	if strings.HasPrefix(address, unixPrefix) {
		// Unix socket
		socketPath := strings.TrimPrefix(address, unixPrefix)
		opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		}))
	} else {
		// TCP socket
		opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.Dial("tcp", addr)
		}))
	}

	// allow gzip compression for grpc.
	if allowCompression {
		opts = append(opts, grpc.WithDefaultCallOptions(
			grpc.UseCompressor(gzip.Name), // Use gzip compression for provider consumer communication
		))
	}

	conn, err := grpc.DialContext(ctx, address, opts...)
	return conn, err
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

func GetSelfSignedConfig() (*tls.Config, error) {
	cert, err := GenerateSelfSignedCertificate()
	if err != nil {
		return nil, utils.LavaFormatError("failed to generate TLS certificate", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
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
		tlsConfig, err = GetSelfSignedConfig()
		if err != nil {
			utils.LavaFormatFatal("failed GetSelfSignedConfig", err)
		}
	}
	return tlsConfig
}

func SortByGeolocations(pairingEndpoints []*Endpoint, currentGeo planstypes.Geolocation) {
	latencyToGeo := func(a, b planstypes.Geolocation) uint64 {
		_, latency := scores.CalcGeoLatency(a, []planstypes.Geolocation{b})
		return latency
	}

	// sort the endpoints by geolocation relevance:
	lessFunc := func(a *Endpoint, b *Endpoint) bool {
		latencyA := int(latencyToGeo(a.Geolocation, currentGeo))
		latencyB := int(latencyToGeo(b.Geolocation, currentGeo))
		return latencyA < latencyB
	}
	slices.SortStableFunc(pairingEndpoints, lessFunc)
}
