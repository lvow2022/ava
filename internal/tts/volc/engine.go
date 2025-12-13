package volc

import (
	"ava/internal/tts"
	"ava/pkg/ws"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gopxl/beep"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func init() {
	//RegisterTTS("tts.volcengine", NewVolcStreamEngine)
}

type VolcEngineOption struct {
	VoiceType  string  `json:"voiceType"`
	ResourceID string  `json:"resourceID"`
	AccessKey  string  `json:"accessKey" `
	AppKey     string  `json:"appKey"`
	Encoding   string  `json:"encoding" default:"pcm"`
	SampleRate int     `json:"sampleRate"  default:"16000"`
	BitDepth   int     `json:"bitDepth"  default:"16"`
	Channels   int     `json:"channels"  default:"1"`
	SpeedRatio float32 `json:"speedRatio"  default:"1.0"`
}

// VolcEngineOptionModifier 用于修改 VolcEngineOption 的函数类型
type VolcEngineOptionModifier func(*VolcEngineOption)

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

// WithAccessKey 设置访问密钥
func WithAccessKey(accessKey string) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.AccessKey = accessKey
	}
}

// WithAppKey 设置应用密钥
func WithAppKey(appKey string) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.AppKey = appKey
	}
}

// WithVoice 使用预定义音色配置
func WithVoice(voice VoiceProfile) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		opt.VoiceType = voice.VoiceType
		opt.ResourceID = voice.ResourceID
		// 应用音色的默认值
		if voice.DefaultSampleRate > 0 {
			opt.SampleRate = voice.DefaultSampleRate
		}
		if voice.DefaultSpeedRatio > 0 {
			opt.SpeedRatio = voice.DefaultSpeedRatio
		}
	}
}

// WithVoiceName 通过音色名称查找并配置音色
// 如果音色不存在，会在 NewVolcEngine 中返回包含可用音色列表的错误
func WithVoiceName(voiceName string) VolcEngineOptionModifier {
	return func(opt *VolcEngineOption) {
		voice, ok := GetVoice(voiceName)
		if !ok {
			// 清空以触发验证错误，错误信息会包含可用音色列表
			opt.VoiceType = ""
			opt.ResourceID = ""
			return
		}
		opt.VoiceType = voice.VoiceType
		opt.ResourceID = voice.ResourceID
		// 应用音色的默认值
		if voice.DefaultSampleRate > 0 {
			opt.SampleRate = voice.DefaultSampleRate
		}
		if voice.DefaultSpeedRatio > 0 {
			opt.SpeedRatio = voice.DefaultSpeedRatio
		}
	}
}

type VolcEngine struct {
	*tts.BaseEngine
	opt    VolcEngineOption
	client *ws.WSClient

	mu       sync.Mutex
	streamer *tts.Streamer // 单 session streamer

	SessionID string

	ctx    context.Context
	cancel context.CancelFunc

	connectionStartedCh chan struct{}
	sessionFinishedCh   chan struct{}
	sessionStartedCh    chan struct{}
	recvFirstAudio      bool
}

// ------------------------ Constructor ------------------------

// NewVolcEngine 使用 option 模式创建引擎并自动建立连接
// 必需参数通过 option 函数提供：WithAccessKey, WithAppKey, 以及音色配置（WithVoice/WithVoiceName 或直接设置 VoiceType/ResourceID）
func NewVolcEngine(ctx context.Context, opts ...VolcEngineOptionModifier) (*VolcEngine, error) {
	// 初始化默认配置
	opt := VolcEngineOption{
		Encoding:   "pcm",
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
		SpeedRatio: 1.0,
	}

	// 应用所有 option
	for _, modifier := range opts {
		modifier(&opt)
	}

	// 验证必需字段
	if opt.AccessKey == "" {
		return nil, errors.New("accessKey is required, use WithAccessKey() option")
	}
	if opt.AppKey == "" {
		return nil, errors.New("appKey is required, use WithAppKey() option")
	}
	if opt.VoiceType == "" && opt.ResourceID == "" {
		return nil, fmt.Errorf("voice configuration is required, use WithVoice(), WithVoiceName(), or set VoiceType/ResourceID directly. Available voices: %v", ListVoices())
	}

	e := &VolcEngine{
		BaseEngine:          tts.NewBaseEngine(nil),
		opt:                 opt,
		connectionStartedCh: make(chan struct{}),
		sessionFinishedCh:   make(chan struct{}),
		sessionStartedCh:    make(chan struct{}),
	}

	// 创建 context
	e.ctx, e.cancel = context.WithCancel(ctx)

	// 建立 WebSocket 连接
	header := http.Header{}
	header.Set("X-Api-App-Key", opt.AppKey)
	header.Set("X-Api-Access-Key", opt.AccessKey)
	header.Set("X-Api-Resource-Id", opt.ResourceID)
	header.Set("X-Api-Connect-Id", uuid.New().String())
	header.Set("X-Control-Require-Usage-Tokens-Return", "*")

	endpoint := "wss://openspeech.bytedance.com/api/v3/tts/bidirection"

	client, err := ws.NewWSClient(endpoint, header, e)
	if err != nil {
		e.cancel() // 清理 context
		return nil, fmt.Errorf("volc: dial websocket: %w", err)
	}

	e.client = client

	// 等待连接启动
	select {
	case <-e.connectionStartedCh:
		logrus.Info("volc: connection started")
	case <-time.After(5 * time.Second):
		e.cancel()
		if e.client != nil {
			e.client.Close()
		}
		return nil, errors.New("volc: start connection timeout")
	case <-e.ctx.Done():
		if e.client != nil {
			e.client.Close()
		}
		return nil, e.ctx.Err()
	}

	return e, nil
}

