package tts

import (
	"context"
)

type Engine interface {
	Initialize(ctx context.Context) error
	Start() (*Streamer, error)
	Synthesize(text string) error
	End() error
	Close() error
}

type EngineInfo struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
	Config       map[string]string
}
