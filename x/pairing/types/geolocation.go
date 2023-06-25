package types

import (
	"strings"

	planstypes "github.com/lavanet/lava/x/plans/types"
)

var (
	GLS = planstypes.Geolocation_GLS
	AF  = planstypes.Geolocation_AF
	AS  = planstypes.Geolocation_AS
	AU  = planstypes.Geolocation_AU
	EU  = planstypes.Geolocation_EU
	USC = planstypes.Geolocation_USC
	USE = planstypes.Geolocation_USE
	USW = planstypes.Geolocation_USW
	GL  = planstypes.Geolocation_GL
)

const DEFAULT_GLOBAL_GEOLOCATION = 0xFFFF

func ExtractGeolocations(g uint64) ([]planstypes.Geolocation, string) {
	locations := make([]planstypes.Geolocation, 0)
	var printStr string

	if g&uint64(GL) == DEFAULT_GLOBAL_GEOLOCATION {
		locations = append(locations, GL)
		printStr += "GL"
		return locations, printStr
	}

	if g&uint64(AF) == uint64(AF) {
		locations = append(locations, AF)
		printStr += "AF,"
	}

	if g&uint64(AS) == uint64(AS) {
		locations = append(locations, AS)
		printStr += "AS,"
	}

	if g&uint64(AU) == uint64(AU) {
		locations = append(locations, AU)
		printStr += "AU,"
	}

	if g&uint64(EU) == uint64(EU) {
		locations = append(locations, EU)
		printStr += "EU,"
	}

	if g&uint64(USC) == uint64(USC) {
		locations = append(locations, USC)
		printStr += "USC,"
	}

	if g&uint64(USE) == uint64(USE) {
		locations = append(locations, USE)
		printStr += "USE,"
	}

	if g&uint64(USW) == uint64(USW) {
		locations = append(locations, USW)
		printStr += "USW,"
	}

	return locations, strings.TrimSuffix(printStr, ",")
}

func IsValidGeoEnum(s string) (uint64, bool) {
	val, ok := planstypes.Geolocation_value[s]
	if ok && val != planstypes.Geolocation_value["GLS"] && val > 0 {
		return uint64(val), true
	}

	return uint64(val), false
}

func GetCurrentGlobalGeolocation() uint64 {
	var globalGeo int32
	for k := range planstypes.Geolocation_name {
		if planstypes.Geolocation_name[k] == "GL" {
			continue
		}
		globalGeo += k
	}

	return uint64(globalGeo)
}
