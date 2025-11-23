package tts

import (
	"ava/internal/protocols"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
	opt      VolcEngineOption
	conn     *websocket.Conn
	streamer *Streamer

	SessionID string

	stopped        atomic.Bool
	closeOnce      sync.Once
	cancel         context.CancelFunc
	ctx            context.Context
	recvLoopStopCh chan struct{}
	recvFirstAudio bool
}

func NewVolcEngine(opt VolcEngineOption, s *Streamer) (*VolcEngine, error) {
	if s == nil {
		return nil, errors.New("streamer is nil")
	}
	return &VolcEngine{
		BaseEngine:     NewBaseEngine(nil),
		opt:            opt,
		streamer:       s,
		recvLoopStopCh: make(chan struct{}),
	}, nil
}

func (e *VolcEngine) Initialize(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.SessionID = uuid.New().String()
	// ---------------- Dial 超时 ----------------
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// websocket 鉴权
	header := http.Header{}
	header.Set("X-Api-App-Key", e.opt.AppKey)
	header.Set("X-Api-Access-Key", e.opt.AccessKey)
	header.Set("X-Api-Resource-Id", e.opt.ResourceID)
	header.Set("X-Api-Connect-Id", e.SessionID)
	header.Set("X-Control-Require-Usage-Tokens-Return", "*")
	endpoint := "wss://openspeech.bytedance.com/api/v3/tts/bidirection"
	// ----------------dial server----------------
	conn, r, err := websocket.DefaultDialer.DialContext(dialCtx, endpoint, header)
	if err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return fmt.Errorf("dial: %w, resp: %v", err, r)
	}
	e.conn = conn

	if err := e.startConnection(e.conn); err != nil {
		return err
	}

	if err := e.startSession(e.conn); err != nil {
		return err
	}
	go e.recvLoop()
	return nil
}

func (e *VolcEngine) recvLoop() {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Recovered from panic in recvLoop: %v", r)
		}
		close(e.recvLoopStopCh)
	}()
Loop:
	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			msg, err := protocols.ReceiveMessage(e.conn)
			logrus.Info("volcengine: recv message: ", msg.String())
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) || strings.Contains(err.Error(), "use of closed network connection") {
					logrus.Info("volcengine: recv message error: ", err)
				} else {
					logrus.Error("volcengine: recv message error: ", err)
				}

				break Loop
			}
			switch {
			case msg.MsgType == protocols.MsgTypeAudioOnlyServer:
				if len(msg.Payload) > 0 {
					if !e.recvFirstAudio {
						e.recvFirstAudio = true
					}
					e.streamer.AppendAudio(msg.Payload)

				}
			case msg.EventType == protocols.EventType_SessionFinished:
				e.streamer.EOS = true
				break Loop
			default:
				//todo log error message
			}
		}
	}
}

func (e *VolcEngine) Synthesize(text string) error {

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
	logrus.Info("volcengine: send task request: ", string(payload))
	return nil
}

func (e *VolcEngine) Stop() error {
	return e.finishSession()
}

func (e *VolcEngine) Close() error {
	var closeErr error

	e.closeOnce.Do(func() {
		if err := e.finishSession(); err != nil {
			closeErr = err
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
		return fmt.Errorf("wait connection started: %w, %v", err, msg)
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

// 一次语音合成：start session -> wait session started -> send task request -> Finish session
func (e *VolcEngine) startSession(conn *websocket.Conn) error {
	startReq := VolcRequest{
		//User: &TTSUser{
		//	UID: uuid.New().String(),
		//},
		Event:     int32(protocols.EventType_StartSession),
		Namespace: "BidirectionalTTS",
		ReqParams: &VolcReqParams{
			Speaker: e.opt.VoiceType,
			AudioParams: &VolcReqAudioParams{
				Format:          e.opt.Encoding,
				SampleRate:      int32(e.opt.SampleRate),
				EnableTimestamp: true,
				SpeechRate:      convertSpeechRate(e.opt.SpeedRatio),
			},
		},
	}
	payload, _ := json.Marshal(&startReq)

	// ----------------start session----------------
	if err := protocols.StartSession(conn, payload, e.SessionID); err != nil {
		return err
	}

	//----------------wait session started----------------
	_, err := protocols.WaitForEvent(conn, protocols.MsgTypeFullServerResponse, protocols.EventType_SessionStarted)
	if err != nil {
		return err
	}

	return nil
}

// finish session to exit recvloop
func (e *VolcEngine) finishSession() error {
	if e.stopped.Swap(true) {
		return nil
	}

	if err := protocols.FinishSession(e.conn, e.SessionID); err != nil {
		return fmt.Errorf("finish session: %w", err)
	}
	// if _, err := protocols.WaitForEvent(e.conn, protocols.MsgTypeFullServerResponse, protocols.EventType_SessionFinished); err != nil {
	// 	return fmt.Errorf("wait session finished: %w", err)
	// }

	return nil
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
