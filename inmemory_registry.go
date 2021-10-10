package ntrip_client

import (
	"fmt"
	"sort"

	"github.com/chewxy/math32"
	"github.com/go-gnss/ntrip"
)

type inMemoryRegistry struct {
	casters  map[string]ntrip.CasterEntry
	stations map[string]ntrip.StreamEntry

	maxLatitudeDiff float32
}

func NewInMemoryRegistry() Registry {
	return &inMemoryRegistry{
		casters:  make(map[string]ntrip.CasterEntry),
		stations: make(map[string]ntrip.StreamEntry),

		maxLatitudeDiff: MetersToLatitudeAngle(MaximumDistanceToBaseInM),
	}
}

func (r *inMemoryRegistry) RegisterCaster(url string, details ntrip.CasterEntry) error {
	if _, ok := r.casters[url]; ok {
		// The url is already registered
	} else {
		r.casters[url] = details
	}

	return nil
}

func (r *inMemoryRegistry) RegisterStation(url string, details ntrip.StreamEntry) error {
	if _, ok := r.stations[url]; ok {
		// The url is already registered
	} else {
		r.stations[url] = details
		fmt.Println("Registered: ", url)
	}

	return nil
}

func (r *inMemoryRegistry) NearestStations(lat float32, lng float32) ([]StationDistance, error) {
	distances := make([]StationDistance, 0, 8)
	maxLongitudeDiff := MetersToLongitudeAngleAtLatitude(MaximumDistanceToBaseInM, lat)

	for k, v := range r.stations {
		if math32.Abs(v.Latitude-lat) <= r.maxLatitudeDiff && math32.Abs(v.Longitude-lng) <= maxLongitudeDiff {
			distance := Distance32(lat, lng, v.Latitude, v.Longitude)
			distances = append(distances, StationDistance{
				mountpoint:  k,
				distanceInM: distance,
			})
		}
	}

	sort.Slice(distances, func(i, j int) bool { return distances[i].distanceInM <= distances[j].distanceInM })

	return distances, nil
}
