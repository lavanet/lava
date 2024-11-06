package lavasession

import (
	"sort"
	"strconv"
	"strings"

	"github.com/lavanet/lava/v4/utils/lavaslices"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

const (
	RouterKeySeparator = "|"
	methodRouteSep     = "method-route:"
	internalPathSep    = "internal-path:"
)

type RouterKey struct {
	methodsRouteUniqueKey int
	uniqueExtensions      []string
	internalPath          string
}

func NewRouterKey(extensions []string) RouterKey {
	routerKey := RouterKey{}
	routerKey.SetExtensions(extensions)
	return routerKey
}

func NewRouterKeyFromExtensions(extensions []*spectypes.Extension) RouterKey {
	extensionsStr := lavaslices.Map(extensions, func(extension *spectypes.Extension) string {
		return extension.Name
	})

	return NewRouterKey(extensionsStr)
}

func GetEmptyRouterKey() RouterKey {
	return NewRouterKey([]string{})
}

func (rk *RouterKey) SetExtensions(extensions []string) {
	// make sure addons have no repetitions
	uniqueExtensions := map[string]struct{}{} // init with the empty extension
	if len(extensions) == 0 {
		uniqueExtensions[""] = struct{}{}
	} else {
		for _, extension := range extensions {
			uniqueExtensions[extension] = struct{}{}
		}
	}

	uniqueExtensionsSlice := []string{}
	for addon := range uniqueExtensions { // we are sorting this anyway so we don't have to keep order
		uniqueExtensionsSlice = append(uniqueExtensionsSlice, addon)
	}

	sort.Strings(uniqueExtensionsSlice)

	rk.uniqueExtensions = uniqueExtensionsSlice
}

func (rk *RouterKey) ApplyMethodsRoute(routeNum int) {
	rk.methodsRouteUniqueKey = routeNum
}

func (rk *RouterKey) ApplyInternalPath(internalPath string) {
	rk.internalPath = internalPath
}

func (rk RouterKey) HasExtension(extension string) bool {
	return lavaslices.Contains(rk.uniqueExtensions, extension)
}

func (rk RouterKey) String() string {
	// uniqueExtensions are sorted on init
	retStr := rk.uniqueExtensions
	if len(retStr) == 0 {
		retStr = append(retStr, "")
	}

	// if we have a route number, we add it to the key
	if rk.methodsRouteUniqueKey != 0 {
		retStr = append(retStr, methodRouteSep+strconv.FormatInt(int64(rk.methodsRouteUniqueKey), 10))
	}

	// if we have an internal path, we add it to the key
	if rk.internalPath != "" {
		retStr = append(retStr, internalPathSep+rk.internalPath)
	}

	return RouterKeySeparator + strings.Join(retStr, RouterKeySeparator) + RouterKeySeparator
}
