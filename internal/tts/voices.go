package tts

// 预定义音色库
// 这些是火山引擎常用的音色配置

var (
	// 中文女声
	VoiceMeilinNvyou = VoiceProfile{
		VoiceType:  "zh_female_meilinvyou_saturn_bigtts",
		ResourceID: "seed-tts-2.0",
		Language:   "zh",
		Version:    "2.0",
		Gender:     "female",
		Name:       "meilin_nvyou",
		Description: "温柔女声",
		SupportedEmotions: []string{
			"happy", "sad", "angry", "surprised", "fear", "hate",
			"excited", "coldness", "neutral", "depressed",
			"lovey-dovey", "shy", "comfort", "tension",
			"tender", "storytelling", "radio", "magnetic",
		},
		DefaultEmotion:   "neutral",
		DefaultSpeedRatio: 1.0,
		DefaultSampleRate: 16000,
	}

	VoiceLengkuGege = VoiceProfile{
		VoiceType:  "zh_male_lengkugege_emo_v2_mars_bigtts",
		ResourceID: "seed-tts-1.0",
		Language:   "zh",
		Version:    "v2",
		Gender:     "male",
		Name:       "lengku_gege",
		Description: "磁性男声",
		SupportedEmotions: []string{
			"happy", "sad", "angry", "surprised", "fear", "hate",
			"excited", "coldness", "neutral", "depressed",
			"lovey-dovey", "shy", "comfort", "tension",
			"tender", "storytelling", "radio", "magnetic",
		},
		DefaultEmotion:   "neutral",
		DefaultSpeedRatio: 1.1,
		DefaultSampleRate: 16000,
	}

	// 可以继续添加更多音色...
)

// VoiceRegistry 音色注册表，用于通过名称快速查找音色
var VoiceRegistry = map[string]VoiceProfile{
	"meilin_nvyou": VoiceMeilinNvyou,
	"lengku_gege":  VoiceLengkuGege,
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

