package main

import (
	"ava/internal/tts"
	"context"
	"log"
	"time"

	"github.com/gopxl/beep"
)

func main() {

	ttsStreamer := tts.NewStreamer(beep.SampleRate(16000), 1)

	ttsOpt := tts.VolcEngineOption{
		VoiceType:  "zh_male_lengkugege_emo_v2_mars_bigtts",
		ResourceID: "seed-tts-1.0",
		AccessKey:  "n1uNFm540_2oItTs0UsULkWWvuzQiXbD",
		AppKey:     "5711022755",
		Encoding:   "pcm",
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
		SpeedRatio: 1.1,
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
	if err := speaker.Say("欢迎来到美丽新世界!", false); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	if err := speaker.Say("让我们一起跳舞吧!", true); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	// 等待3秒后停止播放
	<-time.After(3 * time.Second)
	speaker.Stop()
	select {}
}
