package lavasession

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
)

type NetworkAddressData struct {
	Address    string `yaml:"address,omitempty" json:"address,omitempty" mapstructure:"address,omitempty"` // HOST:PORT
	KeyPem     string `yaml:"key-pem,omitempty" json:"key-pem,omitempty" mapstructure:"key-pem"`
	CertPem    string `yaml:"cert-pem,omitempty" json:"cert-pem,omitempty" mapstructure:"cert-pem"`
	DisableTLS bool   `yaml:"disable-tls,omitempty" json:"disable-tls,omitempty" mapstructure:"disable-tls"`
}

type RPCProviderEndpoint struct {
	NetworkAddress NetworkAddressData `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address,omitempty"`
	ChainID        string             `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"` // spec chain identifier
	ApiInterface   string             `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64             `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
	NodeUrls       []common.NodeUrl   `yaml:"node-urls,omitempty" json:"node-urls,omitempty" mapstructure:"node-urls"`
	Name           string             `yaml:"provider-name,omitempty" json:"provider-name,omitempty" mapstructure:"provider-name"`
}

// RPCStaticProviderEndpoint extends RPCProviderEndpoint with additional fields for static providers
// This allows us to add functionality without modifying the original protobuf-derived type
type RPCStaticProviderEndpoint struct {
	NetworkAddress NetworkAddressData `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address,omitempty"`
	ChainID        string             `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"` // spec chain identifier
	ApiInterface   string             `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64             `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
	NodeUrls       []common.NodeUrl   `yaml:"node-urls,omitempty" json:"node-urls,omitempty" mapstructure:"node-urls"`
	Name           string             `yaml:"name,omitempty" json:"name,omitempty" mapstructure:"name,omitempty"`
	// Stake is an optional stake amount (in ulava) used for provider selection scoring in static-provider tests.
	// If omitted, it is treated as 0 so the weight calculator can apply the legacy
	// "static provider boost" behavior (instead of using an explicit stake value).
	Stake int64 `yaml:"stake,omitempty" json:"stake,omitempty" mapstructure:"stake,omitempty"`
}

// ToBase returns the base RPCProviderEndpoint (for compatibility with existing code)
func (ext *RPCStaticProviderEndpoint) ToBase() *RPCProviderEndpoint {
	return &RPCProviderEndpoint{
		NetworkAddress: ext.NetworkAddress,
		ChainID:        ext.ChainID,
		ApiInterface:   ext.ApiInterface,
		Geolocation:    ext.Geolocation,
		NodeUrls:       ext.NodeUrls,
	}
}

// Validate checks if the RPCStaticProviderEndpoint has valid configuration
func (ext *RPCStaticProviderEndpoint) Validate() error {
	if ext.Name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if len(ext.NodeUrls) == 0 {
		return fmt.Errorf("provider must have at least one node URL")
	}
	if ext.ChainID == "" {
		return fmt.Errorf("provider chain-id cannot be empty")
	}
	if ext.ApiInterface == "" {
		return fmt.Errorf("provider api-interface cannot be empty")
	}
	return nil
}

func (endpoint *RPCProviderEndpoint) UrlsString() string {
	st_urls := make([]string, len(endpoint.NodeUrls))
	for idx, url := range endpoint.NodeUrls {
		st_urls[idx] = url.UrlStr()
	}
	return strings.Join(st_urls, ", ")
}

func (endpoint *RPCProviderEndpoint) AddonsString() string {
	st_urls := make([]string, len(endpoint.NodeUrls))
	for idx, url := range endpoint.NodeUrls {
		st_urls[idx] = strings.Join(url.Addons, ",")
	}
	return strings.Join(st_urls, "; ")
}

func (endpoint *RPCProviderEndpoint) String() string {
	return endpoint.ChainID + ":" + endpoint.ApiInterface + " Network Address:" + endpoint.NetworkAddress.Address + " Node:" + endpoint.UrlsString() + " Geolocation:" + strconv.FormatUint(endpoint.Geolocation, 10) + " Addons:" + endpoint.AddonsString()
}

func (endpoint *RPCProviderEndpoint) Validate() error {
	if len(endpoint.NodeUrls) == 0 {
		return utils.LavaFormatError("Empty URL list for endpoint", nil, utils.Attribute{Key: "endpoint", Value: endpoint.String()})
	}
	for _, url := range endpoint.NodeUrls {
		err := common.ValidateEndpoint(url.Url, endpoint.ApiInterface)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rpcpe *RPCProviderEndpoint) Key() string {
	return rpcpe.ChainID + rpcpe.ApiInterface
}
