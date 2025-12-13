package main

import (
	"ava/internal/tts"
	"ava/internal/tts/volc"
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	ctx := context.Background()

	// 方式1：使用预定义音色（推荐）
	accessKey := "n1uNFm540_2oItTs0UsULkWWvuzQiXbD"
	appKey := "5711022755"

	codec := volc.DefaultCodecConfig()
	codec.SpeedRatio = 1.1 // 自定义语速

	ttsEngine, err := volc.NewVolcEngine(
		ctx,
		volc.AuthConfig{
			AccessKey: accessKey,
			AppKey:    appKey,
		},
		volc.NewVoiceConfig(&volc.VoiceTiaoPigongzhu),
		codec,
	)
	if err != nil {
		log.Fatalf("Failed to create tts engine: %v", err)
	}
	defer ttsEngine.Close() // 确保资源清理

	// 方式2：使用音色名称（便捷方式）
	// voiceConfig, err := volc.NewVoiceConfigByName("meilin_nvyou")
	// if err != nil {
	// 	log.Fatalf("Failed to get voice config: %v", err)
	// }
	// codec := volc.DefaultCodecConfig()
	// codec.SpeedRatio = 1.1
	// ttsEngine, err := volc.NewVolcEngine(
	// 	ctx,
	// 	volc.AuthConfig{
	// 		AccessKey: accessKey,
	// 		AppKey:    appKey,
	// 	},
	// 	voiceConfig,
	// 	codec,
	// )
	// if err != nil {
	// 	log.Fatalf("Failed to create tts engine: %v", err)
	// }

	// 方式3：使用默认 codec（不传 codec 参数）
	// ttsEngine, err := volc.NewVolcEngine(
	// 	ctx,
	// 	volc.AuthConfig{
	// 		AccessKey: accessKey,
	// 		AppKey:    appKey,
	// 	},
	// 	volc.NewVoiceConfig(&volc.VoiceTiaoPigongzhu),
	// )
	// if err != nil {
	// 	log.Fatalf("Failed to create tts engine: %v", err)
	// }

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
	//第一个调用：start=true 表示开始新 session，end=false 表示不结束 session
	if err := speaker.Say(tts.SayRequest{
		Text:    "欢迎来到美丽新世界!",
		Start:   true,
		End:     false,
		Emotion: "happy",
	}); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	// 第二个调用：start=false 表示继续使用当前 session，end=true 表示结束 session
	// 注意：这会等待 session 真正完成后才返回
	if err := speaker.Say(tts.SayRequest{
		Text:    "让我们一起跳舞吧!",
		Start:   false,
		End:     true,
		Emotion: "",
	}); err != nil {
		log.Fatalf("Failed to synthesize: %v", err)
	}

	// 第三个调用：start=true 表示开始新 session（因为上一个已经结束），end=true 表示结束 session
	if err := speaker.Say(tts.SayRequest{
		Text:    "大傻春，你要干嘛!",
		Start:   true,
		End:     true,
		Emotion: "angry",
	}); err != nil {
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
