package main

import (
	"ava/internal/tts"
	"context"
	"fmt"
	"log"
	"time"
)

func main() {

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

	ttsEngine, err := tts.NewVolcEngine(ttsOpt)
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

	// 创建 context 用于控制进度打印 goroutine
	progressCtx, cancelProgress := context.WithCancel(ctx)
	defer cancelProgress()

	// 启动进度打印 goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-progressCtx.Done():
				return
			case <-ticker.C:
				progress := speaker.GetProgress()
				if progress.TotalTime > 0 {
					fmt.Printf("[进度] %.2f/%.2f 秒 (%.1f%%)",
						progress.CurrentTime,
						progress.TotalTime,
						progress.Percentage)
					if progress.CurrentWord != nil {
						fmt.Printf(" - 当前词: %s", progress.CurrentWord.Word)
					}
					if progress.PlayedText != "" {
						fmt.Printf(" - 已播放: %s", progress.PlayedText)
					}
					fmt.Println()
				} else if progress.CurrentTime > 0 {
					fmt.Printf("[进度] %.2f 秒 (总时长未知)", progress.CurrentTime)
					if progress.PlayedText != "" {
						fmt.Printf(" - 已播放: %s", progress.PlayedText)
					}
					fmt.Println()
				}
			}
		}
	}()

	// 合成并播放语音
	// 第一个调用：start=true 表示开始新 session，end=false 表示不结束 session
	if err := speaker.Say("欢迎来到美丽新世界!", true, false); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	// 第二个调用：start=false 表示继续使用当前 session，end=true 表示结束 session
	// 注意：这会等待 session 真正完成后才返回
	if err := speaker.Say("让我们一起跳舞吧!", false, true); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	// 第三个调用：start=true 表示开始新 session（因为上一个已经结束），end=true 表示结束 session
	if err := speaker.Say("大傻春，你要干嘛!", true, true); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	// 等待播放完成或3秒后停止播放
	<-time.After(5 * time.Second)
	speaker.Stop()

	// 停止进度打印
	cancelProgress()

	// 打印最终进度
	finalProgress := speaker.GetProgress()
	fmt.Printf("\n[最终进度] %.2f/%.2f 秒 (%.1f%%)\n",
		finalProgress.CurrentTime,
		finalProgress.TotalTime,
		finalProgress.Percentage)
	if finalProgress.PlayedText != "" {
		fmt.Printf("[已播放文本] %s\n", finalProgress.PlayedText)
	}

	select {}
}
