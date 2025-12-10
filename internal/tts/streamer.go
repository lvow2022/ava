package tts

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/sirupsen/logrus"
)

var (
	ErrStreamStopped = errors.New("stream stopped")
	ErrEndOfStream   = errors.New("end of stream")
)

type Streamer struct {
	format beep.Format

	// 使用 bytes.Buffer 作为缓冲区
	buf *bytes.Buffer
	mu  sync.RWMutex // 保护 buffer 和状态

	// Context 用于取消（消费者调用 Cancel() 时取消）
	ctx    context.Context
	cancel context.CancelFunc

	// 状态管理
	err           error
	eos           bool
	bytesPlayed   int64     // 已播放的字节数
	startTime     time.Time // 开始播放的时间
	totalDuration float64   // 总时长（秒），从 TTS 返回的时间信息计算

	// 时间信息（用于获取已播放文本）
	timings []SentenceTiming
}

func NewStreamer(sampleRate beep.SampleRate, channels int) *Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Streamer{
		format: beep.Format{
			SampleRate:  sampleRate,
			NumChannels: channels,
			Precision:   2,
		},
		buf:    bytes.NewBuffer(make([]byte, 0, 8192)), // 初始容量 8KB
		ctx:    ctx,
		cancel: cancel,
	}

	return s
}

func (s *Streamer) AppendAudio(p []byte) {
	// 检查消费者是否已取消（非阻塞检查）
	select {
	case <-s.ctx.Done():
		return // 消费者已取消，不再写入
	default:
	}

	if len(p) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 再次检查 context（双重检查，避免在加锁期间被取消）
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	// 检查错误状态
	if s.err != nil {
		return
	}

	// 写入数据到 buffer
	_, err := s.buf.Write(p)
	if err != nil {
		s.err = err
		logrus.Errorf("streamer: failed to write to buffer: %v", err)
		return
	}
}

func (s *Streamer) Stream(samples [][2]float64) (int, bool) {
	// 检查是否已取消（非阻塞检查）
	select {
	case <-s.ctx.Done():
		s.mu.Lock()
		if s.err == nil {
			s.err = s.ctx.Err() // context.Canceled
		}
		s.mu.Unlock()
		return 0, false
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查错误状态
	if s.err == ErrStreamStopped {
		return 0, false
	}

	bytesPerSample := int(s.format.NumChannels) * int(s.format.Precision)
	required := len(samples) * bytesPerSample

	// 检查 buffer 是否有数据（非阻塞）
	if s.buf.Len() == 0 {
		if s.eos {
			return 0, false
		}
		if s.err != nil {
			return 0, false
		}
		// 没有数据但流还没结束，返回 (0, true) 让 beep 继续轮询
		// 注意：beep 的 Stream 接口期望非阻塞，所以不能在这里 Wait()
		return 0, true
	}

	// 读取数据
	readSize := min(required, s.buf.Len())
	if readSize == 0 {
		if s.eos {
			return 0, false
		}
		return 0, true
	}

	// 从 buffer 读取数据
	chunk := make([]byte, readSize)
	n, err := s.buf.Read(chunk)
	if err != nil && err != io.EOF {
		s.err = err
		logrus.Errorf("streamer: failed to read from buffer: %v", err)
		return 0, false
	}

	if n == 0 {
		if s.eos {
			return 0, false
		}
		return 0, true
	}

	// 更新进度
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

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Streamer) Err() error {
	// 优先返回 context 错误
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

// Close 实现 io.Closer 接口，关闭流并设置 EOF 错误
// 调用 Close() 后，Stream() 方法将返回 (0, false) 表示流结束
func (s *Streamer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eos = true
	if s.err == nil {
		s.err = io.EOF
	}
	return nil
}

// Cancel 取消流，由消费者调用，通知生产者停止写入
// 调用 Cancel() 后，Stream() 和 AppendAudio() 都会立即停止
func (s *Streamer) Cancel() {
	s.cancel() // 取消 context，通知生产者
	s.mu.Lock()
	if s.err == nil {
		s.err = ErrStreamStopped
	}
	s.mu.Unlock()
}

// SetTotalDuration 设置总时长（秒）
func (s *Streamer) SetTotalDuration(duration float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalDuration = duration
}

// GetProgress 获取播放进度
func (s *Streamer) GetProgress() (currentTime float64, totalTime float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	// 重置 buffer（保留容量）
	s.buf.Reset()
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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
