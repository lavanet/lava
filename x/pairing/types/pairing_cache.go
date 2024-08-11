package types

import "strconv"

var PairingRelayCachePrefix = []byte("PairingRelayCache")

func NewPairingCacheKey(project string, chainID string, epoch uint64) string {
	epochStr := strconv.FormatUint(epoch, 10)
	return project + " " + chainID + " " + epochStr
}
