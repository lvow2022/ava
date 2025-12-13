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
}

func NewBaseEngine(meta *EngineMetadata) *BaseEngine {
	return &BaseEngine{
		metadata: meta,
		metrics:  &EngineMetrics{},
	}
}

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
