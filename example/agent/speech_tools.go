package main

import (
	"ava/internal/tts/tools"
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// HandleUserInputTool 处理用户输入的工具，查询播放进度并返回决策建议
type HandleUserInputTool struct {
	manager *tools.ToolSpeakerManager
}

func NewHandleUserInputTool(manager *tools.ToolSpeakerManager) *HandleUserInputTool {
	return &HandleUserInputTool{
		manager: manager,
	}
}

func (ht *HandleUserInputTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "fetch_playback_progress",
		Desc: `Query current playback progress information. 
This tool should be called when you need to check the current playback status before responding to user input.
It returns:
- Current playback time and total duration
- Playback percentage
- Currently playing word (if available)
- Already played text (all words that have finished playing)
You should use this information to decide whether to interrupt the current playback, wait for it to finish, or respond immediately.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"user_input": {
				Desc:     "The user's input text (for context, but not used in the tool logic)",
				Type:     schema.String,
				Required: false,
			},
		}),
	}, nil
}

func (ht *HandleUserInputTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		UserInput string `json:"user_input"`
	}

	_ = json.Unmarshal([]byte(argumentsInJSON), &args)

	speaker, err := ht.manager.GetSpeaker(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get speaker: %w", err)
	}

	// 查询当前播放进度
	progress := speaker.GetProgress()

	// 计算剩余时间
	remainingTime := progress.TotalTime - progress.CurrentTime
	isPlaying := progress.CurrentTime > 0 && progress.Percentage < 100

	// 构建进度信息（只返回数据，不包含决策）
	progressInfo := map[string]interface{}{
		"current_time":   progress.CurrentTime,
		"total_time":     progress.TotalTime,
		"remaining_time": remainingTime,
		"percentage":     progress.Percentage,
		"is_playing":     isPlaying,
		"played_text":    progress.PlayedText, // 已播放的文本
	}

	if progress.CurrentWord != nil {
		progressInfo["current_word"] = map[string]interface{}{
			"word":       progress.CurrentWord.Word,
			"start_time": progress.CurrentWord.StartTime,
			"end_time":   progress.CurrentWord.EndTime,
		}
	}

	result := map[string]interface{}{
		"success":  true,
		"progress": progressInfo,
	}

	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}
