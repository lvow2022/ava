package audio

import (
	"context"
	"time"
)

type EngineMetadata struct {
	SessionID   string `json:"sessionID"`
	DialogID    string `json:"dialogID"`
	PlayID      int    `json:"playID"`
	Sequence    int    `json:"sequence"`
	IsStreaming bool   `json:"isStreaming"` // 是否为流式请求
}

type EngineMetrics struct {
	TTFB               int64     `json:"ttfb"`               // 首字节延迟（毫秒）
	SynthesisStartTime time.Time `json:"synthesisStartTime"` // 合成开始时间
	FirstAudioTime     time.Time `json:"firstAudioTime"`     // 首音频时间
	FirstRecvTime      time.Time `json:"firstRecvTime"`      // 第一次 recv 时间
	StreamCloseTime    time.Time `json:"streamCloseTime"`    // 流 close 时间
	SessionStarted     bool      `json:"sessionStarted"`     // 会话是否已开始
	SessionID          string    `json:"sessionID"`          // 会话ID
	PlayID             int       `json:"playID"`             // 播放ID
	DialogID           string    `json:"dialogID"`           // 对话ID
	Sequence           int       `json:"sequence"`           // 序列号
	Text               string    `json:"text"`               // 合成文本
	SampleRate         int       `json:"sampleRate"`         // 采样率
}

type TTSEngine interface {
	Initialize(ctx context.Context, metadata *EngineMetadata) (*TTSStream, error)
	Synthesize(ctx context.Context, req *SynthesisRequest) error
	Stop() error
	Close() error
	Metrics() *EngineMetrics
	CacheKey(text string) string
}

type EngineInfo struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
	Config       map[string]string
}

type SynthesisRequest struct {
	SessionID string                 `json:"sessionID"`
	DialogID  string                 `json:"dialogID"`
	Text      string                 `json:"text"`
	PlayID    int                    `json:"playID"`
	Sequence  int                    `json:"sequence"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Finish    bool                   `json:"-"`
}
