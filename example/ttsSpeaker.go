package main

import (
	"ava/internal/tts"
	"context"
	"log"

	"github.com/gopxl/beep"
)

func main() {

	ttsStreamer := tts.NewStreamer(beep.SampleRate(16000), 1)

	ttsOpt := tts.VolcEngineOption{
		VoiceType:  "zh-CN-XiaoxiaoNeural",
		ResourceID: "1234567890",
		AccessKey:  "1234567890",
		AppKey:     "1234567890",
		Encoding:   "pcm",
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
	}

	ttsEngine, err := tts.NewVolcEngine(ttsOpt, ttsStreamer)
	if err != nil {
		log.Fatalf("Failed to create tts engine: %v", err)
	}

	// 初始化引擎
	ctx := context.Background()
	if err := ttsEngine.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize tts engine: %v", err)
	}

	// 创建 Speaker
	speaker := tts.NewSpeaker(ttsEngine)
	speaker.Play(ttsStreamer)

	// 合成并播放语音
	if err := speaker.Say("Hello, world!"); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	// 停止播放
	speaker.Stop()
}
