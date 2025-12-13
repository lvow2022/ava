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

// AuthConfig 认证配置
type AuthConfig struct {
	AccessKey string
	AppKey    string
}

// VoiceConfig 音色配置
type VoiceConfig struct {
	Voice *VoiceProfile
}

// NewVoiceConfig 使用 VoiceProfile 创建音色配置
func NewVoiceConfig(voice *VoiceProfile) VoiceConfig {
	return VoiceConfig{Voice: voice}
}

// NewVoiceConfigByName 通过音色名称创建音色配置
func NewVoiceConfigByName(voiceName string) (VoiceConfig, error) {
	voice, ok := GetVoice(voiceName)
	if !ok {
		return VoiceConfig{}, fmt.Errorf("voice not found: %s. Available voices: %v", voiceName, ListVoices())
	}
	return VoiceConfig{Voice: &voice}, nil
}

// CodecConfig 编解码配置
type CodecConfig struct {
	Encoding   string  // 编码格式，默认 "pcm"
	SampleRate int     // 采样率，默认 16000
	BitDepth   int     // 位深度，默认 16
	Channels   int     // 声道数，默认 1
	SpeedRatio float32 // 语速，默认 1.0
}

// DefaultCodecConfig 返回默认编解码配置
func DefaultCodecConfig() CodecConfig {
	return CodecConfig{
		Encoding:   "pcm",
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
		SpeedRatio: 1.0,
	}
}

type VolcEngine struct {
	auth  AuthConfig
	voice VoiceConfig
	codec CodecConfig

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

	closeOnce sync.Once // 确保只关闭一次
}

// ------------------------ Constructor ------------------------

// NewVolcEngine 创建新的 VolcEngine 并自动建立连接
// auth 和 voice 是必需参数，codec 可选（如果未提供则使用默认值）
func NewVolcEngine(ctx context.Context, auth AuthConfig, voice VoiceConfig, codec ...CodecConfig) (*VolcEngine, error) {
	// 验证必需字段
	if auth.AccessKey == "" {
		return nil, errors.New("accessKey is required")
	}
	if auth.AppKey == "" {
		return nil, errors.New("appKey is required")
	}
	if voice.Voice == nil {
		return nil, errors.New("voice configuration is required, use NewVoiceConfig() or NewVoiceConfigByName()")
	}

	// 使用提供的 codec 配置或默认值
	var codecConfig CodecConfig
	if len(codec) > 0 {
		codecConfig = codec[0]
	} else {
		codecConfig = DefaultCodecConfig()
		// 如果音色有默认值，应用它们
		if voice.Voice.DefaultSampleRate > 0 {
			codecConfig.SampleRate = voice.Voice.DefaultSampleRate
		}
		if voice.Voice.DefaultSpeedRatio > 0 {
			codecConfig.SpeedRatio = voice.Voice.DefaultSpeedRatio
		}
	}

	e := &VolcEngine{
		auth:                auth,
		voice:               voice,
		codec:               codecConfig,
		connectionStartedCh: make(chan struct{}),
		sessionFinishedCh:   make(chan struct{}),
		sessionStartedCh:    make(chan struct{}),
	}

	// 创建 context
	e.ctx, e.cancel = context.WithCancel(ctx)

	// 建立 WebSocket 连接
	header := http.Header{}
	header.Set("X-Api-App-Key", auth.AppKey)
	header.Set("X-Api-Access-Key", auth.AccessKey)
	header.Set("X-Api-Resource-Id", voice.Voice.ResourceID)
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
	e.streamer = tts.NewStreamer(beep.SampleRate(e.codec.SampleRate), e.codec.Channels)
	e.mu.Unlock()

	e.SessionID = uuid.New().String()

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

// Close 主动关闭连接并清理资源
// 实现 Engine 接口，可以安全地多次调用，只会关闭一次
func (e *VolcEngine) Close() error {
	e.closeOnce.Do(func() {
		// 先关闭 streamer
		e.mu.Lock()
		if e.streamer != nil {
			e.streamer.Close()
			e.streamer = nil
		}
		e.mu.Unlock()

		// 关闭 WebSocket 连接（这会触发 OnClose 回调）
		if e.client != nil {
			e.client.Close()
		}

		// 取消 context
		if e.cancel != nil {
			e.cancel()
		}

		logrus.Info("volc: engine closed")
	})
	return nil
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

func (e *VolcEngine) startSession(emotion string) error {
	audioParams := &AudioParams{
		Format:          e.codec.Encoding,
		SampleRate:      int32(e.codec.SampleRate),
		EnableTimestamp: true,
		SpeechRate:      convertSpeechRate(e.codec.SpeedRatio),
		Emotion:         emotion,
	}

	req := NewRequestBuilder().
		WithEvent(EventType_StartSession).
		WithSpeaker(e.voice.Voice.VoiceType).
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

	// 直接添加到 streamer 的时间戳列表
	e.mu.Lock()
	if e.streamer != nil {
		e.streamer.AddTiming(timing)
	}
	e.mu.Unlock()

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
