package tts

import (
	"sync"

	"github.com/gopxl/beep"
)

type StreamQueue struct {
	mu      sync.Mutex
	current beep.Streamer
	queue   []beep.Streamer
}

// GetCurrentStreamer 获取当前正在播放的 Streamer（如果是 *Streamer 类型）
func (q *StreamQueue) GetCurrentStreamer() *Streamer {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.current != nil {
		if s, ok := q.current.(*Streamer); ok {
			return s
		}
	}
	return nil
}

// StopCurrent 停止当前正在播放的 stream
func (q *StreamQueue) StopCurrent() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.current != nil {
		if s, ok := q.current.(*Streamer); ok {
			s.Close()
		}
	}
}

func NewStreamQueue() *StreamQueue {
	return &StreamQueue{}
}

func (q *StreamQueue) Push(s beep.Streamer) {
	q.mu.Lock()
	q.queue = append(q.queue, s)
	q.mu.Unlock()
}

func (q *StreamQueue) Stream(samples [][2]float64) (n int, ok bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		if q.current == nil {
			if len(q.queue) == 0 {
				return 0, true // 暂时无数据，不停止播放
			}
			q.current = q.queue[0]
			q.queue = q.queue[1:]
		}

		n, ok = q.current.Stream(samples)
		if !ok {
			q.current = nil
			continue
		}
		return n, ok
	}
}

func (q *StreamQueue) Err() error { return nil }
