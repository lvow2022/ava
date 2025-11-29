package tts

import (
	"time"

	"github.com/gopxl/beep/speaker"
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
	tts      Engine
	streamer *Streamer
}

func NewSpeaker(tts Engine) *Speaker {
	return &Speaker{
		tts: tts,
	}
}

func (s *Speaker) Say(text string, end bool) error {
	err := s.tts.Synthesize(text)
	if err != nil {
		return err
	}

	//if end {
	//return s.tts.Stop()
	//}

	return nil
}

func (s *Speaker) Play(streamer *Streamer) {
	s.streamer = streamer
	speaker.Init(streamer.format.SampleRate, streamer.format.SampleRate.N(time.Second/10))
	speaker.Play(streamer)
}

func (s *Speaker) Pause() {
	speaker.Lock()
	s.streamer.Paused = true
	speaker.Unlock()
}

func (s *Speaker) Resume() {
	speaker.Lock()
	s.streamer.Paused = false
	speaker.Unlock()
}

func (s *Speaker) Stop() {
	speaker.Lock()
	s.streamer.ClearBuffer() // 清空缓冲区，但流可以继续使用
	// s.tts.Stop()
	speaker.Unlock()
}

// GetProgress 获取当前播放进度
func (s *Speaker) GetProgress() *Progress {
	if s.streamer == nil {
		return &Progress{}
	}

	currentTime, totalTime := s.streamer.GetProgress()
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
	if s.streamer != nil {
		progress.PlayedText = s.streamer.GetPlayedText(currentTime)
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
