package common

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	URL_QUERY_PARAMETERS_SEPARATOR_FROM_PATH        = "?"
	URL_QUERY_PARAMETERS_SEPARATOR_OTHER_PARAMETERS = "&"
	IP_FORWARDING_HEADER_NAME                       = "X-Forwarded-For"
	PROVIDER_ADDRESS_HEADER_NAME                    = "Lava-Provider-Address"
)

type NodeUrl struct {
	Url          string        `yaml:"url,omitempty" json:"url,omitempty" mapstructure:"url"`
	InternalPath string        `yaml:"internal-path,omitempty" json:"internal-path,omitempty" mapstructure:"internal-path"`
	AuthConfig   AuthConfig    `yaml:"auth-config,omitempty" json:"auth-config,omitempty" mapstructure:"auth-config"`
	IpForwarding bool          `yaml:"ip-forwarding,omitempty" json:"ip-forwarding,omitempty" mapstructure:"ip-forwarding"`
	Timeout      time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" mapstructure:"timeout"`
	Addons       []string      `yaml:"addons,omitempty" json:"addons,omitempty" mapstructure:"addons"`
}

func (nurl NodeUrl) String() string {
	urlStr := nurl.UrlStr()

	if len(nurl.Addons) > 0 {
		return urlStr + "(" + strings.Join(nurl.Addons, ",") + ")"
	}
	return urlStr
}

func (nurl *NodeUrl) UrlStr() string {
	parsedURL, err := url.Parse(nurl.Url)
	if err != nil {
		return nurl.Url
	}
	parsedURL.User = nil
	return parsedURL.String()
}

func (url *NodeUrl) SetAuthHeaders(ctx context.Context, headerSetter func(string, string)) {
	for header, headerValue := range url.AuthConfig.AuthHeaders {
		headerSetter(header, headerValue)
	}
}

func (url *NodeUrl) SetIpForwardingIfNecessary(ctx context.Context, headerSetter func(string, string)) {
	if !url.IpForwarding {
		// not necessary
		return
	}
	peerAddress := GetIpFromGrpcContext(ctx)
	if peerAddress != "" {
		headerSetter(IP_FORWARDING_HEADER_NAME, peerAddress)
	}
}

func (url *NodeUrl) LowerContextTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if url == nil || url.Timeout <= 0 {
		return LowerContextTimeout(ctx, timeout)
	}
	return LowerContextTimeout(ctx, timeout+url.Timeout)
}

type AuthConfig struct {
	AuthHeaders map[string]string `yaml:"auth-headers,omitempty" json:"auth-headers,omitempty" mapstructure:"auth-headers"`
	AuthQuery   string            `yaml:"auth-query,omitempty" json:"auth-query,omitempty" mapstructure:"auth-query"`
	UseTLS      bool              `yaml:"use-tls,omitempty" json:"use-tls,omitempty" mapstructure:"use-tls"`
	KeyPem      string            `yaml:"key-pem,omitempty" json:"key-pem,omitempty" mapstructure:"key-pem"`
	CertPem     string            `yaml:"cert-pem,omitempty" json:"cert-pem,omitempty" mapstructure:"cert-pem"`
	CaCert      string            `yaml:"cacert-pem,omitempty" json:"cacert-pem,omitempty" mapstructure:"cacert-pem"`
}

func (ac *AuthConfig) GetUseTls() bool {
	if ac == nil {
		return false
	}
	return ac.UseTLS
}

// File containing client certificate (public key), to present to the
// server. + File containing client private key, to present to the server.
func (ac *AuthConfig) GetLoadingCertificateParams() (string, string) {
	if ac == nil {
		return "", ""
	}
	if ac.KeyPem == "" || ac.CertPem == "" {
		return "", ""
	}
	return ac.KeyPem, ac.CertPem
}

// File containing trusted root certificates for verifying the server.
func (ac *AuthConfig) GetCaCertificateParams() string {
	if ac == nil {
		return ""
	}
	return ac.CaCert
}

