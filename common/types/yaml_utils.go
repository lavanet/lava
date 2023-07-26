package types

import (
	fmt "fmt"
	"path/filepath"
	"reflect"

	"github.com/lavanet/lava/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func ReadYaml(filePath string, primaryKey string, content interface{}, hooks []EnumDecodeHookFuncType, allowMissingFields bool) (missingFields []string, err error) {
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

	err = viper.ReadInConfig()
	if err != nil {
		return missingFields, err
	}

	err = viper.GetViper().UnmarshalKey(primaryKey, content, opts...)
	if err != nil {
		return missingFields, err
	}

	if allowMissingFields {
		missingFields = findMissingFields(content)
	}

	return missingFields, nil
}

func findMissingFields(content interface{}) []string {
	var missingFields []string

	expectedFields := make(map[string]bool)
	contentValue := reflect.ValueOf(content)
	contentType := contentValue.Type()

	// Get the type of the struct
	if contentType.Kind() == reflect.Ptr {
		contentType = contentType.Elem()
		contentValue = contentValue.Elem()
	}

	// panic:ok Ensure content is a struct or a pointer to a struct
	if contentType.Kind() != reflect.Struct {
		utils.LavaFormatPanic("findMissingFields was called with a non-struct type", fmt.Errorf("cannot read yaml"),
			utils.Attribute{Key: "type", Value: contentType.Kind().String()},
		)
	}

	// Extract the expected field names from the struct
	for i := 0; i < contentType.NumField(); i++ {
		field := contentType.Field(i)
		fieldName := field.Name
		expectedFields[fieldName] = true
	}

	// Check if each expected field is present in the unmarshaled content
	for fieldName := range expectedFields {
		fieldValue := contentValue.FieldByName(fieldName)

		// note that this check will always treat zero value fields as missing (which will make them get the declared default value)
		if !fieldValue.IsValid() || fieldValue.IsZero() {
			missingFields = append(missingFields, fieldName)
		}
	}

	return missingFields
}

func SetDefaultValues(content interface{}, defaultValues map[string]interface{}) {
	contentValue := reflect.ValueOf(content).Elem()
	contentType := contentValue.Type()

	for fieldName, fieldValue := range defaultValues {
		field, found := contentType.FieldByName(fieldName)
		if found {
			fieldValue := reflect.ValueOf(fieldValue)
			fieldType := field.Type

			if fieldValue.IsValid() && fieldValue.Type().AssignableTo(fieldType) {
				contentFieldValue := contentValue.FieldByName(fieldName)
				if contentFieldValue.IsValid() && contentFieldValue.CanSet() {
					if fieldValue.Type() != fieldType {
						fieldValue = fieldValue.Convert(fieldType)
					}
					contentFieldValue.Set(fieldValue)
				}
			}
		}
	}
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
