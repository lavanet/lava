package types

import (
	fmt "fmt"
	"path/filepath"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func ReadYaml(filePath string, primaryKey string, content interface{}, hooks []EnumDecodeHookFuncType) error {
	configPath, configName := filepath.Split(filePath)
	if configPath == "" {
		configPath = "."
	}
	viper.SetConfigName(configName)
	viper.SetConfigType("yml")
	viper.AddConfigPath(configPath)

	// handle proto enums
	opts := []viper.DecoderConfigOption{
		func(dc *mapstructure.DecoderConfig) {
			dc.ErrorUnused = true
		},
	}
	if len(hooks) != 0 {
		hookFuncs := make([]mapstructure.DecodeHookFunc, 0, len(hooks))
		for _, h := range hooks {
			hookFuncs = append(hookFuncs, mapstructure.ComposeDecodeHookFunc(h))
		}
		opts = append(opts, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(hookFuncs...)))
	}

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	err = viper.GetViper().UnmarshalKey(primaryKey, content, opts...)
	if err != nil {
		return err
	}

	return nil
}

// EnumDecodeHookFuncType represents the signature of enum decode hook functions.
type EnumDecodeHookFuncType func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error)

// EnumParseFunc represents the signature of enum parse functions.
type EnumParseFunc func(enumType interface{}, strVal string) (interface{}, error)

// EnumDecodeHook decodes an enum value based on the provided enumType.
func EnumDecodeHook(enumType interface{}, enumParser EnumParseFunc) EnumDecodeHookFuncType {
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
