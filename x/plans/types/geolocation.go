package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// for convenience (calculate once only)
var (
	allGeoEnumRegionsList []Geolocation
	allGeoEnumRegions     int32
)

// initialize convenience vars at start-up
func init() {
	var geoAmount int
	for _, geoloc := range Geolocation_value {
		if geoloc != int32(Geolocation_GLS) && geoloc != int32(Geolocation_GL) {
			geoAmount += 1
			allGeoEnumRegions |= geoloc
		}
	}

	for i := 0; i < geoAmount; i++ {
		allGeoEnumRegionsList = append(allGeoEnumRegionsList, Geolocation(1<<i))
	}
}

// IsValidGeoEnum tests the validity of a given geolocation
func IsValidGeoEnum(geoloc int32) bool {
	return geoloc != int32(Geolocation_GLS) && (geoloc & ^allGeoEnumRegions) == 0
}

// IsGeoEnumSingleBit returns true if at most one bit is set
func IsGeoEnumSingleBit(geoloc int32) bool {
	return (geoloc & (geoloc - 1)) == 0
}

// ParseGeoEnum parses a string into GeoEnum bitmask.
// The string may be a number or a comma-separated geolocations codes.
func ParseGeoEnum(arg string) (geoloc int32, err error) {
	geoloc64, err := strconv.ParseUint(arg, 10, 32)
	geoloc = int32(geoloc64)
	if err == nil {
		if geoloc != int32(Geolocation_GL) {
			if !IsValidGeoEnum(geoloc) {
				return 0, fmt.Errorf("invalid geolocation value: %s", arg)
			}
		}
		return geoloc, nil
	}

	split := strings.Split(arg, ",")
	for _, s := range split {
		val, ok := Geolocation_value[s]
		if !ok || val == int32(Geolocation_GLS) {
			return 0, fmt.Errorf("invalid geolocation code: %s", s)
		}
		geoloc |= val
	}

	return geoloc, nil
}

func GetAllGeolocations() []Geolocation {
	return allGeoEnumRegionsList
}

func GetGeolocationsFromUint(geoloc int32) []Geolocation {
	geoList := []Geolocation{}
	allGeos := GetAllGeolocations()

	if geoloc == int32(Geolocation_GL) {
		return allGeoEnumRegionsList
	} else {
		for i := 0; i < len(allGeos); i++ {
			if (geoloc>>i)&1 == 1 {
				geoList = append(geoList, Geolocation(1<<i))
			}
		}
	}

	return geoList
}

// allows unmarshaling parser func
func (g Geolocation) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(Geolocation_name[int32(g)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (g *Geolocation) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*g = Geolocation(Geolocation_value[j])
	return nil
}

func PrintGeolocations() string {
	var geos []int32
	for _, geoInt := range Geolocation_value {
		geos = append(geos, geoInt)
	}

	sort.Slice(geos, func(i, j int) bool {
		return geos[i] < geos[j]
	})

	var geosStr []string
	for _, geoInt := range geos {
		if geoInt == int32(Geolocation_GLS) {
			continue
		}
		geosStr = append(geosStr, Geolocation_name[geoInt]+": 0x"+strconv.FormatInt(int64(geoInt), 16))
	}

	return strings.Join(geosStr, ", ")
}
