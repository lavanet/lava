package plans

import "fmt"

// Geolocation is a bitmask identifying one or more geographic regions.
// Values match the on-chain protobuf enum lavanet.lava.plans.Geolocation.
type Geolocation int32

const (
	// Geolocation_GL selects all geolocations (global).
	Geolocation_GL Geolocation = 0
	// Geolocation_USC — US Center.
	Geolocation_USC Geolocation = 1
	// Geolocation_EU — Europe.
	Geolocation_EU Geolocation = 2
	// Geolocation_USE — US East.
	Geolocation_USE Geolocation = 4
	// Geolocation_USW — US West.
	Geolocation_USW Geolocation = 8
	// Geolocation_AF — Africa.
	Geolocation_AF Geolocation = 16
	// Geolocation_AS — Asia.
	Geolocation_AS Geolocation = 32
	// Geolocation_AU — Australia / Oceania.
	Geolocation_AU Geolocation = 64
)

// String returns a human-readable name for a geolocation constant.
func (geo Geolocation) String() string {
	switch geo {
	case Geolocation_GL:
		return "GL"
	case Geolocation_USC:
		return "USC"
	case Geolocation_EU:
		return "EU"
	case Geolocation_USE:
		return "USE"
	case Geolocation_USW:
		return "USW"
	case Geolocation_AF:
		return "AF"
	case Geolocation_AS:
		return "AS"
	case Geolocation_AU:
		return "AU"
	default:
		return fmt.Sprintf("Geolocation(%d)", int32(geo))
	}
}
