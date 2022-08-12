package ntrip_client

import (
	"errors"
	"io"
	"log"
	"sync"
	"sync/atomic"

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
	Invalidate() error
}

type registrySelector struct {
	registry Registry

	significantChange float32

	// Mutable.
	invalidated   atomic.Bool
	refLat        float32
	refLng        float32
	refMountpoint string
	refRtcm       input.RtcmInput

	mu                  sync.Mutex
	newMountpoints      chan input.RtcmInput
	stop                chan struct{}
	consecutiveFailures int
}

func NewRegistrySelector(registry Registry, significantChange float32) Selector {
	return &registrySelector{
		registry: registry,

		significantChange:   significantChange,
		refLat:              0,
		refLng:              0,
		refMountpoint:       "",
		refRtcm:             nil,
		newMountpoints:      make(chan input.RtcmInput, 1),
		consecutiveFailures: 0,
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

	go func(sc <-chan struct{}, newMountpoints <-chan input.RtcmInput) {
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
				// Stop streaming
				return
			case mp := <-newMountpoints:
				// Listen to a new mountpoint
				if currentRtcm != nil {
					currentRtcm.Close()
				}
				currentRtcm = mp
				s.consecutiveFailures = 0

				if mp != nil {
					currentMountpoint, err = mp.Stream()
					if err != nil {
						log.Println("Error in streaming mountpoint", err)
					}
				} else {
					currentMountpoint = nil
				}
			case data, ok := <-currentMountpoint:
				// RTCM data received from mountpoint
				if ok {
					ch <- data
				} else {
					// The mountpoint stopped its channel. Do something about it!
					currentMountpoint = nil
					s.consecutiveFailures += 1

					// Try reconnecting
					if currentRtcm != nil {
						currentMountpoint, err = currentRtcm.Stream()
						if err != nil {
							log.Println("Error in streaming mountpoint", err)
						}
					}
				}
			}
		}
	}(s.stop, s.newMountpoints)

	return ch, nil
}

func (s *registrySelector) SetLocation(lat float32, lng float32) error {
	if !s.invalidated.Load() && math32.Abs(lat-s.refLat) <= s.significantChange &&
		math32.Abs(lng-s.refLng) <= s.significantChange {

		// No significant change
		return nil
	}

	s.invalidated.Store(false)
	s.refLat = lat
	s.refLng = lng

	// Determine the new mountpoint to use
	nearests, err := s.registry.NearestStations(lat, lng)
	if err != nil {
		return err
	}
	if len(nearests) <= 0 {
		return errors.New("no station found nearby")
	}

	nearestMP := nearests[0].mountpoint
	if nearestMP == s.refMountpoint {
		// No change
		return nil
	}

	// Update the mountpoint without breaking the stream
	log.Printf("Selected mountpoint: %s at distance %f m", nearestMP, nearests[0].distanceInM)
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

func (s *registrySelector) Invalidate() error {
	s.invalidated.Store(true)
	if s.refLat != 0 && s.refLng != 0 {
		return s.SetLocation(s.refLat, s.refLng)
	}
	return nil
}
