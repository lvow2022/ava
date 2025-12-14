package tts

import "fmt"

type TagAwareSpeaker struct {
	speaker      *Speaker
	parser       *TagParser
	currentContext []string // 保存当前 say 标签的 context
}

func NewTagAwareSpeaker(s *Speaker) *TagAwareSpeaker {
	tas := &TagAwareSpeaker{
		speaker: s,
		parser:  NewTagParser(),
	}

	// say 标签
	tas.parser.RegisterTag("say", TagCallbacks{
		OnStart: func(attrs map[string]string) {
			emotion := attrs["emotion"]     // 从属性中获取 emotion（可选，推荐使用 context）
			context := attrs["context"]     // 从属性中获取 context（推荐使用）
			var contextTexts []string
			if context != "" {
				contextTexts = []string{context}
				tas.currentContext = contextTexts // 保存 context 供 OnMiddle 使用
			} else {
				tas.currentContext = nil
			}
			if err := s.Say(SayRequest{
				Text:         "",
				Start:        true,
				End:          false,
				Emotion:      emotion,
				ContextTexts: contextTexts,
			}); err != nil {
				fmt.Println("开始 Say 错误:", err)
			}
			fmt.Println("[say] 开始播放，属性:", attrs)
		},
		OnMiddle: func(text string) {
			// 使用保存的 context
			if err := s.Say(SayRequest{
				Text:         text,
				Start:       false,
				End:         false,
				Emotion:     "",
				ContextTexts: tas.currentContext,
			}); err != nil {
				fmt.Println("Say 错误:", err)
			}
		},
		OnEnd: func() {
			if err := s.Say(SayRequest{
				Text:         "",
				Start:       false,
				End:         true,
				Emotion:     "",
				ContextTexts: nil,
			}); err != nil {
				fmt.Println("结束 Say 错误:", err)
			}
			tas.currentContext = nil // 清除 context
			fmt.Println("[say] 播放结束")
		},
	})

	// pause 标签
	// tas.parser.RegisterTag("pause", TagCallbacks{
	// 	OnStart: func(attrs map[string]string) {
	// 		reason := attrs["reason"]
	// 		fmt.Println("[pause] 暂停播放, 原因:", reason)
	// 		s.Pause()
	// 	},
	// })

	// stop 标签
	tas.parser.RegisterTag("stop", TagCallbacks{
		OnStart: func(attrs map[string]string) {
			reason := attrs["reason"]
			fmt.Println("[stop] 停止播放, 原因:", reason)
			s.Stop()
		},
	})

	// resume 标签
	// tas.parser.RegisterTag("resume", TagCallbacks{
	// 	OnStart: func(attrs map[string]string) {
	// 		reason := attrs["reason"]
	// 		fmt.Println("[resume] 恢复播放, 原因:", reason)
	// 		s.Resume()
	// 	},
	// })

	return tas
}

// Feed 输入 LLM 输出 XML
func (tas *TagAwareSpeaker) Feed(xmlChunk string) {
	tas.parser.Feed(xmlChunk)
}
