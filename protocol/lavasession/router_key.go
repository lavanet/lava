package lavasession

import (
	"sort"
	"strings"

	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const (
	sep = "|"
)

type RouterKey string

func newRouterKeyInner(uniqueExtensions map[string]struct{}) RouterKey {
	uniqueExtensionsSlice := []string{}
	for addon := range uniqueExtensions { // we are sorting this anyway so we don't have to keep order
		uniqueExtensionsSlice = append(uniqueExtensionsSlice, addon)
	}
	sort.Strings(uniqueExtensionsSlice)
	return RouterKey(sep + strings.Join(uniqueExtensionsSlice, sep) + sep)
}

func NewRouterKey(extensions []string) RouterKey {
	// make sure addons have no repetitions
	uniqueExtensions := map[string]struct{}{}
	for _, extension := range extensions {
		uniqueExtensions[extension] = struct{}{}
	}
	return newRouterKeyInner(uniqueExtensions)
}

func NewRouterKeyFromExtensions(extensions []*spectypes.Extension) RouterKey {
	// make sure addons have no repetitions
	uniqueExtensions := map[string]struct{}{}
	for _, extension := range extensions {
		uniqueExtensions[extension.Name] = struct{}{}
	}
	return newRouterKeyInner(uniqueExtensions)
}

func GetEmptyRouterKey() RouterKey {
	return NewRouterKey([]string{})
}
