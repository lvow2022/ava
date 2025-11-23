package main

import (
	"ava/internal/tts"
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// SayTextTool 合成并播放文本的工具
type SayTextTool struct {
	manager *tts.ToolSpeakerManager
}

func NewSayTextTool(manager *tts.ToolSpeakerManager) *SayTextTool {
	return &SayTextTool{
		manager: manager,
	}
}

func (st *SayTextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "say_text",
		Desc: "Synthesize and play text using TTS. Use this tool when you need to speak text to the user. For streaming output, call with end=false for each chunk, and end=true for the final chunk.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"text": {
				Desc:     "The text to synthesize and speak",
				Type:     schema.String,
				Required: true,
			},
			"end": {
				Desc:     "Whether this is the final chunk of text (true) or more text will follow (false). Set to false for streaming chunks, true for the final chunk.",
				Type:     schema.Boolean,
				Required: true,
			},
		}),
	}, nil
}

func (st *SayTextTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Text string `json:"text"`
		End  bool   `json:"end"`
	}

	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Text == "" {
		return "", fmt.Errorf("text parameter is required and cannot be empty")
	}

	speaker, err := st.manager.GetSpeaker(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get speaker: %w", err)
	}

	if err := speaker.Say(args.Text, args.End); err != nil {
		return "", fmt.Errorf("failed to synthesize text: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"text":    args.Text,
		"end":     args.End,
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// PauseSpeechTool 暂停播放的工具
type PauseSpeechTool struct {
	manager *tts.ToolSpeakerManager
}

func NewPauseSpeechTool(manager *tts.ToolSpeakerManager) *PauseSpeechTool {
	return &PauseSpeechTool{
		manager: manager,
	}
}

func (pt *PauseSpeechTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "pause_speech",
		Desc:        "Pause the current speech playback",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (pt *PauseSpeechTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	speaker, err := pt.manager.GetSpeaker(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get speaker: %w", err)
	}

	speaker.Pause()

	result := map[string]interface{}{
		"success": true,
		"action":  "paused",
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// ResumeSpeechTool 恢复播放的工具
type ResumeSpeechTool struct {
	manager *tts.ToolSpeakerManager
}

func NewResumeSpeechTool(manager *tts.ToolSpeakerManager) *ResumeSpeechTool {
	return &ResumeSpeechTool{
		manager: manager,
	}
}

func (rt *ResumeSpeechTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "resume_speech",
		Desc:        "Resume the paused speech playback",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (rt *ResumeSpeechTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	speaker, err := rt.manager.GetSpeaker(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get speaker: %w", err)
	}

	speaker.Resume()

	result := map[string]interface{}{
		"success": true,
		"action":  "resumed",
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// StopSpeechTool 停止播放的工具
type StopSpeechTool struct {
	manager *tts.ToolSpeakerManager
}

func NewStopSpeechTool(manager *tts.ToolSpeakerManager) *StopSpeechTool {
	return &StopSpeechTool{
		manager: manager,
	}
}

func (st *StopSpeechTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "stop_speech",
		Desc:        "Stop the current speech playback immediately",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (st *StopSpeechTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	speaker, err := st.manager.GetSpeaker(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get speaker: %w", err)
	}

	speaker.Stop()

	result := map[string]interface{}{
		"success": true,
		"action":  "stopped",
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

