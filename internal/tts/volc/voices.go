package volc

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

// 预定义音色库
// 这些是火山引擎常用的音色配置

var (
	// 中文女声
	VoiceMeilinNvyou = VoiceProfile{
		VoiceType:   "zh_female_meilinvyou_saturn_bigtts",
		ResourceID:  "seed-tts-2.0",
		Language:    "zh",
		Version:     "2.0",
		Gender:      "female",
		Name:        "meilin_nvyou",
		Description: "魅力女友",
		SupportedEmotions: []string{
			"happy", "sad", "angry", "surprised", "fear", "hate",
			"excited", "coldness", "neutral", "depressed",
			"lovey-dovey", "shy", "comfort", "tension",
			"tender", "storytelling", "radio", "magnetic",
		},
		DefaultEmotion:    "neutral",
		DefaultSpeedRatio: 1.0,
		DefaultSampleRate: 16000,
	}
	VoiceTiaoPigongzhu = VoiceProfile{
		VoiceType:   "saturn_zh_female_tiaopigongzhu_tob",
		ResourceID:  "seed-tts-2.0",
		Language:    "zh",
		Version:     "v2",
		Gender:      "female",
		Name:        "tiaopigongzhu",
		Description: "调皮公主",
		SupportedEmotions: []string{
			"happy", "sad", "angry", "surprised", "fear", "hate",
			"excited", "coldness", "neutral", "depressed",
			"lovey-dovey", "shy", "comfort", "tension",
			"tender", "storytelling", "radio", "magnetic",
		},
		DefaultEmotion:    "neutral",
		DefaultSpeedRatio: 1.1,
		DefaultSampleRate: 16000,
	}

	VoiceLengkuGege = VoiceProfile{
		VoiceType:   "zh_male_lengkugege_emo_v2_mars_bigtts",
		ResourceID:  "seed-tts-1.0",
		Language:    "zh",
		Version:     "v2",
		Gender:      "male",
		Name:        "lengku_gege",
		Description: "冷酷哥哥",
		SupportedEmotions: []string{
			"happy", "sad", "angry", "surprised", "fear", "hate",
			"excited", "coldness", "neutral", "depressed",
		},
		DefaultEmotion:    "neutral",
		DefaultSpeedRatio: 1.1,
		DefaultSampleRate: 16000,
	}

	// 可以继续添加更多音色...
)

// VoiceRegistry 音色注册表，用于通过名称快速查找音色
var VoiceRegistry = map[string]VoiceProfile{
	"meilin_nvyou":  VoiceMeilinNvyou,
	"tiaopigongzhu": VoiceTiaoPigongzhu,
	"lengku_gege":   VoiceLengkuGege,
	// 可以继续添加更多音色...
}

// GetVoice 根据名称获取音色配置
func GetVoice(name string) (VoiceProfile, bool) {
	voice, ok := VoiceRegistry[name]
	return voice, ok
}

// RegisterVoice 注册新的音色（运行时动态添加）
func RegisterVoice(name string, voice VoiceProfile) {
	VoiceRegistry[name] = voice
}

// ListVoices 列出所有已注册的音色名称
func ListVoices() []string {
	names := make([]string, 0, len(VoiceRegistry))
	for name := range VoiceRegistry {
		names = append(names, name)
	}
	return names
}

// FindVoicesByLanguage 根据语种查找音色
func FindVoicesByLanguage(language string) []VoiceProfile {
	var voices []VoiceProfile
	for _, voice := range VoiceRegistry {
		if voice.Language == language {
			voices = append(voices, voice)
		}
	}
	return voices
}

// FindVoicesByGender 根据性别查找音色
func FindVoicesByGender(gender string) []VoiceProfile {
	var voices []VoiceProfile
	for _, voice := range VoiceRegistry {
		if voice.Gender == gender {
			voices = append(voices, voice)
		}
	}
	return voices
}
