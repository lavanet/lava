package rpcsmartrouter

import (
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	"github.com/spf13/viper"
)

// ParseStaticProviderEndpoints parses static provider configuration into extended endpoint types.
func ParseStaticProviderEndpoints(viperEndpoints *viper.Viper, endpointsConfigName string, geolocation uint64) (endpoints []*lavasession.RPCStaticProviderEndpoint, err error) {
	err = viperEndpoints.UnmarshalKey(endpointsConfigName, &endpoints)
	if err != nil {
		utils.LavaFormatFatal("could not unmarshal extended endpoints", err, utils.Attribute{Key: "viper_endpoints", Value: viperEndpoints.AllSettings()})
	}
	for _, endpoint := range endpoints {
		endpoint.Geolocation = geolocation

		// Validate that the provider name is not empty
		if err := endpoint.Validate(); err != nil {
			return nil, utils.LavaFormatError("invalid provider configuration", err,
				utils.Attribute{Key: "chainID", Value: endpoint.ChainID},
				utils.Attribute{Key: "apiInterface", Value: endpoint.ApiInterface})
		}
	}
	return endpoints, err
}
