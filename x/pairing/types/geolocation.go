package types

import (
	"fmt"
	"strconv"
	"strings"

	commontypes "github.com/lavanet/lava/common/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/spf13/cast"
)

const (
	DEFAULT_GLOBAL_GEOLOCATION = 0xFFFF
	GLOBAL_GEO                 = "GL"
	GLOBAL_STRICT_GEO          = "GLS"
)

func ExtractGeolocations(g uint64) ([]planstypes.Geolocation, string) {
	locations := make([]planstypes.Geolocation, 0)
	var printStr string

	for geoName, geo := range planstypes.Geolocation_value {
		if geoName == GLOBAL_GEO || geoName == GLOBAL_STRICT_GEO {
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
	if ok && val != planstypes.Geolocation_value[GLOBAL_STRICT_GEO] && val > 0 {
		return uint64(val), true
	}

	return uint64(val), false
}

func GetCurrentGlobalGeolocation() uint64 {
	var globalGeo int32
	for k := range planstypes.Geolocation_name {
		if planstypes.Geolocation_name[k] == GLOBAL_GEO || planstypes.Geolocation_name[k] == GLOBAL_STRICT_GEO {
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

func HandleEndpointsAndGeolocationArgs(endpArg []string, geoArg string) (endp []epochstoragetypes.Endpoint, geo uint64, err error) {
	for _, endpointStr := range endpArg {
		splitted := strings.Split(endpointStr, ",")
		if len(splitted) != 3 {
			return nil, 0, fmt.Errorf("invalid argument format in endpoints, must be: HOST:PORT,useType,geolocation HOST:PORT,useType,geolocation, received: %s", endpointStr)
		}

		// geolocation of an endpoint can be uint or a string representing a single geo region
		var geoloc uint64
		geoloc, valid := IsValidGeoEnum(splitted[2])
		if !valid {
			geoloc, err = strconv.ParseUint(splitted[2], 10, 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid argument format in endpoints, geolocation must be a number or valid geolocation string")
			}
		}

		// if the user specified global ("GL"), append the endpoint in all possible geolocations
		if geoloc == uint64(planstypes.Geolocation_value[GLOBAL_GEO]) {
			for geoName, geoVal := range planstypes.Geolocation_value {
				if geoName == GLOBAL_GEO || geoName == GLOBAL_STRICT_GEO {
					continue
				}
				endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], UseType: splitted[1], Geolocation: uint64(geoVal)}
				endp = append(endp, endpoint)
			}
		} else {
			// if it's not global, verify that the endpoint's geolocation represents a single geo region
			geoRegions, _ := ExtractGeolocations(geoloc)
			if len(geoRegions) != 1 {
				return nil, 0, fmt.Errorf("invalid geolocation for endpoint, must represent one region")
			}
			endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], UseType: splitted[1], Geolocation: geoloc}
			endp = append(endp, endpoint)
		}
	}

	// handle the case of string geolocation (example: "EU,AF,AS")
	splitted := strings.Split(geoArg, ",")
	strEnums := commontypes.RemoveDuplicatesFromSlice(splitted)
	for _, s := range strEnums {
		g, valid := IsValidGeoEnum(s)
		if valid {
			// if one of the endpoints is global, assign global value and break
			if g == uint64(planstypes.Geolocation_GL) {
				geo = GetCurrentGlobalGeolocation()
				break
			}
			// for non-global geolocations constructs the final geolocation uint value (addition works because we
			// removed duplicates and the geo regions represent a bitmap)
			geo += g
		}
	}

	// geolocation is not a list of enums, try to parse it as an uint
	if geo == 0 {
		geo, err = cast.ToUint64E(geoArg)
		if err != nil {
			return nil, 0, err
		}
	}
	return endp, geo, nil
}
