package types

import (
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func ReadYaml(filePath string, primaryKey string, content interface{}) error {
	configPath, configName := filepath.Split(filePath)
	if configPath == "" {
		configPath = "."
	}
	viper.SetConfigName(configName)
	viper.SetConfigType("yml")
	viper.AddConfigPath(configPath)

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	err = viper.GetViper().UnmarshalKey(primaryKey, content, func(dc *mapstructure.DecoderConfig) {
		dc.ErrorUnset = true
		dc.ErrorUnused = true
	})
	if err != nil {
		return err
	}

	return nil
}
