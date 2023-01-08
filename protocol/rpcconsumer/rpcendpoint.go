package rpcconsumer

type RPCEndpoint struct {
	Address      string // IP:PORT
	ChainID      string // spec chain identifier
	ApiInterface string
	Geolocation  uint64
}

func (rpce *RPCEndpoint) New(address string, chainID string, apiInterface string, geolocation uint64) *RPCEndpoint {
	// TODO: validate correct url address
	rpce.Address = address
	rpce.ChainID = chainID
	rpce.ApiInterface = apiInterface
	rpce.Geolocation = geolocation
	return rpce
}

func (rpce *RPCEndpoint) Key() string {
	return rpce.ChainID + rpce.ApiInterface
}
