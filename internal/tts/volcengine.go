package tts

import (
	"ava/internal/protocols"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gopxl/beep"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func init() {
	//RegisterTTS("tts.volcengine", NewVolcStreamEngine)
}

type VolcRequest struct {
	User      *VolcReqUser   `json:"user"`
	Event     int32          `json:"event"`
	Namespace string         `json:"namespace"`
	ReqParams *VolcReqParams `json:"req_params"`
}

type VolcReqUser struct {
	Uid            string `json:"uid,omitempty"`
	Did            string `json:"did,omitempty"`
	DevicePlatform string `json:"device_platform,omitempty"`
	DeviceType     string `json:"device_type,omitempty"`
	VersionCode    string `json:"version_code,omitempty"`
	Language       string `json:"language,omitempty"`
}

type VolcReqParams struct {
	Text           string               `json:"text,omitempty"`
	Texts          *VolcReqTexts        `json:"texts,omitempty"`
	Ssml           string               `json:"ssml,omitempty"`
	Speaker        string               `json:"speaker,omitempty"`
	AudioParams    *VolcReqAudioParams  `json:"audio_params,omitempty"`
	EngineParams   *VolcReqEngineParams `json:"engine_params,omitempty"`
	EnableAudio2Bs bool                 `json:"enable_audio2bs,omitempty"`
	EnableTextSeg  bool                 `json:"enable_text_seg,omitempty"`
	Additions      map[string]string    `json:"additions,omitempty"`
}

type VolcReqTexts struct {
	Texts []string `json:"texts,omitempty"`
}

type VolcReqAudioParams struct {
	Format          string `json:"format,omitempty"`
	SampleRate      int32  `json:"sample_rate,omitempty"`
	Channel         int32  `json:"channel,omitempty"`
	SpeechRate      int32  `json:"speech_rate,omitempty"`
	PitchRate       int32  `json:"pitch_rate,omitempty"`
	BitRate         int32  `json:"bit_rate,omitempty"`
	Volume          int32  `json:"volume,omitempty"`
	Lang            string `json:"lang,omitempty"`
	Emotion         string `json:"emotion,omitempty"`
	Gender          string `json:"gender,omitempty"`
	EnableTimestamp bool   `json:"enable_timestamp,omitempty"`
}

type VolcReqEngineParams struct {
	EngineContext                string   `json:"engine_context,omitempty"`
	PhonemeSize                  string   `json:"phoneme_size,omitempty"`
	EnableFastTextSeg            bool     `json:"enable_fast_text_seg,omitempty"`
	ForceBreak                   bool     `json:"force_break,omitempty"`
	BreakByProsody               int32    `json:"break_by_prosody,omitempty"`
	EnableEngineDebugInfo        bool     `json:"enable_engine_debug_info,omitempty"`
	FlushSentence                bool     `json:"flush_sentence,omitempty"`
	LabVersion                   string   `json:"lab_version,omitempty"`
	EnableIpaExtraction          bool     `json:"enable_ipa_extraction,omitempty"`
	EnableNaiveTn                bool     `json:"enable_naive_tn,omitempty"`
	EnableLatexTn                bool     `json:"enable_latex_tn,omitempty"`
	DisableNewlineStrategy       bool     `json:"disable_newline_strategy,omitempty"`
	SupportedLanguages           []string `json:"supported_languages,omitempty"`
	ContextLanguage              string   `json:"context_language,omitempty"`
	ContextTexts                 []string `json:"context_texts,omitempty"`
	EnableRecoverPuncts          bool     `json:"enable_recover_puncts,omitempty"`
	EosProsody                   int32    `json:"eos_prosody,omitempty"`
	PrependSilenceSeconds        float64  `json:"prepend_silence_seconds,omitempty"`
	MaxParagraphPhonemeSize      int32    `json:"max_paragraph_phoneme_size,omitempty"`
	ParagraphSubSentences        []string `json:"paragraph_sub_sentences,omitempty"`
	MaxLengthToFilterParenthesis int32    `json:"max_length_to_filter_parenthesis,omitempty"`
	EnableLanguageDetector       bool     `json:"enable_language_detector,omitempty"`
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

type VolcEngine struct {
	*BaseEngine
	opt  VolcEngineOption
	conn *websocket.Conn

	streamerMu sync.RWMutex
	streamer   *Streamer // 单 session streamer

	SessionID string

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once

	sessionFinishedCh chan struct{}
	sessionStartedCh  chan struct{}

	recvFirstAudio  bool
	sessionFinished atomic.Bool // 防止重复调用 finishSession
}

// ------------------------ Constructor ------------------------

// NewVolcEngine 使用 VolcEngineOption 创建引擎（传统方式，保持向后兼容）
func NewVolcEngine(opt VolcEngineOption) (*VolcEngine, error) {
	return &VolcEngine{
		BaseEngine:        NewBaseEngine(nil),
		opt:               opt,
		sessionFinishedCh: make(chan struct{}, 1),
		sessionStartedCh:  make(chan struct{}, 1),
	}, nil
}

// NewVolcEngineWithVoice 使用预定义音色创建引擎（推荐方式）
func NewVolcEngineWithVoice(voice VoiceProfile, accessKey, appKey string, opts ...VolcEngineOptionModifier) (*VolcEngine, error) {
	option := voice.ToVolcEngineOption(accessKey, appKey, opts...)
	return NewVolcEngine(option)
}

// NewVolcEngineWithVoiceName 通过音色名称创建引擎（便捷方式）
func NewVolcEngineWithVoiceName(voiceName, accessKey, appKey string, opts ...VolcEngineOptionModifier) (*VolcEngine, error) {
	voice, ok := GetVoice(voiceName)
	if !ok {
		return nil, fmt.Errorf("voice '%s' not found, available voices: %v", voiceName, ListVoices())
	}
	return NewVolcEngineWithVoice(voice, accessKey, appKey, opts...)
}

// ------------------------ Initialization ------------------------

func (e *VolcEngine) Initialize(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)

	ws, r, err := e.dialWebsocket(ctx)
	if err != nil {
		if ws != nil {
			_ = ws.Close()
		}
		return fmt.Errorf("dial: %w, resp: %v", err, r)
	}

	e.conn = ws

	if err := e.startConnection(ws); err != nil {
		return err
	}

	go e.recvLoop()
	return nil
}