func (ac *AuthConfig) AddAuthPath(url string) string {
	// there is no auth provided
	if ac.AuthQuery == "" {
		return url
	}
	// AuthPath is expected to be added as a uri optional parameter
	if strings.Contains(url, "?") {
		// there are already optional parameters
		return url + URL_QUERY_PARAMETERS_SEPARATOR_OTHER_PARAMETERS + ac.AuthQuery
	}
	// path doesn't have query parameters
	return url + URL_QUERY_PARAMETERS_SEPARATOR_FROM_PATH + ac.AuthQuery
}

func ValidateEndpoint(endpoint, apiInterface string) error {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC, spectypes.APIInterfaceTendermintRPC, spectypes.APIInterfaceRest:
		parsedUrl, err := url.Parse(endpoint)
		if err != nil {
			return utils.LavaFormatError("could not parse node url", err, utils.Attribute{Key: "url", Value: endpoint}, utils.Attribute{Key: "apiInterface", Value: apiInterface})
		}
		switch parsedUrl.Scheme {
		case "http", "https":
			return nil
		case "ws", "wss":
			return nil
		default:
			return utils.LavaFormatError("URL scheme should be websocket (ws/wss) or (http/https), got: "+parsedUrl.Scheme, nil, utils.Attribute{Key: "apiInterface", Value: apiInterface})
		}
	case spectypes.APIInterfaceGrpc:
		if endpoint == "" {
			return utils.LavaFormatError("invalid grpc URL, empty", nil)
		}
		parsedUrl, err := url.Parse(endpoint)
		if err == nil {
			// user provided a valid url with a scheme
			if parsedUrl.Scheme != "" && strings.Contains(endpoint, "/") {
				return utils.LavaFormatError("grpc URL scheme should be empty and it is not, endpoint definition example: 127.0.0.1:9090 -or- my-node.com/grpc", nil, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "scheme", Value: parsedUrl.Scheme})
			}
			return nil
		} else {
			// user provided no scheme, make sure before returning its correct
			_, err = url.Parse("//" + endpoint)
			if err == nil {
				return nil
			}
			return utils.LavaFormatError("invalid grpc URL, usage example: 127.0.0.1:9090 or my-node.com/grpc", nil, utils.Attribute{Key: "apiInterface", Value: apiInterface}, utils.Attribute{Key: "url", Value: endpoint})
		}
	default:
		return utils.LavaFormatError("unsupported apiInterface", nil, utils.Attribute{Key: "apiInterface", Value: apiInterface})
	}
}

type ConflictHandlerInterface interface {
	ConflictAlreadyReported() bool
	StoreConflictReported()
}

type RelayResult struct {
	Request         *pairingtypes.RelayRequest
	Reply           *pairingtypes.RelayReply
	ProviderAddress string
	ReplyServer     *pairingtypes.Relayer_RelaySubscribeClient
	Finalized       bool
	ConflictHandler ConflictHandlerInterface
	StatusCode      int
}

func (rr *RelayResult) GetReplyServer() *pairingtypes.Relayer_RelaySubscribeClient {
	if rr == nil {
		return nil
	}
	return rr.ReplyServer
}

func (rr *RelayResult) GetReply() *pairingtypes.RelayReply {
	if rr == nil {
		return nil
	}
	return rr.Reply
}

func (rr *RelayResult) GetStatusCode() int {
	if rr == nil {
		return 0
	}
	return rr.StatusCode
}

func (rr *RelayResult) GetProvider() string {
	if rr == nil {
		return ""
	}
	return rr.ProviderAddress
}

func GetIpFromGrpcContext(ctx context.Context) string {
	// peers should be always available
	grpcPeer, exists := peer.FromContext(ctx)
	if exists {
		return grpcPeer.Addr.String()
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		ipforwardingHeader := md.Get(IP_FORWARDING_HEADER_NAME)
		if len(ipforwardingHeader) > 0 {
			return ipforwardingHeader[0]
		}
	}
	return ""
}

func GetTokenFromGrpcContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		ipforwardingHeader := md.Get(IP_FORWARDING_HEADER_NAME)
		if len(ipforwardingHeader) > 0 {
			return ipforwardingHeader[0]
		}
	}
	grpcPeer, exists := peer.FromContext(ctx)
	if exists {
		return grpcPeer.Addr.String()
	}
	return ""
}

func GetUniqueToken(consumerAddress string, ip string) string {
	data := []byte(consumerAddress + ip)
	return base64.StdEncoding.EncodeToString(sigs.HashMsg(data))
}
