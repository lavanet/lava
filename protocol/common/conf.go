package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Test_mode_ctx_key struct{}

const (
	PlainTextConnection                = "allow-plaintext-connection"
	EndpointsConfigName                = "endpoints"
	StaticProvidersConfigName          = "static-providers"
	SaveConfigFlagName                 = "save-conf"
	GeolocationFlag                    = "geolocation"
	TestModeFlagName                   = "test-mode"
	MaximumConcurrentProvidersFlagName = "concurrent-providers"
	StatusCodeMetadataKey              = "status-code"
	VersionMetadataKey                 = "lavap-version"
	TimeOutForFetchingLavaBlocksFlag   = "timeout-for-fetching-lava-blocks"
)

func ParseEndpointArgs(endpoint_strings, yaml_config_properties []string, endpointsConfigName string) (viper_endpoints *viper.Viper, err error) {
	numFieldsInConfig := len(yaml_config_properties)
	viper_endpoints = viper.New()
	if len(endpoint_strings)%numFieldsInConfig != 0 {
		return nil, fmt.Errorf("invalid endpoint_strings length %d, needs to divide by %d without residue", len(endpoint_strings), numFieldsInConfig)
	}
	endpoints := []map[string]interface{}{}
	for idx := 0; idx < len(endpoint_strings); idx += numFieldsInConfig {
		toAdd := map[string]interface{}{}
		for inner_idx := 0; inner_idx < numFieldsInConfig; inner_idx++ {
			config_property_name := yaml_config_properties[inner_idx]
			var property_elements []string
			if strings.Contains(yaml_config_properties[inner_idx], ".") {
				// means to set the config in a dictionary property
				property_elements = strings.Split(yaml_config_properties[inner_idx], ".")
				config_property_name = property_elements[0]
				property_elements = property_elements[1:]
			}

			setPropertyElements := func(config_elements ...interface{}) {
				modified_elements := make([]interface{}, len(config_elements))
				if len(property_elements) > 0 {
					// means we need to set the value inside a property
					for idx, element := range config_elements {
						modified_element_as_map := map[string]interface{}{}
						// set one property key, corresponding index
						index_to_take := idx
						if index_to_take >= len(property_elements) {
							index_to_take = 0 // take the first element if there are more instances than elements
						}
						property_element := property_elements[index_to_take]
						modified_element_as_map[property_element] = element
						modified_elements[idx] = modified_element_as_map
					}
				} else {
					// put them as is
					modified_elements = config_elements
				}
				var element_to_set interface{} = modified_elements
				if len(modified_elements) == 1 {
					element_to_set = modified_elements[0]
				}
				toAdd[config_property_name] = element_to_set
			}

			if strings.Contains(endpoint_strings[idx+inner_idx], ",") {
				separated_arguments := strings.Split(endpoint_strings[idx+inner_idx], ",")
				config_elements := make([]interface{}, len(separated_arguments))
				for element_idx, config_element := range separated_arguments {
					config_elements[element_idx] = config_element
				}
				setPropertyElements(config_elements...)
			} else {
				setPropertyElements(endpoint_strings[idx+inner_idx])
			}
		}
		endpoints = append(endpoints, toAdd)
	}

	viper_endpoints.Set(endpointsConfigName, endpoints)
	return viper_endpoints, err
}

func IsTestMode(ctx context.Context) bool {
	test_mode, ok := ctx.Value(Test_mode_ctx_key{}).(bool)
	return ok && test_mode
}
