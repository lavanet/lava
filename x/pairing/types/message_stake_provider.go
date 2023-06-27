package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const TypeMsgStakeProvider = "stake_provider"

var _ sdk.Msg = &MsgStakeProvider{}

func NewMsgStakeProvider(creator string, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation uint64, moniker string) *MsgStakeProvider {
	return &MsgStakeProvider{
		Creator:     creator,
		ChainID:     chainID,
		Amount:      amount,
		Endpoints:   endpoints,
		Geolocation: geolocation,
		Moniker:     moniker,
	}
}

func (msg *MsgStakeProvider) Route() string {
	return RouterKey
}

func (msg *MsgStakeProvider) Type() string {
	return TypeMsgStakeProvider
}

func (msg *MsgStakeProvider) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgStakeProvider) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeProvider) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Moniker == "" {
		return sdkerrors.Wrapf(MonikerEmptyError, "invalid moniker (%s)", msg.Moniker)
	}

	if len(msg.Moniker) > MAX_LEN_MONIKER {
		return sdkerrors.Wrapf(MonikerTooLongError, "invalid moniker (%s)", msg.Moniker)
	}

	return nil
}

// // verify that the geolocation arg is a union of the endpoints' geolocation
// func ValidateGeoFields(endp []epochstoragetypes.Endpoint, geo uint64) error {
// 	geoSeen := map[string]struct{}{}
// 	var endpointsGeoStr string
// 	for _, endp := range endp {
// 		geoStr := planstypes.Geolocation_name[int32(endp.Geolocation)]
// 		_, ok := geoSeen[geoStr]
// 		if !ok {
// 			geoSeen[geoStr] = struct{}{}
// 			endpointsGeoStr += geoStr + ","
// 		}
// 	}

// 	endpointsGeoStr = strings.TrimSuffix(endpointsGeoStr, ",")
// 	geoEnums, geoStr := ExtractGeolocations(geo)
// 	if len(geoEnums) != len(geoSeen) {
// 		return sdkerrors.Wrapf(GeolocationNotMatchWithEndpointsError,
// 			"invalid geolocation (endpoints combined geolocation: {%s}, provider geolocation: {%s})", endpointsGeoStr, geoStr)
// 	}
// 	for _, geoE := range geoEnums {
// 		_, ok := geoSeen[geoE.String()]
// 		if !ok {
// 			if geoE != planstypes.Geolocation_GL {
// 				return sdkerrors.Wrapf(GeolocationNotMatchWithEndpointsError,
// 					"invalid geolocation (endpoints combined geolocation: {%s}, provider geolocation: {%s})", endpointsGeoStr, geoStr)
// 			} else if len(geoSeen) != len(planstypes.Geolocation_name)-2 {
// 				// handle the global case (should see all geos minus 2 global geos)
// 				return sdkerrors.Wrapf(GeolocationNotMatchWithEndpointsError,
// 					"invalid global geolocation (endpoints combined geolocation: {%s}, provider geolocation: {%s})", endpointsGeoStr, geoStr)
// 			}
// 		}
// 	}

// 	return nil
// }
