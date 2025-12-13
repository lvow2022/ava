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
	Start(emotion string) (*Streamer, error) // emotion 参数用于设置情感，可以为空
	Synthesize(text string) error
	End() error

	// WordTimestamps 返回所有句子的词级别时间戳信息
	// 外部可以通过 WordTimestamps() 获取时间戳后，使用 GetCurrentWordFromTimings 辅助函数计算当前词
	WordTimestamps() []SentenceTiming
}

type EngineInfo struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
	Config       map[string]string
}
