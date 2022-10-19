package lavasession

type ProviderSessionManager struct {
}

//
func (ps *ProviderSessionManager) GetSession(address string, id uint64, epoch uint64, relayNum uint64) (err error) {
	return nil
}

func (ps *ProviderSessionManager) ReportConsumer() (address string, epoch uint64, err error) {
	return "", 0, nil
}

func (ps *ProviderSessionManager) GetDataReliabilitySession(address string, epoch uint64) (err error) {
	return nil
}

func (ps *ProviderSessionManager) DoneWithSession(proof string) (epoch uint64, err error) {
	return 0, nil
}
