package common

import (
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const (
	CONSISTENCY_SELECT_ALL_PROVIDERS = 1
	NO_STATE                         = 0
)

func GetExtensionNames(extensionCollection []*spectypes.Extension) (extensions []string) {
	for _, extension := range extensionCollection {
		extensions = append(extensions, extension.Name)
	}
	return
}
