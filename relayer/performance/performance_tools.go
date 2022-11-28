//go:build performance_tools

package performance

import (
	"fmt"
	"github.com/lavanet/performance_tools/cache"
)

var usedCache cache.Cache

func Init() {
	usedCache = cache.InitCache()
	fmt.Println("\nPerformance Tools Cache Initialized\n")
}

func GetEntry(request []byte, block uint64, blockHash []byte, chainID string, apiInterface string) (exists bool, response []byte) {
	return usedCache.GetEntry(request, block, blockHash, chainID, apiInterface)

}

func SetEntry(request []byte, block uint64, blockHash []byte, chainID string, apiInterface string, response []byte) {
	usedCache.SetEntry(request, block, blockHash, chainID, apiInterface, response)
}
