package lavasession

type ProviderSessionsWithConsumer struct {
}

//
func (cs *ProviderSessionsWithConsumer) GetSession(address string, id uint64, epoch uint64, relayNum uint64) (err error) {
	return nil
}

func (cs *ProviderSessionsWithConsumer) ReportConsumer() (address string, epoch uint64, err error) {
	return "", 0, nil
}

func (cs *ProviderSessionsWithConsumer) GetDataReliabilitySession(address string, epoch uint64) (err error) {
	return nil
}

func (cs *ProviderSessionsWithConsumer) DoneWithSession(proof string) (epoch uint64, err error) {
	return 0, nil
}
