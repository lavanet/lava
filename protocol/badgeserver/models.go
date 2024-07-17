package badgeserver

import "github.com/lavanet/lava/v2/x/pairing/types"

type GelocationToProjectsConfiguration map[string]map[string]*ProjectConfiguration

type ProjectConfiguration struct {
	EpochsMaxCu  int64                                     `yaml:"epochs-max-cu,omitempty" json:"epochs-max-cu,omitempty" mapstructure:"epochs-max-cu,omitempty"`
	UpdatedEpoch map[string]uint64                         `yaml:"update-epoch,omitempty" json:"update-epoch,omitempty" mapstructure:"update-epoch,omitempty"`
	PairingList  map[string]*types.QueryGetPairingResponse `yaml:"pairing-list,omitempty" json:"pairing-list,omitempty" mapstructure:"pairing-list,omitempty"`
}

type UserBadgeItem struct {
	AllowedCu int64
	Epoch     uint64
	Signature string
	PublicKey string
	UserId    string
}

type IpData struct {
	FromIp      int64
	ToIP        int64
	CountryCode string
	Geolocation int
}