// ------------------------ EventHandler 实现 ------------------------

// OnOpen 实现 EventHandler 接口，连接建立时调用
func (e *VolcEngine) OnOpen(c *ws.WSClient) {
	msg := NewMessageBuilder().
		WithEventType(EventType_StartConnection).
		WithPayload([]byte("{}")).
		Build()

	frame, _ := msg.Marshal()
	c.SendText(frame)
}

func (e *VolcEngine) OnMessage(c *ws.WSClient, msgType int, msg []byte) {
	protocolMsg, err := NewMessageFromBytes(msg)
	if err != nil {
		logrus.Warnf("volc: failed to parse message: %v", err)
		return
	}

	e.dispatch(protocolMsg)
}

func (e *VolcEngine) OnError(c *ws.WSClient, err error) {
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		logrus.Info("volc: normal ws close")
	} else {
		logrus.Warnf("volce: ws close error: %v", err)
	}
}

func (e *VolcEngine) OnClose(c *ws.WSClient) {
	logrus.Info("volc: connection closed")

	e.mu.Lock()
	if e.streamer != nil {
		e.streamer.Close()
		e.streamer = nil
	}
	e.mu.Unlock()

	if e.cancel != nil {
		e.cancel()
	}
}

func (e *VolcEngine) dispatch(msg *Message) {
	logrus.Infof("volc: recv message: %s", msg.String())
	switch {
	case msg.MsgType == MsgTypeFullServerResponse &&
		msg.EventType == EventType_ConnectionStarted:
		e.connectionStartedCh <- struct{}{}

	case msg.EventType == EventType_SessionStarted:
		e.sessionStartedCh <- struct{}{}

	case msg.MsgType == MsgTypeAudioOnlyServer:
		e.mu.Lock()
		streamer := e.streamer
		e.mu.Unlock()
		if streamer != nil {
			streamer.AppendAudio(msg.Payload)
		}

	case msg.MsgType == MsgTypeFullServerResponse &&
		msg.EventType == EventType_TTSSentenceEnd:
		e.handleSentenceEnd(msg.Payload)

	case msg.EventType == EventType_SessionFinished:
		e.sessionFinishedCh <- struct{}{}

	case msg.MsgType == MsgTypeError:
		logrus.Error("volc: received error message: ", msg.String())

	}
}

// ------------------------ Session Logic ------------------------

func (e *VolcEngine) Start(emotion string) (*tts.Streamer, error) {
	e.mu.Lock()
	if e.streamer != nil {
		e.streamer.Close()
	}
	e.streamer = tts.NewStreamer(beep.SampleRate(e.opt.SampleRate), e.opt.Channels)
	e.mu.Unlock()

	e.SessionID = uuid.New().String()
	e.resetTimings()

	if err := e.startSession(emotion); err != nil {
		return nil, err
	}

	select {
	case <-e.sessionStartedCh:
		logrus.Info("volc: session started")
	case <-time.After(5 * time.Second):
		return nil, errors.New("volc: start session timeout")

	}

	return e.streamer, nil
}

