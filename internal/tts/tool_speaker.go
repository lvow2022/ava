package tts

import (
	"context"
	"errors"
	"sync"
)

// ToolSpeakerManager 管理全局的 Speaker 实例，支持懒加载和线程安全访问
type ToolSpeakerManager struct {
	mu      sync.RWMutex
	once    sync.Once
	speaker *Speaker
	engine  Engine
	config  *ToolSpeakerConfig
	initErr error
}

// ToolSpeakerConfig 配置 Speaker 管理器
type ToolSpeakerConfig struct {
	// VolcEngine 配置
	VoiceType  string
	ResourceID string
	AccessKey  string
	AppKey     string
	Encoding   string
	SampleRate int
	BitDepth   int
	Channels   int
	SpeedRatio float32
}

var (
	globalManager *ToolSpeakerManager
	managerOnce   sync.Once
)

// GetGlobalManager 获取全局的 Speaker 管理器实例
func GetGlobalManager() *ToolSpeakerManager {
	managerOnce.Do(func() {
		globalManager = &ToolSpeakerManager{}
	})
	return globalManager
}

// SetConfig 设置配置（需要在初始化前调用）
func (m *ToolSpeakerManager) SetConfig(config *ToolSpeakerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// GetSpeaker 获取 Speaker 实例，如果未初始化则进行懒加载初始化
func (m *ToolSpeakerManager) GetSpeaker(ctx context.Context) (*Speaker, error) {
	m.mu.RLock()
	if m.speaker != nil {
		speaker := m.speaker
		m.mu.RUnlock()
		return speaker, nil
	}
	if m.initErr != nil {
		err := m.initErr
		m.mu.RUnlock()
		return nil, err
	}
	m.mu.RUnlock()

	// 执行初始化（只执行一次）
	m.once.Do(func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.config == nil {
			m.initErr = errors.New("ToolSpeakerConfig not set, call SetConfig first")
			return
		}

		// 创建 Engine
		engineOpt := VolcEngineOption{
			VoiceType:  m.config.VoiceType,
			ResourceID: m.config.ResourceID,
			AccessKey:  m.config.AccessKey,
			AppKey:     m.config.AppKey,
			Encoding:   m.config.Encoding,
			SampleRate: m.config.SampleRate,
			BitDepth:   m.config.BitDepth,
			Channels:   m.config.Channels,
			SpeedRatio: m.config.SpeedRatio,
		}

		engine, err := NewVolcEngine(engineOpt)
		if err != nil {
			m.initErr = err
			return
		}

		// 初始化 Engine
		if err := engine.Initialize(ctx); err != nil {
			m.initErr = err
			return
		}

		m.engine = engine

		// 创建 Speaker
		// 注意：不需要在这里启动 session，Say() 方法会自动调用 StartSession()
		speaker := NewSpeaker(engine)
		m.speaker = speaker
	})

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.initErr != nil {
		return nil, m.initErr
	}
	return m.speaker, nil
}

// Close 关闭 Speaker 和 Engine，释放资源
func (m *ToolSpeakerManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	if m.speaker != nil {
		m.speaker.Stop()
	}

	if m.engine != nil {
		if err := m.engine.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

