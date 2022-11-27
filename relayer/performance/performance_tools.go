//go:build pro

package performance

import (
	"fmt"
	"github.com/lavanet/performance-tools/cache"
)

var usedCache cache.Cache

func Init() {
	usedCache = cache.Cache.Init()
	fmt.Println("\nPerformance Tools Cache Initialized\n")
}

func GetEntry(request []byte, block uint64, blockHash []byte, chainID string, apiInterface string) (exists bool, response []byte) {
	return cache.Cache.GetEntry(request, block, blockHash, chainID, apiInterface)

}

func SetEntry(request []byte, block uint64, blockHash []byte, chainID string, apiInterface string, response []byte) {
	cache.Cache.SetEntry(request, block, blockHash, chainID, apiInterface, response)
}
