package ntrip_client

import (
	"net/url"

	"github.com/bezineb5/ntrip-client/input"
	"github.com/go-gnss/ntrip"
)

type CasterSource interface {
	RegisterMountPoints(Registry) error
}

type ntripSource struct {
	input       input.SourceTableInput
	sourceTable *ntrip.Sourcetable
}

func NewSource(input input.SourceTableInput) CasterSource {
	return &ntripSource{
		input: input,
	}
}

func (s *ntripSource) RegisterMountPoints(registry Registry) error {
	if s.sourceTable == nil {
		src, err := s.input.SourceTable()
		if err != nil {
			return err
		}

		s.sourceTable = &src
	}

	// Mount points
	for _, mountpoint := range s.sourceTable.Mounts {
		/*url := url.URL{}
		url.Host = caster.Host
		if caster.Port != 0 {
			url.Host = url.Host + ":" + strconv.Itoa(caster.Port)
		}

		mountpoint.String()*/

		mpUrl, err := buildMountpointUrl(s.input.Url(), mountpoint.Name)
		if err != nil {
			return err
		}

		registry.RegisterStation(
			mpUrl,
			mountpoint)
	}

	// Other referenced casters
	/*for _, casters := range s.sourceTable.Casters {

	}*/

	return nil
}

func buildMountpointUrl(casterUrl string, mountpoint string) (string, error) {
	// Build a URL with the mount point
	u, err := url.Parse(casterUrl)
	if err != nil {
		return casterUrl, err
	}
	u.Path = url.PathEscape(mountpoint)

	return u.String(), nil
}

/*func (s *ntripSource) RegisterCasters(registry Registry) error {
	if s.sourceTable == nil {
		src, err := s.input.SourceTable()
		if err != nil {
			return err
		}

		s.sourceTable = &src
	}

	for _, caster := range s.sourceTable.Casters {
		url := url.URL{}
		url.Host = caster.Host
		if caster.Port != 0 {
			url.Host = url.Host + ":" + strconv.Itoa(caster.Port)
		}

		registry.Register(
			s,
			url.String(),
			caster.Identifier,
			caster.Latitude,
			caster.Longitude)
	}

	return nil
}*/
