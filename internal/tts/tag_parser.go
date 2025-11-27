package tts

import (
	"encoding/xml"
	"io"
	"strings"
)

// TagCallbacks 定义标签的回调
type TagCallbacks struct {
	OnStart  func(attrs map[string]string)
	OnMiddle func(text string)
	OnEnd    func()
}

// TagParser 解析器
type TagParser struct {
	tags map[string]TagCallbacks
}

// NewTagParser 创建解析器
func NewTagParser() *TagParser {
	return &TagParser{
		tags: make(map[string]TagCallbacks),
	}
}

// RegisterTag 注册一个标签及其回调
func (p *TagParser) RegisterTag(name string, cb TagCallbacks) {
	p.tags[name] = cb
}

// Feed 解析 XML 数据（可流式调用多次）
func (p *TagParser) Feed(data string) error {
	decoder := xml.NewDecoder(strings.NewReader(data))

	var activeTag string
	var attrs map[string]string

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			cb, exists := p.tags[name]
			if exists {
				activeTag = name
				attrs = make(map[string]string)
				for _, a := range t.Attr {
					attrs[a.Name.Local] = a.Value
				}
				if cb.OnStart != nil {
					cb.OnStart(attrs)
				}
			} else {
				activeTag = ""
			}
		case xml.EndElement:
			name := t.Name.Local
			if cb, exists := p.tags[name]; exists && name == activeTag {
				if cb.OnEnd != nil {
					cb.OnEnd()
				}
				activeTag = ""
			}
		case xml.CharData:
			if activeTag != "" {
				if cb, exists := p.tags[activeTag]; exists && cb.OnMiddle != nil {
					cb.OnMiddle(string(t))
				}
			}
		}
	}
}
