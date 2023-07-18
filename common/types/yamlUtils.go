package types

import (
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func ReadYaml(filePath string, primaryKey string, content interface{}, hooks []mapstructure.DecodeHookFuncType) error {
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
			dc.ErrorUnset = true
			dc.ErrorUnused = true
		},
	}
	if len(hooks) != 0 {
		for _, h := range hooks {
			opts = append(opts, viper.DecodeHook(h))
		}
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
