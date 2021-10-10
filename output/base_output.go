package output

import "io"

type RtcmOutput interface {
	io.Closer
	Stream(<-chan []byte) error
}
