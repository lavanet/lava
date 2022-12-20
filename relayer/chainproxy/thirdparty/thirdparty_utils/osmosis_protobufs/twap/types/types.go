package types

import paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

type AmmInterface struct{}

func ParamKeyTable() paramtypes.Subspace {
	return paramtypes.Subspace{}
}
