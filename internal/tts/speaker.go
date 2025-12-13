package tts

import (
	"fmt"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/sirupsen/logrus"
)

// Progress 表示播放进度信息
type Progress struct {
	CurrentTime float64     // 当前播放时间（秒）
	TotalTime   float64     // 总时长（秒）
	CurrentWord *WordTiming // 当前正在播放的词（如果有）
	PlayedText  string      // 已播放的文本（所有已播放完成的词拼接）
	Percentage  float64     // 播放进度百分比 (0-100)
}

// SayRequest 表示 Say 方法的请求参数
type SayRequest struct {
	Text    string // 要合成的文本内容
	Start   bool   // 是否启动新 session
	End     bool   // 是否结束 session
	Emotion string // 情感设置（可选），如：happy, sad, angry 等
	// 未来可以扩展更多参数，如：Speed, Pitch, Volume 等
}

type Speaker struct {
	tts         Engine
	streamQueue *StreamQueue
}

func NewSpeaker(tts Engine) *Speaker {
	s := &Speaker{
		tts:         tts,
		streamQueue: NewStreamQueue(),
	}

	// 初始化 speaker
	// 使用默认采样率，Engine 实现应该在创建 Streamer 时设置正确的采样率
	sampleRate := beep.SampleRate(16000)

	// 初始化 beep speaker
	speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	speaker.Play(s.streamQueue)

	return s
}

// Say 使用 SayRequest 进行语音合成和播放
func (s *Speaker) Say(req SayRequest) error {
	var streamer *Streamer
	var err error

	if req.Start {
		streamer, err = s.tts.Start(req.Emotion)
		if err != nil {
			return fmt.Errorf("start session failed: %w", err)
		}
		s.streamQueue.Push(streamer)
	}

	// 只有当 Text 不为空时才调用 Synthesize
	if req.Text != "" {
		err = s.tts.Synthesize(req.Text)
		if err != nil {
			return fmt.Errorf("synthesize failed: %w", err)
		}
	}

	if req.End {
		if err := s.tts.End(); err != nil {
			logrus.Warnf("speaker: failed to finish session: %v", err)
		}
	}

	return nil
}

func (s *Speaker) Play(streamer *Streamer) {
	s.streamQueue.Push(streamer)
}

// func (s *Speaker) Pause() {
// 	speaker.Lock()
// 	speaker.Suspend()
// 	speaker.Unlock()
// }

// func (s *Speaker) Resume() {
// 	speaker.Lock()
// 	speaker.Resume()
// 	speaker.Unlock()
// }

// 停止播放当前streamer
func (s *Speaker) Stop() {
	speaker.Lock()
	s.streamQueue.StopCurrent()
	speaker.Unlock()

	// 结束当前的 TTS session，确保下次 Say() 时能正常开始新 session
	if err := s.tts.End(); err != nil {
		logrus.Warnf("speaker: failed to finish session after stop: %v", err)
	}
}

// GetProgress 获取当前播放进度
func (s *Speaker) GetProgress() *Progress {
	// 从 StreamQueue 获取当前正在播放的 streamer
	currentStreamer := s.streamQueue.CurrentStreamer()
	if currentStreamer == nil {
		return &Progress{}
	}

	currentTime, totalTime := currentStreamer.GetProgress()
	progress := &Progress{
		CurrentTime: currentTime,
		TotalTime:   totalTime,
	}

	// 计算百分比
	if totalTime > 0 {
		progress.Percentage = (currentTime / totalTime) * 100
		if progress.Percentage > 100 {
			progress.Percentage = 100
		}
	}

	// 从 streamer 获取时间戳并计算当前词
	timingList := currentStreamer.GetTimings()
	progress.CurrentWord = GetCurrentWordFromTimings(timingList, currentTime)

	// 从 streamer 获取已播放文本
	progress.PlayedText = currentStreamer.GetPlayedText(currentTime)

	return progress
}

// GetTimings 获取所有句子的时间信息
func (s *Speaker) GetTimings() []SentenceTiming {
	currentStreamer := s.streamQueue.CurrentStreamer()
	if currentStreamer == nil {
		return nil
	}
	return currentStreamer.GetTimings()
}
