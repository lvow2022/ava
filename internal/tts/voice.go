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