func (e *VolcEngine) dialWebsocket(ctx context.Context) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	header.Set("X-Api-App-Key", e.opt.AppKey)
	header.Set("X-Api-Access-Key", e.opt.AccessKey)
	header.Set("X-Api-Resource-Id", e.opt.ResourceID)
	header.Set("X-Api-Connect-Id", uuid.New().String())
	header.Set("X-Control-Require-Usage-Tokens-Return", "*")

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	endpoint := "wss://openspeech.bytedance.com/api/v3/tts/bidirection"
	return websocket.DefaultDialer.DialContext(dialCtx, endpoint, header)
}

// ------------------------ recvLoop ------------------------

func (e *VolcEngine) recvLoop() {

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			msg, err := protocols.ReceiveMessage(e.conn)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					logrus.Info("volcengine: normal ws close")
				} else {
					logrus.Warnf("volcengine: recv error=%v", err)
				}
				return
			}
			logrus.Info("volc: recv message: ", msg.String())
			e.handleMessage(msg)
		}
	}
}

func (e *VolcEngine) handleMessage(msg *protocols.Message) {
	switch {
	case msg.MsgType == protocols.MsgTypeAudioOnlyServer:
		e.handleAudio(msg.Payload)

	case msg.MsgType == protocols.MsgTypeFullServerResponse &&
		msg.EventType == protocols.EventType_TTSSentenceEnd:
		e.handleSentenceEnd(msg.Payload)

	case msg.MsgType == protocols.MsgTypeError:
		logrus.Warnf("volcengine: received error message: %s", msg.String())
	case msg.EventType == protocols.EventType_SessionFinished:
		if e.streamer != nil {
			e.streamer.Close()
		}
		e.sessionFinishedCh <- struct{}{}

	case msg.EventType == protocols.EventType_SessionStarted:
		e.sessionStartedCh <- struct{}{}
	}
}

func (e *VolcEngine) handleAudio(audio []byte) {
	if len(audio) == 0 {
		return
	}
	if !e.recvFirstAudio {
		e.recvFirstAudio = true
	}

	e.streamerMu.RLock()
	s := e.streamer
	e.streamerMu.RUnlock()

	if s != nil {
		s.AppendAudio(audio)
	}
}

// ------------------------ Session Logic ------------------------

func (e *VolcEngine) Start(emotion string) (*Streamer, error) {
	if e.conn == nil {
		return nil, errors.New("connection not initialized")
	}

	streamer := NewStreamer(beep.SampleRate(e.opt.SampleRate), e.opt.Channels)

	e.SessionID = uuid.New().String()
	e.recvFirstAudio = false
	e.resetTimings()
	e.sessionFinished.Store(false)

	if err := e.startSession(e.conn, emotion); err != nil {
		return nil, err
	}

	e.setStreamer(streamer)
	return streamer, nil
}

func (e *VolcEngine) End() error {
	if !e.sessionFinished.CompareAndSwap(false, true) {
		return nil
	}

	if err := e.finishSession(); err != nil {
		return err
	}

	return nil
}

func (e *VolcEngine) Synthesize(text string) error {
	// 获取当前 streamer（线程安全）
	e.streamerMu.RLock()
	streamer := e.streamer
	e.streamerMu.RUnlock()

	if streamer == nil {
		return errors.New("streamer is nil")
	}

	// 重置进度和时间信息，开始新的合成任务
	streamer.ResetProgress()
	e.BaseEngine.ResetTimings() // 清空时间信息
	streamer.SetTimings(nil)    // 清空 streamer 的时间信息

	ttsReq := VolcRequest{
		Event:     int32(protocols.EventType_TaskRequest),
		Namespace: "BidirectionalTTS",
		ReqParams: &VolcReqParams{
			Text: text,
		},
	}

	payload, _ := json.Marshal(&ttsReq)

	// ----------------send task request----------------
	if err := protocols.TaskRequest(e.conn, payload, e.SessionID); err != nil {
		return err
	}
	logrus.Info("volc: send task request: ", string(payload))
	return nil
}

