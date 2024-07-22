package types

type PairingCacheKey struct {
	Project string
	ChainID string
	Epoch   uint64
}

func NewPairingCacheKey(project string, chainID string, epoch uint64) PairingCacheKey {
	return PairingCacheKey{
		Project: project,
		ChainID: chainID,
		Epoch:   epoch,
	}
}
