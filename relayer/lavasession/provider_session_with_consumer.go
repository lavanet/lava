package lavasession

type ProviderSessionWithConsumer struct {
}

//
func (cs *ProviderSessionWithConsumer) GetSession(address string, id uint64, epoch uint64, relayNum uint64) (err error) {
	return nil
}

func (cs *ProviderSessionWithConsumer) ReportConsumer() (address string, epoch uint64, err error) {
	return "", 0, nil
}

func (cs *ProviderSessionWithConsumer) GetDataReliabilitySession(address string, epoch uint64) (err error) {
	return nil
}

func (cs *ProviderSessionWithConsumer) DoneWithSession(proof string) (epoch uint64, err error) {
	return 0, nil
}
