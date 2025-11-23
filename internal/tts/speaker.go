package tts

import (
	"time"

	"github.com/gopxl/beep/speaker"
)

type Speaker struct {
	tts      Engine
	streamer *Streamer
}

func NewSpeaker(tts Engine) *Speaker {
	return &Speaker{
		tts: tts,
	}
}

func (s *Speaker) Say(text string) error {
	err := s.tts.Synthesize(text)
	if err != nil {
		return err
	}
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
	s.tts.Stop()
	s.streamer.EOS = true
	speaker.Unlock()
}
