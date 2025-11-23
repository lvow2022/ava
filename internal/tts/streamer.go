package tts

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/gopxl/beep"
)

var ErrStreamStopped = errors.New("stream stopped")

type Streamer struct {
	mu sync.Mutex

	format beep.Format
	buf    []byte

	EOS    bool
	Paused bool
	err    error // 用于立即停止流
}

func NewStreamer(sampleRate beep.SampleRate, channels int) *Streamer {
	return &Streamer{
		format: beep.Format{
			SampleRate:  sampleRate,
			NumChannels: channels,
			Precision:   2,
		},
		buf: make([]byte, 0, 4096),
	}
}

func (s *Streamer) AppendAudio(p []byte) {
	if len(p) == 0 {
		return
	}
	s.mu.Lock()
	s.buf = append(s.buf, p...)
	s.mu.Unlock()
}

func (s *Streamer) Stream(samples [][2]float64) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果设置了错误，立即停止流（即使 buf 中还有数据）
	if s.err != nil {
		return 0, false
	}

	if s.Paused {
		return 0, true // 暂停 → 告诉 speaker "暂时没数据"
	}

	bytesPerSample := int(s.format.NumChannels) * int(s.format.Precision)
	required := len(samples) * bytesPerSample

	if len(s.buf) == 0 {
		if s.EOS {
			return 0, false
		}
		return 0, true // 没数据但还没结束 → 等待动态 append
	}

	n := required
	if len(s.buf) < n {
		n = len(s.buf)
	}

	// 从内部 buffer 读取 n 字节
	chunk := s.buf[:n]
	s.buf = s.buf[n:] // 保留剩余区域

	// 转换到 samples
	samplesRead := n / bytesPerSample
	for i := 0; i < samplesRead; i++ {
		offset := i * bytesPerSample

		if s.format.NumChannels == 1 {
			v := pcm16ToFloat(chunk[offset:])
			samples[i][0] = v
			samples[i][1] = v
		} else {
			l := pcm16ToFloat(chunk[offset:])
			r := pcm16ToFloat(chunk[offset+2:])
			samples[i][0] = l
			samples[i][1] = r
		}
	}

	return samplesRead, true
}

func pcm16ToFloat(b []byte) float64 {
	if len(b) < 2 {
		return 0
	}
	v := int16(binary.LittleEndian.Uint16(b))
	return float64(v) / 32768.0
}

func (s *Streamer) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close 立即停止流，即使缓冲区中还有数据
func (s *Streamer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = ErrStreamStopped
}
