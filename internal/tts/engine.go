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
	Close() error // 关闭连接并清理资源
}

type EngineInfo struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
	Config       map[string]string
}
