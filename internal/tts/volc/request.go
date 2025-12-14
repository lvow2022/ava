package volc

import "encoding/json"

// Request 表示发送给火山引擎的请求结构
type Request struct {
	User      *User      `json:"user"`
	Event     int32      `json:"event"`
	Namespace string     `json:"namespace"`
	ReqParams *ReqParams `json:"req_params"`
}

type RequestBuilder struct {
	req Request
}

func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		req: Request{
			Namespace: "BidirectionalTTS",
		},
	}
}

func (b *RequestBuilder) WithEvent(eventType EventType) *RequestBuilder {
	b.req.Event = int32(eventType)
	return b
}

func (b *RequestBuilder) WithNamespace(namespace string) *RequestBuilder {
	b.req.Namespace = namespace
	return b
}

func (b *RequestBuilder) WithUser(user *User) *RequestBuilder {
	b.req.User = user
	return b
}

func (b *RequestBuilder) WithText(text string) *RequestBuilder {
	if b.req.ReqParams == nil {
		b.req.ReqParams = &ReqParams{}
	}
	b.req.ReqParams.Text = text
	return b
}

func (b *RequestBuilder) WithSpeaker(speaker string) *RequestBuilder {
	if b.req.ReqParams == nil {
		b.req.ReqParams = &ReqParams{}
	}
	b.req.ReqParams.Speaker = speaker
	return b
}

func (b *RequestBuilder) WithAudioParams(audioParams *AudioParams) *RequestBuilder {
	if b.req.ReqParams == nil {
		b.req.ReqParams = &ReqParams{}
	}
	b.req.ReqParams.AudioParams = audioParams
	return b
}

func (b *RequestBuilder) WithContextTexts(contextTexts []string) *RequestBuilder {
	if b.req.ReqParams == nil {
		b.req.ReqParams = &ReqParams{}
	}
	// 构建 additions 对象
	additions := map[string]interface{}{
		"context_texts": contextTexts,
	}
	// 将 additions 对象序列化为 JSON 字符串
	if additionsJSON, err := json.Marshal(additions); err == nil {
		b.req.ReqParams.Additions = string(additionsJSON)
	}
	return b
}

func (b *RequestBuilder) Build() *Request {
	return &b.req
}

type User struct {
	Uid            string `json:"uid,omitempty"`
	Did            string `json:"did,omitempty"`
	DevicePlatform string `json:"device_platform,omitempty"`
	DeviceType     string `json:"device_type,omitempty"`
	VersionCode    string `json:"version_code,omitempty"`
	Language       string `json:"language,omitempty"`
}

// ReqParams 请求参数
type ReqParams struct {
	Text        string       `json:"text,omitempty"`
	Texts       *Texts       `json:"texts,omitempty"`
	Ssml        string       `json:"ssml,omitempty"`
	Speaker     string       `json:"speaker,omitempty"`
	AudioParams *AudioParams `json:"audio_params,omitempty"`
	Additions   string       `json:"additions,omitempty"` // additions 需要是 JSON 字符串
}

// ReqTexts 文本列表
type Texts struct {
	Texts []string `json:"texts,omitempty"`
}

// AudioParams 音频参数
type AudioParams struct {
	Format          string `json:"format,omitempty"`
	SampleRate      int32  `json:"sample_rate,omitempty"`
	Channel         int32  `json:"channel,omitempty"`
	SpeechRate      int32  `json:"speech_rate,omitempty"`
	PitchRate       int32  `json:"pitch_rate,omitempty"`
	BitRate         int32  `json:"bit_rate,omitempty"`
	Volume          int32  `json:"volume,omitempty"`
	Lang            string `json:"lang,omitempty"`
	Emotion         string `json:"emotion,omitempty"`
	EnableTimestamp bool   `json:"enable_timestamp,omitempty"`
}
