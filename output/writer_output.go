package output

import (
	"io"
	"sync"
)

type writerOutput struct {
	writer io.Writer

	// Mutable.
	mu   sync.Mutex
	stop chan struct{}
}

func NewWriterOutput(writer io.Writer) RtcmOutput {
	return &writerOutput{
		writer: writer,
	}
}

func (o *writerOutput) Stream(input <-chan []byte) error {
	// We need to lock if there are multiple Stream
	// calls simultaneously.
	o.mu.Lock()
	defer o.mu.Unlock()

	o.stop = make(chan struct{})

	go func(s <-chan struct{}) {
		for {
			select {
			case <-s:
				return
			case data := <-input:
				o.writer.Write(data)
			}
		}
	}(o.stop)

	return nil
}

func (o *writerOutput) Close() error {
	if o.writer != nil {
		oldWriter := o.writer
		o.writer = nil

		if writerCloser, ok := oldWriter.(io.WriteCloser); ok {
			return writerCloser.Close()
		}
	}

	return nil
}
