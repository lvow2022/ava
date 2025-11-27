package tts

import (
	"strings"
	"testing"
)

func TestTagParser(t *testing.T) {
	type recorder struct {
		startAttrs map[string]string
		middleFeed []string
		endCalled  bool
		endCount   int
	}

	tests := []struct {
		name     string
		feeds    []string
		register func(p *TagParser, r *recorder)
		verify   func(t *testing.T, r *recorder)
	}{
		{
			name:  "streaming callbacks",
			feeds: []string{`<say tone="happy"`, ` mood="calm">你好`, `，世界</say>`},
			register: func(p *TagParser, r *recorder) {
				p.RegisterTag("say", TagCallbacks{
					OnStart: func(attrs map[string]string) {
						r.startAttrs = attrs
					},
					OnMiddle: func(text string) {
						r.middleFeed = append(r.middleFeed, text)
					},
					OnEnd: func() {
						r.endCalled = true
					},
				})
			},
			verify: func(t *testing.T, r *recorder) {
				wantAttrs := map[string]string{"tone": "happy", "mood": "calm"}
				if !mapsEqual(r.startAttrs, wantAttrs) {
					t.Fatalf("unexpected attrs, got=%v want=%v", r.startAttrs, wantAttrs)
				}

				gotMiddle := concatChunks(r.middleFeed)
				if gotMiddle != "你好，世界" {
					t.Fatalf("unexpected middle text, got=%q want=%q", gotMiddle, "你好，世界")
				}

				if !r.endCalled {
					t.Fatalf("expected OnEnd to be called")
				}
			},
		},
		{
			name:  "unknown tag ignored",
			feeds: []string{`<foo attr="x">bar</foo>`, `<say>ok</say>`},
			register: func(p *TagParser, r *recorder) {
				p.RegisterTag("say", TagCallbacks{
					OnMiddle: func(text string) {
						r.middleFeed = append(r.middleFeed, text)
					},
					OnEnd: func() {
						r.endCount++
					},
				})
			},
			verify: func(t *testing.T, r *recorder) {
				if len(r.middleFeed) != 1 || r.middleFeed[0] != "ok" {
					t.Fatalf("unexpected middle feed %+v", r.middleFeed)
				}
				if r.endCount != 1 {
					t.Fatalf("expected OnEnd to run once, got %d", r.endCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			parser := NewTagParser()

			tt.register(parser, rec)

			for _, chunk := range tt.feeds {
				parser.Feed(chunk)
			}

			tt.verify(t, rec)
		})
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func concatChunks(chunks []string) string {
	var builder strings.Builder
	for _, chunk := range chunks {
		builder.WriteString(chunk)
	}
	return builder.String()
}
