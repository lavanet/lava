package common

import (
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	CONSISTENCY_SELECT_ALLPROVIDERS = 1
	NOSTATE                         = 0
)

func GetExtensionNames(extensionCollection []*spectypes.Extension) (extensions []string) {
	for _, extension := range extensionCollection {
		extensions = append(extensions, extension.Name)
	}
	return
}
