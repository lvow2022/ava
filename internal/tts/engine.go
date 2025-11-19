package tts

import (
	"context"
)

type TTSEngine interface {
	Initialize(ctx context.Context) (*TTSStream, error)
	Synthesize(req *SynthesisRequest) error
	Stop() error
	Close() error
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