func (e *VolcEngine) End() error {
	defer func() {
		e.mu.Lock()
		if e.streamer != nil {
			e.streamer.Close()
			e.streamer = nil
		}
		e.mu.Unlock()
	}()

	e.finishSession()

	select {
	case <-e.sessionFinishedCh:
		logrus.Info("volc: session finished")
	case <-time.After(30 * time.Second):
		return errors.New("volc: finish session timeout")

	}

	return nil
}

func (e *VolcEngine) Synthesize(text string) error {
	req := NewRequestBuilder().
		WithEvent(EventType_TaskRequest).
		WithText(text).
		Build()
	payload, _ := json.Marshal(req)

	msg := NewMessageBuilder().
		WithEventType(EventType_TaskRequest).
		WithSessionID(e.SessionID).
		WithPayload(payload).
		Build()

	frame, _ := msg.Marshal()
	e.client.SendText(frame)

	logrus.Info("volc: send task request: ", string(payload))
	return nil
}

// ------------------------ Streamer helpers ------------------------

func (e *VolcEngine) resetTimings() {
	e.BaseEngine.ResetTimings()
}

func (e *VolcEngine) WordTimestamps() []tts.SentenceTiming {
	return e.BaseEngine.WordTimestamps()
}

func (e *VolcEngine) buildStartSessionRequest(emotion string) ([]byte, error) {
	audioParams := &AudioParams{
		Format:          e.opt.Encoding,
		SampleRate:      int32(e.opt.SampleRate),
		EnableTimestamp: true,
		SpeechRate:      convertSpeechRate(e.opt.SpeedRatio),
		Emotion:         emotion,
	}

	startReq := NewRequestBuilder().
		WithEvent(EventType_StartSession).
		WithSpeaker(e.opt.VoiceType).
		WithAudioParams(audioParams).
		Build()

	return json.Marshal(startReq)
}

func (e *VolcEngine) startSession(emotion string) error {
	audioParams := &AudioParams{
		Format:          e.opt.Encoding,
		SampleRate:      int32(e.opt.SampleRate),
		EnableTimestamp: true,
		SpeechRate:      convertSpeechRate(e.opt.SpeedRatio),
		Emotion:         emotion,
	}

	req := NewRequestBuilder().
		WithEvent(EventType_StartSession).
		WithSpeaker(e.opt.VoiceType).
		WithAudioParams(audioParams).
		Build()
	payload, _ := json.Marshal(req)

	msg := NewMessageBuilder().
		WithEventType(EventType_StartSession).
		WithSessionID(e.SessionID).
		WithPayload(payload).
		Build()

	frame, _ := msg.Marshal()
	e.client.SendText(frame)

	return nil
}

func (e *VolcEngine) finishSession() {
	msg := NewMessageBuilder().
		WithEventType(EventType_FinishSession).
		WithSessionID(e.SessionID).
		WithPayload([]byte("{}")).
		Build()

	frame, _ := msg.Marshal()
	e.client.SendText(frame)
}

func (e *VolcEngine) handleSentenceEnd(payload []byte) {
	var timing tts.SentenceTiming
	if err := json.Unmarshal(payload, &timing); err != nil {
		logrus.Warnf("volc: failed to parse sentence end timing: %v", err)
		return
	}

	// 添加到 BaseEngine 的时间戳列表
	e.BaseEngine.AddTiming(timing)

	// 获取时间戳的副本用于更新 streamer
	timingsCopy := e.BaseEngine.GetTimings()

	// 更新 streamer 的时间信息（需要加锁保护）
	e.mu.Lock()
	streamer := e.streamer
	e.mu.Unlock()

	if streamer != nil {
		// 更新 streamer 的时间信息
		streamer.SetTimings(timingsCopy)

		// 计算总时长并更新 streamer
		if len(timing.Words) > 0 {
			lastWord := timing.Words[len(timing.Words)-1]
			totalDuration := lastWord.EndTime
			streamer.SetTotalDuration(totalDuration)
			logrus.Infof("volc: sentence end, total duration: %.2fs, words: %d", totalDuration, len(timing.Words))
		}
	}
}

func convertSpeechRate(speedRatio float32) int32 {
	var rate float32
	switch {
	case speedRatio <= 1:
		// 0–1 之间: 从 -50 → 0
		rate = -50 + 50*speedRatio
	default:
		// 1–2 之间: 从 0 → 100
		rate = 100 * (speedRatio - 1)
	}

	if rate < -50 {
		rate = -50
	} else if rate > 100 {
		rate = 100
	}
	return int32(rate)
}
