package common

import (
	"net/url"
	"strings"

	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func ValidateEndpoint(endpoint string, apiInterface string) error {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC, spectypes.APIInterfaceTendermintRPC, spectypes.APIInterfaceRest:
		parsedUrl, err := url.Parse(endpoint)
		if err != nil {
			return utils.LavaFormatError("could not parse node url", err, &map[string]string{"url": endpoint, "apiInterface": apiInterface})
		}
		switch parsedUrl.Scheme {
		case "http", "https":
			return nil
		case "ws", "wss":
			return nil
		default:
			return utils.LavaFormatError("URL scheme should be websocket (ws/wss) or (http/https), got: "+parsedUrl.Scheme, nil, &map[string]string{"apiInterface": apiInterface})
		}
	case spectypes.APIInterfaceGrpc:
		parsedUrl, err := url.Parse(endpoint)
		if err == nil {
			// user provided a valid url with a scheme
			if parsedUrl.Scheme != "" && strings.Contains(endpoint, "/") {
				return utils.LavaFormatError("grpc URL scheme should be empty and it is not, endpoint definition example: 127.0.0.1:9090 -or- my-node.com/grpc", nil, &map[string]string{"apiInterface": apiInterface, "scheme": parsedUrl.Scheme})
			}
			return nil
		} else {
			// user provided no scheme, make sure before returning its correct
			_, err = url.Parse("//" + endpoint)
			if err == nil {
				return nil
			}
			return utils.LavaFormatError("invalid grpc URL, usage example: 127.0.0.1:9090 or my-node.com/grpc", nil, &map[string]string{"apiInterface": apiInterface, "url": endpoint})
		}
	default:
		return utils.LavaFormatError("unsupported apiInterface", nil, &map[string]string{"apiInterface": apiInterface})
	}
}
