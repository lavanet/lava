package types

const TokenDenom = "ulava"

const (
	NewEpochEventName            = "new_epoch"
	EarliestEpochEventName       = "earliest_epoch"
	FixatedParamChangeEventName  = "fixated_params_change"
	FixatedParamCleanedEventName = "fixated_params_clean"
	StakeStorageKeyUnstakeConst  = "Unstake"
)

// returns a deep copy of the stake storage with the same index
func (stksto StakeStorage) Copy() (returnedStorage StakeStorage) {
	returnedStorage = StakeStorage{Index: stksto.Index, StakeEntries: []StakeEntry{}, EpochBlockHash: stksto.EpochBlockHash}
	for _, stakeEntry := range stksto.StakeEntries {
		endpoints := make([]Endpoint, len(stakeEntry.Endpoints))
		copy(endpoints, stakeEntry.Endpoints)
		newStakeEntry := StakeEntry{
			Stake:             stakeEntry.Stake,
			Address:           stakeEntry.Address,
			StakeAppliedBlock: stakeEntry.StakeAppliedBlock,
			Endpoints:         endpoints,
			Geolocation:       stakeEntry.Geolocation,
			Chain:             stakeEntry.Chain,
		}
		returnedStorage.StakeEntries = append(returnedStorage.StakeEntries, newStakeEntry)
	}
	return
}
