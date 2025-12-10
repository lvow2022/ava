package tts

import "sync"

// GetCurrentWordFromTimings 根据时间戳列表和当前时间计算正在播放的词
// 这是一个包级别的辅助函数，可以被外部使用
// 如果当前时间不在任何词的时间范围内，返回最近播放的词或下一个要播放的词
func GetCurrentWordFromTimings(timings []SentenceTiming, currentTime float64) *WordTiming {
	var lastWord *WordTiming
	var nextWord *WordTiming
	var foundNext bool

	for _, sentence := range timings {
		for _, word := range sentence.Words {
			// 如果当前时间在词的时间范围内，直接返回
			if currentTime >= word.StartTime && currentTime <= word.EndTime {
				return &word
			}

			// 记录最近播放的词（结束时间 <= 当前时间）
			if word.EndTime <= currentTime {
				lastWord = &word
			}

			// 记录下一个要播放的词（开始时间 > 当前时间，且还没找到下一个）
			if !foundNext && word.StartTime > currentTime {
				nextWord = &word
				foundNext = true
			}
		}
	}

	// 如果当前时间在词与词之间的间隔中，优先返回最近播放的词
	// 如果最近播放的词距离当前时间较远（>0.5秒），则返回下一个要播放的词
	if lastWord != nil {
		timeSinceLastWord := currentTime - lastWord.EndTime
		if timeSinceLastWord <= 0.5 {
			return lastWord
		}
	}

	// 返回下一个要播放的词
	if nextWord != nil {
		return nextWord
	}

	// 如果都没有，返回最近播放的词
	return lastWord
}

type EngineMetadata struct {
	Name    string
	Version string
	Vendor  string
	// ... anything else
}

type EngineMetrics struct {
	FramesGenerated int64
	TimeCostMs      int64
	// ... anything else
}

type BaseEngine struct {
	metadata *EngineMetadata
	metrics  *EngineMetrics

	mu sync.RWMutex

	// 时间戳信息
	timings []SentenceTiming
}

func NewBaseEngine(meta *EngineMetadata) *BaseEngine {
	return &BaseEngine{
		metadata: meta,
		metrics:  &EngineMetrics{},
		timings:  make([]SentenceTiming, 0),
	}
}

// 注意：BaseEngine 不直接实现 Engine 接口，但提供了 WordTimestamps() 方法
// 组合了 BaseEngine 的 Engine 实现可以通过委托来满足 Engine 接口要求

func (b *BaseEngine) Metadata() *EngineMetadata {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.metadata
}

func (b *BaseEngine) Metrics() *EngineMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.metrics
}

// AddTiming 添加一个句子的时间信息
func (b *BaseEngine) AddTiming(timing SentenceTiming) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.timings = append(b.timings, timing)
}

// GetTimings 获取所有句子的时间信息（内部方法）
func (b *BaseEngine) GetTimings() []SentenceTiming {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]SentenceTiming, len(b.timings))
	copy(result, b.timings)
	return result
}

// WordTimestamps 实现 Engine 接口的 WordTimestamps() 方法
// 供组合了 BaseEngine 的 Engine 使用
func (b *BaseEngine) WordTimestamps() []SentenceTiming {
	return b.GetTimings()
}

// ResetTimings 清空时间信息
func (b *BaseEngine) ResetTimings() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.timings = b.timings[:0]
}

// GetCurrentWord 根据当前播放时间获取正在播放的词（内部方法，供 BaseEngine 使用）
// 如果当前时间不在任何词的时间范围内，返回最近播放的词或下一个要播放的词
func (b *BaseEngine) GetCurrentWord(currentTime float64) *WordTiming {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return GetCurrentWordFromTimings(b.timings, currentTime)
}
