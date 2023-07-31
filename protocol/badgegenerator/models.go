package badgegenerator

import "github.com/lavanet/lava/x/epochstorage/types"

type ProjectConfiguration struct {
	ProjectPublicKey  string              `json:"project_public_key"`
	ProjectPrivateKey string              `json:"private_key"`
	EpochsMaxCu       int64               `json:"epochs_max_cu"`
	UpdatedEpoch      uint64              `json:"update_epoch,omitempty"`
	PairingList       *[]types.StakeEntry `json:"pairing_list,omitempty"`
}

type UserBadgeItem struct {
	AllowedCu int64
	Epoch     uint64
	Signature string
	PublicKey string
	UserId    string
}
