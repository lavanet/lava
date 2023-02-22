package mock

type MockEpochUpdatable struct {
	epoch uint64
}

func (m MockEpochUpdatable) UpdateEpoch(epoch uint64) {
	m.epoch = epoch
}
