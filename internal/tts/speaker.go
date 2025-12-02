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
	// 从 Engine 获取 format 信息（目前支持 VolcEngine）
	var sampleRate beep.SampleRate

	if volcEngine, ok := tts.(*VolcEngine); ok {
		sampleRate = beep.SampleRate(volcEngine.opt.SampleRate)
	} else {
		// 默认值，如果 Engine 未初始化或不是 VolcEngine
		sampleRate = beep.SampleRate(16000)
	}

	// 初始化 beep speaker
	speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	speaker.Play(s.streamQueue)

	return s
}

func (s *Speaker) Say(text string, start, end bool) error {
	var streamer *Streamer
	var err error

	// 1. 启动新 session，获取新的 streamer
	//    StartSession() 内部会结束旧 session（如果有）
	if start {
		streamer, err = s.tts.StartSession()
		if err != nil {
			return fmt.Errorf("start session failed: %w", err)
		}
		s.streamQueue.Push(streamer)
	}

	// 2. 合成文本（使用当前 session）
	err = s.tts.Synthesize(text)
	if err != nil {
		return fmt.Errorf("synthesize failed: %w", err)
	}

	// 3. 如果 end 为 true，结束 session
	if end {
		if err := s.tts.FinishSession(); err != nil {
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
	if err := s.tts.FinishSession(); err != nil {
		logrus.Warnf("speaker: failed to finish session after stop: %v", err)
	}
}

// GetProgress 获取当前播放进度
func (s *Speaker) GetProgress() *Progress {
	// 从 StreamQueue 获取当前正在播放的 streamer
	currentStreamer := s.streamQueue.GetCurrentStreamer()
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

	// 如果 Engine 是 VolcEngine，尝试获取当前词
	if volcEngine, ok := s.tts.(*VolcEngine); ok {
		progress.CurrentWord = volcEngine.GetCurrentWord(currentTime)
	}

	// 从 streamer 获取已播放文本
	if currentStreamer != nil {
		progress.PlayedText = currentStreamer.GetPlayedText(currentTime)
	}

	return progress
}

// GetTimings 获取所有句子的时间信息（仅对 VolcEngine 有效）
func (s *Speaker) GetTimings() []SentenceTiming {
	if volcEngine, ok := s.tts.(*VolcEngine); ok {
		return volcEngine.GetTimings()
	}
	return nil
}
