package tts

import "sync"

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
