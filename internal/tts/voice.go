package tts

// VoiceProfile 表示一个音色的完整配置信息
type VoiceProfile struct {
	// 必需字段
	VoiceType  string // 音色名称/ID，如 "zh_female_meilinvyou_saturn_bigtts"
	ResourceID string // 资源ID，如 "seed-tts-2.0"

	// 音色属性
	Language string // 语种，如 "zh"、"en"
	Version  string // 版本，如 "v2"、"2.0"
	Gender   string // 性别，如 "male"、"female"

	// 描述信息
	Name        string // 音色名称（简短），如 "meilin_nvyou"
	Description string // 详细描述，如 "温柔女声"

	// 支持的功能
	SupportedEmotions []string // 支持的情感列表，如 ["happy", "sad", "comfort"]

	// 默认配置（可选）
	DefaultEmotion    string  // 默认情感
	DefaultSpeedRatio float32 // 默认语速，如 1.0
	DefaultSampleRate int     // 默认采样率，如 16000

	// 元数据（用于扩展）
	Metadata map[string]string
}

// GetVoiceType 获取音色类型
func (v *VoiceProfile) GetVoiceType() string {
	return v.VoiceType
}

// GetResourceID 获取资源ID
func (v *VoiceProfile) GetResourceID() string {
	return v.ResourceID
}

// SupportsEmotion 检查是否支持指定的情感
func (v *VoiceProfile) SupportsEmotion(emotion string) bool {
	if len(v.SupportedEmotions) == 0 {
		return true // 如果没有限制，默认支持所有情感
	}
	for _, e := range v.SupportedEmotions {
		if e == emotion {
			return true
		}
	}
	return false
}

// ToVolcEngineOption 将 VoiceProfile 转换为 VolcEngineOption（需要提供认证信息）
func (v *VoiceProfile) ToVolcEngineOption(accessKey, appKey string, opts ...VolcEngineOptionModifier) VolcEngineOption {
	option := VolcEngineOption{
		VoiceType:  v.VoiceType,
		ResourceID: v.ResourceID,
		AccessKey:  accessKey,
		AppKey:     appKey,
		Encoding:   "pcm",
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
		SpeedRatio: 1.0,
	}

	// 应用默认值
	if v.DefaultSampleRate > 0 {
		option.SampleRate = v.DefaultSampleRate
	}
	if v.DefaultSpeedRatio > 0 {
		option.SpeedRatio = v.DefaultSpeedRatio
	}

	// 应用额外的修改器
	for _, modifier := range opts {
		modifier(&option)
	}

	return option
}

// WithBitDepth 设置位深度
func WithBitDepth(depth int) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.BitDepth = depth
	}
}

// WithChannels 设置声道数
func WithChannels(channels int) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.Channels = channels
	}
}

// VolcEngineOptionModifier 用于修改 VolcEngineOption 的函数类型
type VolcEngineOptionModifier func(*VolcEngineOption)

// WithSampleRate 设置采样率
func WithSampleRate(rate int) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.SampleRate = rate
	}
}

// WithSpeedRatio 设置语速
func WithSpeedRatio(ratio float32) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.SpeedRatio = ratio
	}
}

// WithEncoding 设置编码格式
func WithEncoding(encoding string) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.Encoding = encoding
	}
}
