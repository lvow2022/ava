package tts

import "fmt"

type TagAwareSpeaker struct {
	speaker *Speaker
	parser  *TagParser
}

func NewTagAwareSpeaker(s *Speaker) *TagAwareSpeaker {
	tas := &TagAwareSpeaker{
		speaker: s,
		parser:  NewTagParser(),
	}

	// say 标签
	tas.parser.RegisterTag("say", TagCallbacks{
		OnStart: func(attrs map[string]string) {
			fmt.Println("[say] 开始播放，属性:", attrs)
		},
		OnMiddle: func(text string) {
			if err := s.Say(text, false); err != nil {
				fmt.Println("Say 错误:", err)
			}
		},
		OnEnd: func() {
			if err := s.Say("", true); err != nil {
				fmt.Println("结束 Say 错误:", err)
			}
			fmt.Println("[say] 播放结束")
		},
	})

	// pause 标签
	tas.parser.RegisterTag("pause", TagCallbacks{
		OnStart: func(attrs map[string]string) {
			reason := attrs["reason"]
			fmt.Println("[pause] 暂停播放, 原因:", reason)
			s.Pause()
		},
	})

	// stop 标签
	tas.parser.RegisterTag("stop", TagCallbacks{
		OnStart: func(attrs map[string]string) {
			reason := attrs["reason"]
			fmt.Println("[stop] 停止播放, 原因:", reason)
			s.Stop()
		},
	})

	// resume 标签
	tas.parser.RegisterTag("resume", TagCallbacks{
		OnStart: func(attrs map[string]string) {
			reason := attrs["reason"]
			fmt.Println("[resume] 恢复播放, 原因:", reason)
			s.Resume()
		},
	})

	return tas
}

// Feed 输入 LLM 输出 XML
func (tas *TagAwareSpeaker) Feed(xmlChunk string) {
	tas.parser.Feed(xmlChunk)
}
