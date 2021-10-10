package input

import (
	"io"

	"github.com/go-gnss/ntrip"
)

type SourceTableInput interface {
	Url() string
	SourceTable() (ntrip.Sourcetable, error)
}

type RtcmInput interface {
	io.Closer
	Stream() (<-chan []byte, error)
}
