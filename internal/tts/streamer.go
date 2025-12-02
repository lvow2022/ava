package tts

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/gopxl/beep"
)

var (
	ErrStreamStopped = errors.New("stream stopped")
	ErrEndOfStream   = errors.New("end of stream")
)

type Streamer struct {
	mu sync.Mutex

	format beep.Format
	buf    []byte

	err error
	// 播放进度跟踪
	bytesPlayed   int64     // 已播放的字节数
	startTime     time.Time // 开始播放的时间
	totalDuration float64   // 总时长（秒），从 TTS 返回的时间信息计算

	// 时间信息（用于获取已播放文本）
	timings []SentenceTiming
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

	// 如果被停止，立即返回
	if s.err == ErrStreamStopped {
		return 0, false
	}

	bytesPerSample := int(s.format.NumChannels) * int(s.format.Precision)
	required := len(samples) * bytesPerSample

	if len(s.buf) == 0 {
		// 如果 buffer 为空且已经到达流结束，返回 false
		if s.err == ErrEndOfStream {
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

	// 更新已播放字节数
	if s.bytesPlayed == 0 && s.startTime.IsZero() {
		s.startTime = time.Now()
	}
	s.bytesPlayed += int64(n)

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

// SetError 设置流的错误状态，只有在当前没有错误时才设置（避免覆盖已有错误）
func (s *Streamer) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

// Close 立即停止流，即使缓冲区中还有数据（流将不再可用）
func (s *Streamer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = ErrStreamStopped
}

// SetTotalDuration 设置总时长（秒）
func (s *Streamer) SetTotalDuration(duration float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalDuration = duration
}

// GetProgress 获取播放进度
func (s *Streamer) GetProgress() (currentTime float64, totalTime float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalTime = s.totalDuration

	if s.startTime.IsZero() {
		return 0, totalTime
	}

	// 根据已播放的字节数计算当前时间
	bytesPerSecond := float64(s.format.SampleRate) * float64(s.format.NumChannels) * float64(s.format.Precision)
	if bytesPerSecond > 0 {
		currentTime = float64(s.bytesPlayed) / bytesPerSecond
	}

	// 如果总时长已知，确保当前时间不超过总时长
	if totalTime > 0 && currentTime > totalTime {
		currentTime = totalTime
	}

	return currentTime, totalTime
}

// ResetProgress 重置播放进度（用于新的播放任务）
func (s *Streamer) ResetProgress() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bytesPlayed = 0
	s.startTime = time.Time{}
	s.totalDuration = 0
	s.timings = s.timings[:0] // 清空时间信息
}

// SetTimings 设置时间信息（用于获取已播放文本）
func (s *Streamer) SetTimings(timings []SentenceTiming) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timings = make([]SentenceTiming, len(timings))
	copy(s.timings, timings)
}

// GetPlayedText 根据当前播放时间获取已播放的文本（所有已播放完成的词拼接）
func (s *Streamer) GetPlayedText(currentTime float64) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var playedWords []string

	for _, sentence := range s.timings {
		for _, word := range sentence.Words {
			// 如果词的结束时间 <= 当前时间，说明这个词已经播放完成
			if word.EndTime <= currentTime {
				playedWords = append(playedWords, word.Word)
			} else {
				// 如果词的开始时间 > 当前时间，说明还没播放到，停止遍历
				// 如果当前时间在词的时间范围内，也不加入已播放列表（因为还没播放完成）
				break
			}
		}
		// 如果当前句子中还有未播放的词，停止遍历后续句子
		if len(playedWords) > 0 {
			// 检查当前句子是否还有未播放的词
			allPlayed := true
			for _, word := range sentence.Words {
				if word.EndTime > currentTime {
					allPlayed = false
					break
				}
			}
			if !allPlayed {
				break
			}
		}
	}

	// 拼接已播放的词（词与词之间不需要空格，因为词本身可能包含标点）
	result := ""
	for _, word := range playedWords {
		result += word
	}

	return result
}
