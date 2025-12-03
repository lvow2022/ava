package tts

import (
	"context"
)

type Engine interface {
	Initialize(ctx context.Context) error
	Start(emotion string) (*Streamer, error) // emotion 参数用于设置情感，可以为空
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
