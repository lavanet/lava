package common

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const (
	EndpointsConfigName = "endpoints"
	SaveConfigFlagName  = "save-conf"
	GeolocationFlag     = "geolocation"
)

func ParseEndpointArgs(endpoint_strings []string, yaml_config_properties []string, endpointsConfigName string) (viper_endpoints *viper.Viper, err error) {
	numFieldsInConfig := len(yaml_config_properties)
	viper_endpoints = viper.New()
	if len(endpoint_strings)%numFieldsInConfig != 0 {
		return nil, fmt.Errorf("invalid endpoint_strings length %d, needs to divide by %d without residue", len(endpoint_strings), numFieldsInConfig)
	}
	endpoints := []map[string]interface{}{}
	for idx := 0; idx < len(endpoint_strings); idx += numFieldsInConfig {
		toAdd := map[string]interface{}{}
		for inner_idx := 0; inner_idx < numFieldsInConfig; inner_idx++ {
			if strings.Contains(endpoint_strings[idx+inner_idx], ",") {
				toAdd[yaml_config_properties[inner_idx]] = strings.Split(endpoint_strings[idx+inner_idx], ",")
			} else {
				toAdd[yaml_config_properties[inner_idx]] = endpoint_strings[idx+inner_idx]
			}
		}
		endpoints = append(endpoints, toAdd)
	}

	viper_endpoints.Set(endpointsConfigName, endpoints)
	return
}
