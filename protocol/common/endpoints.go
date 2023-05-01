package common

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/peer"
)

const (
	URL_QUERY_PARAMETERS_SEPARATOR_FROM_PATH        = "?"
	URL_QUERY_PARAMETERS_SEPARATOR_OTHER_PARAMETERS = "&"
	IP_FORWARDING_HEADER_NAME                       = "X-Forwarded-For"
)

type NodeUrl struct {
	Url          string        `yaml:"url,omitempty" json:"url,omitempty" mapstructure:"url"`
	AuthConfig   AuthConfig    `yaml:"auth-config,omitempty" json:"auth-config,omitempty" mapstructure:"auth-config"`
	IpForwarding bool          `yaml:"ip-forwarding,omitempty" json:"ip-forwarding,omitempty" mapstructure:"ip-forwarding"`
	Timeout      time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" mapstructure:"timeout"`
}

func (url *NodeUrl) String() string {
	if url == nil {
		return ""
	}
	return url.Url
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
	grpcPeer, exists := peer.FromContext(ctx)
	if exists {
		peerAddress := grpcPeer.Addr.String()
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
}

func (ac *AuthConfig) GetUseTls() bool {
	if ac == nil {
		return false
	}
	return ac.UseTLS
}

func (ac *AuthConfig) GetLoadingCertificateParams() (string, string) {
	if ac == nil {
		return "", ""
	}
	if ac.KeyPem == "" || ac.CertPem == "" {
		return "", ""
	}
	return ac.KeyPem, ac.CertPem
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

func ValidateEndpoint(endpoint string, apiInterface string) error {
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
