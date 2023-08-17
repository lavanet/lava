package common

import (
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func GetExtensionNames(extensionCollection []*spectypes.Extension) (extensions []string) {
	for _, extension := range extensionCollection {
		extensions = append(extensions, extension.Name)
	}
	return
}
