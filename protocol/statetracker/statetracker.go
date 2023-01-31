package statetracker

type EpochUpdatable interface {
	UpdateEpoch(epoch uint64)
}
