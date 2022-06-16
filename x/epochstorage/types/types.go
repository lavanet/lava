package types

const TokenDenom = "ulava"

//returns a deep copy of the stake storage with the same index
func (stksto StakeStorage) Copy() (returnedStorage StakeStorage) {
	returnedStorage = StakeStorage{Index: stksto.Index, StakeEntries: []StakeEntry{}}
	for _, stakeEntry := range stksto.StakeEntries {
		endpoints := make([]Endpoint, len(stakeEntry.Endpoints))
		copy(endpoints, stakeEntry.Endpoints)
		newStakeEntry := StakeEntry{
			Stake:       stakeEntry.Stake,
			Address:     stakeEntry.Address,
			Deadline:    stakeEntry.Deadline,
			Endpoints:   endpoints,
			Geolocation: stakeEntry.Geolocation,
			Chain:       stakeEntry.Chain,
		}
		returnedStorage.StakeEntries = append(returnedStorage.StakeEntries, newStakeEntry)
	}
	return
}
