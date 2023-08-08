package yaml

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

func DecodeFile(path string, key string, result interface{}, hooks []mapstructure.DecodeHookFunc, unset, unused *[]string) error {
	input, err := os.ReadFile(path + ".yml")
	if err != nil {
		return err
	}

	return Decode(string(input), key, result, hooks, unset, unused)
}

func Decode(input string, key string, result interface{}, hooks []mapstructure.DecodeHookFunc, unset, unused *[]string) error {
	var config map[string]interface{}

	err := yaml.Unmarshal([]byte(input), &config)
	if err != nil {
		return err
	}

	if config == nil {
		return fmt.Errorf("yaml: empty input")
	}

	// key may be nested, as in: "Policy.chain_policies", so need to dig the
	// correct "entry-point" into the map.
	for _, elem := range strings.Split(key, ".") {
		if config == nil {
			return fmt.Errorf("yaml: key not found: %q (of %q)", elem, key)
		}
		value, ok := config[elem]
		if !ok {
			return fmt.Errorf("yaml: key not found: %q (of %q)", elem, key)
		}
		config, ok = value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("yaml: key not found: %q (of %q)", elem, key)
		}
	}

	decoderHookFunc := mapstructure.ComposeDecodeHookFunc(hooks...)
	var decoderMetadata mapstructure.Metadata

	decoderConfig := mapstructure.DecoderConfig{
		DecodeHook: decoderHookFunc,
		Metadata:   &decoderMetadata,
		Result:     result,
	}

	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return err
	}

	err = decoder.Decode(config)
	if err != nil {
		return err
	}

	if unset != nil {
		*unset = decoderMetadata.Unset
	}
	if unused != nil {
		*unused = decoderMetadata.Unused
	}

	return nil
}

func SetDefaultValues(input map[string]interface{}, result interface{}) error {
	return mapstructure.Decode(input, result)
}

// EnumParseFunc represents the signature of enum parse functions.
type EnumParseFunc func(enumType interface{}, strVal string) (interface{}, error)

// EnumDecodeHook decodes an enum value based on the provided enumType.
func EnumDecodeHook(enumType interface{}, enumParser EnumParseFunc) mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if t == reflect.TypeOf(enumType) {
			strVal, ok := data.(string)
			if !ok {
				return data, nil
			}

			enumVal, err := enumParser(enumType, strVal)
			if err != nil {
				return nil, fmt.Errorf("failed to parse enum value: %w", err)
			}

			return enumVal, nil
		}

		// Return data as is for non-enum types
		return data, nil
	}
}
