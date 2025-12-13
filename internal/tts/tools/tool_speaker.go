package tools

import (
	"ava/internal/tts"
	"ava/internal/tts/volc"
	"context"
	"errors"
	"sync"
)

// ToolSpeakerManager 管理全局的 Speaker 实例，支持懒加载和线程安全访问
type ToolSpeakerManager struct {
	mu      sync.RWMutex
	once    sync.Once
	speaker *tts.Speaker
	engine  tts.Engine
	config  *ToolSpeakerConfig
	initErr error
}

// ToolSpeakerConfig 配置 Speaker 管理器
type ToolSpeakerConfig struct {
	// 认证信息
	AccessKey string
	AppKey    string

	// 方式1：使用预定义音色（推荐）
	Voice volc.VoiceProfile

	// 方式2：使用音色名称（便捷方式）
	VoiceName string

	// 方式3：传统方式（保持向后兼容）
	VoiceType  string
	ResourceID string

	// 音频参数（可选，会覆盖音色的默认值）
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
func (m *ToolSpeakerManager) GetSpeaker(ctx context.Context) (*tts.Speaker, error) {
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
		var engine *volc.VolcEngine
		var err error

		// 构建 option 列表
		opts := []volc.VolcEngineOptionModifier{
			volc.WithAccessKey(m.config.AccessKey),
			volc.WithAppKey(m.config.AppKey),
		}

		// 配置音色
		if m.config.Voice.VoiceType != "" {
			opts = append(opts, volc.WithVoice(m.config.Voice))
		} else if m.config.VoiceName != "" {
			opts = append(opts, volc.WithVoiceName(m.config.VoiceName))
		} else {
			// 传统方式：直接设置 VoiceType 和 ResourceID
			voiceType := m.config.VoiceType
			resourceID := m.config.ResourceID
			opts = append(opts, func(opt *volc.VolcEngineOption) {
				opt.VoiceType = voiceType
				opt.ResourceID = resourceID
			})
		}

		// 配置音频参数
		if m.config.Encoding != "" {
			opts = append(opts, volc.WithEncoding(m.config.Encoding))
		}
		if m.config.SampleRate > 0 {
			opts = append(opts, volc.WithSampleRate(m.config.SampleRate))
		}
		if m.config.SpeedRatio > 0 {
			opts = append(opts, volc.WithSpeedRatio(m.config.SpeedRatio))
		}
		if m.config.BitDepth > 0 {
			opts = append(opts, volc.WithBitDepth(m.config.BitDepth))
		}
		if m.config.Channels > 0 {
			opts = append(opts, volc.WithChannels(m.config.Channels))
		}

		engine, err = volc.NewVolcEngine(ctx, opts...)

		if err != nil {
			m.initErr = err
			return
		}

		m.engine = engine

		// 创建 Speaker
		// 注意：不需要在这里启动 session，Say() 方法会自动调用 Start()
		speaker := tts.NewSpeaker(engine)
		m.speaker = speaker
	})

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.initErr != nil {
		return nil, m.initErr
	}
	return m.speaker, nil
}

// Close 关闭 Speaker，释放资源
// Engine 的生命周期由 context 管理，连接关闭时会自动清理
func (m *ToolSpeakerManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.speaker != nil {
		m.speaker.Stop()
	}

	// Engine 不需要显式关闭，资源会在连接关闭时自动清理
	return nil
}

