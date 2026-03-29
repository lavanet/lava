package types

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

// GeolocationsAll returns a slice of all single-region geolocation constants
// (excluding the global wildcard GL).
func GeolocationsAll() []Geolocation {
	return []Geolocation{
		Geolocation_USC,
		Geolocation_EU,
		Geolocation_USE,
		Geolocation_USW,
		Geolocation_AF,
		Geolocation_AS,
		Geolocation_AU,
	}
}

// Contains reports whether geo has all the bits set that are present in other.
func (geo Geolocation) Contains(other Geolocation) bool {
	if geo == Geolocation_GL {
		return true
	}
	return geo&other != 0
}