// ------------------------ Streamer helpers ------------------------

func (e *VolcEngine) setStreamer(s *Streamer) {
	e.streamerMu.Lock()
	e.streamer = s
	e.streamerMu.Unlock()
}

func (e *VolcEngine) resetTimings() {
	e.BaseEngine.ResetTimings()
}

// WordTimestamps 实现 Engine 接口，委托给 BaseEngine
func (e *VolcEngine) WordTimestamps() []SentenceTiming {
	return e.BaseEngine.WordTimestamps()
}

func (e *VolcEngine) Close() error {
	var closeErr error

	e.closeOnce.Do(func() {
		e.streamerMu.RLock()
		streamer := e.streamer
		e.streamerMu.RUnlock()
		if streamer != nil {
			streamer.Cancel()
		}
		// 关闭 websocket 连接
		if err := e.closeConnection(); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	})

	if closeErr != nil {
		return fmt.Errorf("volcengine: close error: %w", closeErr)
	}
	return nil
}

func (e *VolcEngine) startConnection(conn *websocket.Conn) error {
	if err := protocols.StartConnection(conn); err != nil {
		return err
	}

	msg, err := protocols.WaitForEvent(e.conn, protocols.MsgTypeFullServerResponse, protocols.EventType_ConnectionStarted)
	if err != nil {
		return fmt.Errorf("volc: wait connection started: %w, %v", err, msg)
	}

	return nil
}

func (e *VolcEngine) closeConnection() error {
	if e.conn != nil {
		_ = protocols.FinishConnection(e.conn)
		if err := e.conn.Close(); err != nil {
			return fmt.Errorf("close websocket: %w", err)
		}
		e.conn = nil
	}
	return nil
}

// startSession 内部方法，用于启动 session
func (e *VolcEngine) startSession(conn *websocket.Conn, emotion string) error {
	audioParams := &VolcReqAudioParams{
		Format:          e.opt.Encoding,
		SampleRate:      int32(e.opt.SampleRate),
		EnableTimestamp: true,
		SpeechRate:      convertSpeechRate(e.opt.SpeedRatio),
	}

	// 如果提供了 emotion，设置到 AudioParams 中
	if emotion != "" {
		audioParams.Emotion = emotion
	}

	startReq := VolcRequest{
		//User: &TTSUser{
		//	UID: uuid.New().String(),
		//},
		Event:     int32(protocols.EventType_StartSession),
		Namespace: "BidirectionalTTS",
		ReqParams: &VolcReqParams{
			Speaker:     e.opt.VoiceType,
			AudioParams: audioParams,
		},
	}
	payload, _ := json.Marshal(&startReq)

	// ----------------start session----------------
	if err := protocols.StartSession(conn, payload, e.SessionID); err != nil {
		return err
	}

	//----------------wait session started----------------
	// 清空之前的通知（如果有）
	select {
	case <-e.sessionStartedCh:
	default:
	}
	// 等待新的 session started 通知
	select {
	case <-e.sessionStartedCh:
	case <-time.After(5 * time.Second):
		return errors.New("volc: session started timeout")
	}
	return nil
}

// finishSession 内部方法，用于结束 session
func (e *VolcEngine) finishSession() error {
	// 清空之前的完成通知（如果有），确保等待的是新的完成通知
	select {
	case <-e.sessionFinishedCh:
	default:
	}

	if err := protocols.FinishSession(e.conn, e.SessionID); err != nil {
		return fmt.Errorf("volc: finish session: %w", err)
	}

	// 等待新的 session finished 通知
	select {
	case <-e.sessionFinishedCh:
	case <-time.After(5 * time.Second):
		return errors.New("volc: session finished timeout")
	}
	return nil
}

func (e *VolcEngine) handleSentenceEnd(payload []byte) {
	var timing SentenceTiming
	if err := json.Unmarshal(payload, &timing); err != nil {
		logrus.Warnf("volc: failed to parse sentence end timing: %v", err)
		return
	}

	// 添加到 BaseEngine 的时间戳列表
	e.BaseEngine.AddTiming(timing)

	// 获取时间戳的副本用于更新 streamer
	timingsCopy := e.BaseEngine.GetTimings()

	// handleSentenceEnd 在 handleMessage 中调用，handleMessage 在 recvLoop 中单线程顺序执行，不需要 streamerMu 锁
	if e.streamer != nil {
		// 更新 streamer 的时间信息
		e.streamer.SetTimings(timingsCopy)

		// 计算总时长并更新 streamer
		if len(timing.Words) > 0 {
			lastWord := timing.Words[len(timing.Words)-1]
			totalDuration := lastWord.EndTime
			e.streamer.SetTotalDuration(totalDuration)
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
