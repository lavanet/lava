package types

import (
	"strings"

	planstypes "github.com/lavanet/lava/x/plans/types"
)

func ExtractGeolocations(g uint64) ([]planstypes.Geolocation, string) {
	locations := make([]planstypes.Geolocation, 0)
	var printStr string

	for geoName, geo := range planstypes.Geolocation_value {
		if geoName == planstypes.Geolocation_GL.String() || geoName == planstypes.Geolocation_GLS.String() {
			continue
		}
		if g&uint64(geo) == uint64(geo) {
			locations = append(locations, planstypes.Geolocation(geo))
			printStr += geoName + ","
		}
	}

	return locations, strings.TrimSuffix(printStr, ",")
}

func IsValidGeoEnum(s string) (uint64, bool) {
	val, ok := planstypes.Geolocation_value[s]
	if ok && val != planstypes.Geolocation_value[planstypes.Geolocation_GLS.String()] && val > 0 {
		return uint64(val), true
	}

	return uint64(val), false
}

func GetCurrentGlobalGeolocation() uint64 {
	var globalGeo int32
	for k := range planstypes.Geolocation_name {
		if planstypes.Geolocation_name[k] == planstypes.Geolocation_GL.String() ||
			planstypes.Geolocation_name[k] == planstypes.Geolocation_GLS.String() {
			continue
		}
		globalGeo += k
	}

	return uint64(globalGeo)
}

func CalculateGeoUintFromStrings(geos []string) uint64 {
	var uintGeo uint64
	for _, geo := range geos {
		geoVal := planstypes.Geolocation_value[geo]
		uintGeo += uint64(geoVal)
	}

	return uintGeo
}
