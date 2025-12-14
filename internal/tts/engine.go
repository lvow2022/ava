package tts

// WordTiming 表示一个词的时间信息
type WordTiming struct {
	Word       string  `json:"word"`
	StartTime  float64 `json:"startTime"`
	EndTime    float64 `json:"endTime"`
	Confidence float64 `json:"confidence"`
}

// SentenceTiming 表示一个句子的时间信息
type SentenceTiming struct {
	Text  string       `json:"text"`
	Words []WordTiming `json:"words"`
}

type Engine interface {
	Start(emotion string, contextTexts []string) (*Streamer, error) // 启动 session，emotion 参数用于设置情感（可选，推荐使用 contextTexts 替代），contextTexts 用于上下文辅助合成（推荐使用，可通过自然语言描述替代 emotion）
	Synthesize(text string, contextTexts []string) error            // 合成文本，contextTexts 用于上下文辅助合成（推荐使用，可通过自然语言描述替代 emotion）
	End() error
	Close() error // 关闭连接并清理资源
}

type EngineInfo struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
	Config       map[string]string
}
