package lavasession

import (
	"sort"
	"strconv"
	"strings"
)

const (
	sep            = "|"
	methodRouteSep = "method-route:"
)

type RouterKey string

func (rk *RouterKey) ApplyMethodsRoute(routeNum int) RouterKey {
	additionalPath := strconv.FormatInt(int64(routeNum), 10)
	return RouterKey(string(*rk) + methodRouteSep + additionalPath)
}

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

func GetEmptyRouterKey() RouterKey {
	return NewRouterKey([]string{})
}
