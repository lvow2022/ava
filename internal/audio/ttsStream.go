package audio

import (
	"errors"
	"sync"
)

var (
	ErrStreamClosed = errors.New("stream closed")
	ErrEOF          = errors.New("eof")
)

type TTSStream struct {
	queue chan Frame
	done  chan struct{}

	closeOnce sync.Once
}

func NewTTSStream(size int) *TTSStream {
	return &TTSStream{
		queue: make(chan Frame, size),
		done:  make(chan struct{}),
	}
}

func (s *TTSStream) Write(frame Frame) error {
	select {
	case <-s.done:
		return ErrStreamClosed
	case s.queue <- frame:
		return nil
	}
}

func (s *TTSStream) Read() (*Frame, error) {
	select {
	case <-s.done:
		return nil, ErrStreamClosed
	case frame := <-s.queue:
		return &frame, nil
	}
}

// reciever should call this to close the stream
func (s *TTSStream) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	return nil
}

func (s *TTSStream) Done() <-chan struct{} {
	return s.done
}
