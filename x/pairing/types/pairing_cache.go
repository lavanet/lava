package types

import "strconv"

var (
	PairingRelayCachePrefix = []byte("PairingRelayCache")
)

func NewPairingQueryCacheKey(project string, chainID string, epoch uint64) string {
	epochStr := strconv.FormatUint(epoch, 10)
	return project + " " + chainID + " " + epochStr
}

func NewPairingRelayCacheKey(project string, chainID string, provider string) string {
	return project + " " + chainID + " " + provider
}
