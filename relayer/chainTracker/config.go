package chaintracker

import "time"

type ChainTrackerConfig struct {
	ForkCallback      func() //a function to be called when a fork is detected
	NewLatestCallback func() //a function to be called when a new block is detected
	ServerAddress     string //if not empty will open up a grpc server for that address
	blocksToSave      uint64
	averageBlockTime  time.Duration // how often to query latest block
}

func (cnf *ChainTrackerConfig) validate() error {
	if cnf.blocksToSave == 0 {
		return InvalidConfigErrorBlocksToSave
	}
	if cnf.averageBlockTime == 0 {
		return InvalidConfigBlockTime
	}
	//TODO: validate address is in the right format if not empty
	return nil
}
