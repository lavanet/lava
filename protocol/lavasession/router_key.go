package lavasession

import (
	"sort"
	"strings"
)

const (
	sep = "|"
)

type RouterKey string

func NewRouterKey(extensions []string) RouterKey {
	// make sure addons have no repetitions
	uniqueExtensions := map[string]struct{}{}
	for _, extension := range extensions {
		uniqueExtensions[extension] = struct{}{}
	}
	uniqueExtensionsSlice := []string{}
	for addon := range uniqueExtensions { // we are sorting this anyway so we don't have to keep order
		uniqueExtensionsSlice = append(uniqueExtensionsSlice, addon)
	}
	sort.Strings(uniqueExtensionsSlice)
	return RouterKey(sep + strings.Join(uniqueExtensionsSlice, sep) + sep)
}
