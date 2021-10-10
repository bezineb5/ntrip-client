package ntrip_client

import (
	"errors"
	"io"
	"log"
	"sync"

	"github.com/bezineb5/ntrip-client/input"
	"github.com/chewxy/math32"
)

const (
	SIGNIFICANT_CHANGE = 0.01
)

type Selector interface {
	io.Closer
	Stream() (<-chan []byte, error)
	SetLocation(lat float32, lng float32) error
}

type registrySelector struct {
	registry Registry

	significantChange float32

	// Mutable.
	refLat        float32
	refLng        float32
	refMountpoint string
	refRtcm       input.RtcmInput

	mu             sync.Mutex
	newMountpoints chan input.RtcmInput
	stop           chan struct{}
}

func NewRegistrySelector(registry Registry, significantChange float32) Selector {
	return &registrySelector{
		registry: registry,

		significantChange: significantChange,
		refLat:            0,
		refLng:            0,
		refMountpoint:     "",
		refRtcm:           nil,
		newMountpoints:    make(chan input.RtcmInput, 1),
	}
}

func (s *registrySelector) Stream() (<-chan []byte, error) {
	// We need to lock if there are multiple Stream, Close or SetLocation
	// calls simultaneously.
	s.mu.Lock()
	defer s.mu.Unlock()

	// First release the current continuous reading if there is one
	if s.stop != nil {
		s.stop <- struct{}{}
		s.stop = nil
	}
	s.stop = make(chan struct{})
	ch := make(chan []byte, 4)

	go func(sc <-chan struct{}) {
		defer close(ch)

		currentRtcm := s.refRtcm
		var currentMountpoint <-chan []byte = nil
		var err error

		if currentRtcm != nil {
			if currentMountpoint, err = s.refRtcm.Stream(); err != nil {
				log.Println("Error in streaming mountpoint", err)
			}
		}

		for {
			select {
			case <-sc:
				return
			case mp := <-s.newMountpoints:
				if currentRtcm != nil {
					currentRtcm.Close()
				}
				currentRtcm = mp

				if mp != nil {
					currentMountpoint, err = mp.Stream()
					if err != nil {
						log.Println("Error in streaming mountpoint", err)
					}
				} else {
					currentMountpoint = nil
				}
			case data := <-currentMountpoint:
				ch <- data
			}
		}
	}(s.stop)

	return ch, nil
}

func (s *registrySelector) SetLocation(lat float32, lng float32) error {
	if math32.Abs(lat-s.refLat) <= s.significantChange &&
		math32.Abs(lng-s.refLng) <= s.significantChange {

		// No significant change
		return nil
	}

	s.refLat = lat
	s.refLng = lng

	// Determine the new mountpoint to use
	nearests, err := s.registry.NearestStations(lat, lng)
	if err != nil {
		return err
	}
	if len(nearests) <= 0 {
		return errors.New("No station found nearby")
	}

	nearestMP := nearests[0].mountpoint
	if nearestMP == s.refMountpoint {
		// No change
		return nil
	}

	// Update the mountpoint without breaking the stream
	client := input.NewNtripV2MountPointClient(nearestMP)
	s.refMountpoint = nearestMP

	s.newMountpoints <- client

	return nil
}

func (s *registrySelector) Close() error {
	// We need to lock if there are multiple
	// calls simultaneously.
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stop != nil {
		s.stop <- struct{}{}
		s.stop = nil
	}
	return nil
}
