package ntrip_client

import (
	"math"

	"github.com/chewxy/math32"
	"github.com/go-gnss/ntrip"
)

const (
	MaximumDistanceToBaseInM = 100 * 1000
)

type StationDistance struct {
	mountpoint  string
	distanceInM float32
}

type Registry interface {
	RegisterCaster(url string, details ntrip.CasterEntry) error
	RegisterStation(url string, details ntrip.StreamEntry) error
	NearestStations(lat float32, lng float32) (distances []StationDistance, err error)
}

type CasterRegistry interface {
}

type StationsRegistry interface {
	Register(source CasterSource, mountpoint string, lat float32, lng float32)
}

const (
	earthRadiusMeters = 6371010.0
	radiansPerDegree  = math.Pi / 180.0
)

// DistanceTo computes the distance in meters between 2 points
// It uses the Haversine formula:
// https://www.movable-type.co.uk/scripts/latlong.html
func Distance32(latA float32, lngA float32, latB float32, lngB float32) float32 {
	// Convert to radians as float64
	lat1 := latA * radiansPerDegree
	lat2 := latB * radiansPerDegree
	lon1 := lngA * radiansPerDegree
	lon2 := lngB * radiansPerDegree

	sinDiffLat := math32.Sin((lat2 - lat1) / 2.0)
	sinDiffLon := math32.Sin((lon2 - lon1) / 2.0)

	a := sinDiffLat*sinDiffLat +
		math32.Cos(lat1)*math32.Cos(lat2)*
			sinDiffLon*sinDiffLon
	c := 2 * math32.Atan2(math32.Sqrt(a), math32.Sqrt(1-a))

	return earthRadiusMeters * c
}

func MetersToDegrees32(lengthInM float32, radius float32) float32 {
	angle := lengthInM / (radius * radiansPerDegree)
	return angle
}

func MetersToLatitudeAngle(lengthInM float32) float32 {
	return MetersToDegrees32(lengthInM, earthRadiusMeters)
}

func MetersToLongitudeAngleAtLatitude(lengthInM float32, latitude float32) float32 {
	return MetersToDegrees32(lengthInM, earthRadiusMeters*math32.Cos(latitude*radiansPerDegree))
}
