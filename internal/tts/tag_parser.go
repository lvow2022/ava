package tts

import (
	"regexp"
	"strings"
)

type TagCallbacks struct {
	OnStart  func(attrs map[string]string)
	OnMiddle func(text string)
	OnEnd    func()
}

type registeredTag struct {
	name string
	cb   TagCallbacks
}

type TagParser struct {
	tags        map[string]registeredTag
	buffer      string
	activeTag   *registeredTag
	activeAttrs map[string]string
}

func NewTagParser() *TagParser {
	return &TagParser{
		tags: make(map[string]registeredTag),
	}
}

func (p *TagParser) RegisterTag(name string, cb TagCallbacks) {
	p.tags[name] = registeredTag{name: name, cb: cb}
}

var startTagRe = regexp.MustCompile(`^<([a-zA-Z0-9]+)([^>]*)>`)
var endTagRe = regexp.MustCompile(`^</([a-zA-Z0-9]+)>`)
var attrRe = regexp.MustCompile(`([a-zA-Z0-9_-]+)="([^"]*)"`)

func parseAttrs(s string) map[string]string {
	m := make(map[string]string)
	all := attrRe.FindAllStringSubmatch(s, -1)
	for _, a := range all {
		m[a[1]] = a[2]
	}
	return m
}

func (p *TagParser) Feed(chunk string) {
	p.buffer += chunk
	p.parse()
}

func (p *TagParser) parse() {
	for {
		if p.activeTag == nil {
			// 查找下一个 '<'
			i := strings.Index(p.buffer, "<")
			if i == -1 {
				return
			}

			// 抛弃外面的纯文本
			if i > 0 {
				p.buffer = p.buffer[i:]
			}

			// 尝试匹配开始标签
			if m := startTagRe.FindStringSubmatch(p.buffer); m != nil {
				name := m[1]
				attrs := parseAttrs(m[2])

				tag, ok := p.tags[name]
				if ok {
					p.activeTag = &tag
					p.activeAttrs = attrs
					if tag.cb.OnStart != nil {
						tag.cb.OnStart(attrs)
					}
					// 丢掉开始标签
					p.buffer = p.buffer[len(m[0]):]
					continue
				} else {
					// 未注册的标签，跳过整个标签（包括开始标签、内容和结束标签）
					// 先移除开始标签
					p.buffer = p.buffer[len(m[0]):]
					// 查找对应的结束标签
					endPattern := "</" + name + ">"
					endIdx := strings.Index(p.buffer, endPattern)
					if endIdx != -1 {
						// 找到结束标签，跳过整个标签内容
						p.buffer = p.buffer[endIdx+len(endPattern):]
						continue
					}
					// 如果还没找到结束标签，等待更多数据
					return
				}
			}
			return
		}

		// 活动标签模式（寻找正文或结束标签）
		endIdx := strings.Index(p.buffer, "</")

		if endIdx == -1 {
			// 全是正文 → 流式输出
			if p.activeTag.cb.OnMiddle != nil {
				p.activeTag.cb.OnMiddle(p.buffer)
			}
			p.buffer = ""
			return
		}

		// 中间正文
		if endIdx > 0 {
			text := p.buffer[:endIdx]
			if p.activeTag.cb.OnMiddle != nil {
				p.activeTag.cb.OnMiddle(text)
			}
			p.buffer = p.buffer[endIdx:]
		}

		// 匹配结束标签
		if m := endTagRe.FindStringSubmatch(p.buffer); m != nil {
			endName := m[1]

			if p.activeTag.name == endName {
				if p.activeTag.cb.OnEnd != nil {
					p.activeTag.cb.OnEnd()
				}
				p.activeTag = nil
				p.activeAttrs = nil
				p.buffer = p.buffer[len(m[0]):]
				continue
			}
		}

		return
	}
}
