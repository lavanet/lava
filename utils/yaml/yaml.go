package yaml

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

func DecodeFile(path string, key string, result interface{}, hooks []mapstructure.DecodeHookFunc, unset, unused *[]string) error {
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return Decode(string(input), key, result, hooks, unset, unused)
}

func Decode(input string, key string, result interface{}, hooks []mapstructure.DecodeHookFunc, unset, unused *[]string) error {
	var config map[string]interface{}
	inputBytes := []byte(input)

	if isJSON(inputBytes) {
		err := json.Unmarshal(inputBytes, &config)
		if err != nil {
			return err
		}
	} else {
		err := yaml.Unmarshal(inputBytes, &config)
		if err != nil {
			return err
		}
	}

	if config == nil {
		return fmt.Errorf("yaml: empty input")
	}

	// get the desired section in the yaml/config per the given key
	config, result, err := configByKey(key, config, result)
	if err != nil {
		return err
	}

	var decoderMetadata mapstructure.Metadata
	decoderHookFunc := mapstructure.ComposeDecodeHookFunc(hooks...)

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

func isJSON(data []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(data, &js) == nil
}

func configByKey(key string, config map[string]interface{}, result interface{}) (map[string]interface{}, interface{}, error) {
	// key may be nested, as in: "Policy.chain_policies", so need to dig the
	// correct "entry-point" into the map.

	split := strings.Split(key, ".")

	for i, elem := range split {
		value, ok := config[elem]
		if !ok {
			return nil, nil, fmt.Errorf("yaml: key not found: %q (of %q)", elem, key)
		}

		// typically the next component is a nested map[string]interface{}; if not then
		// a) this must be the last component, b) it match the type of "result"
		config, ok = value.(map[string]interface{})
		if !ok {
			// this must be the last component
			if i < len(split)-1 {
				return nil, nil, fmt.Errorf("yaml: key not found: %q (of %q)", elem, key)
			}

			kind := reflect.ValueOf(value).Kind()

			// must match the type of "result"
			if kind != reflect.ValueOf(result).Elem().Kind() {
				return nil, nil, fmt.Errorf("yaml: key type mismatch: %q", key)
			}

			if kind == reflect.Slice {
				config = map[string]interface{}{
					"Result": value,
				}
				result = &struct {
					Result interface{}
				}{
					Result: result,
				}
			} else {
				return nil, nil, fmt.Errorf("yaml: unsupported type %v", kind)
			}
		}
	}

	return config, result, nil
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
