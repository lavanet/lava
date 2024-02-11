package monitoring

import "strconv"

type HealthRPCEndpoint struct {
	NetworkAddress string   `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address"` // HOST:PORT
	ChainID        string   `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"`                      // spec chain identifier
	ApiInterface   string   `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64   `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
	Addons         []string `yaml:"addons,omitempty" json:"addons,omitempty" mapstructure:"addons"`
}

func (endpoint *HealthRPCEndpoint) String() string {
	return endpoint.ChainID + ":" + endpoint.ApiInterface + " Network Address:" + endpoint.NetworkAddress + " Geolocation:" + strconv.FormatUint(endpoint.Geolocation, 10)
}
