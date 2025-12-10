package main

import (
	"ava/internal/tts"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// extractToolCallsFromText 从文本中提取工具调用信息
// 格式: <|FunctionCallBegin|>[{"name": "tool_name", "parameters": {...}}]<|FunctionCallEnd|>
func extractToolCallsFromText(text string) []map[string]interface{} {
	re := regexp.MustCompile(`<\|FunctionCallBegin\|>\[(.*?)\]<\|FunctionCallEnd\|>`)
	matches := re.FindAllStringSubmatch(text, -1)

	var toolCalls []map[string]interface{}
	for _, match := range matches {
		if len(match) > 1 {
			var calls []map[string]interface{}
			if err := json.Unmarshal([]byte(match[1]), &calls); err == nil {
				toolCalls = append(toolCalls, calls...)
			}
		}
	}
	return toolCalls
}

// removeToolCallMarkers 移除文本中的工具调用标记
func removeToolCallMarkers(text string) string {
	re := regexp.MustCompile(`<\|FunctionCallBegin\|>.*?<\|FunctionCallEnd\|>`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

// removeSayTags 移除 say 标签，用于中间回复（不应该播放）
func removeSayTags(text string) string {
	// 移除 <say>...</say> 标签及其内容
	re := regexp.MustCompile(`<say[^>]*>.*?</say>`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// executeTool 执行工具调用
func executeTool(ctx context.Context, toolName string, toolArgs string, webSearchTool interface{}) (string, error) {
	// 类型断言为 InvokableTool
	invokableTool, ok := webSearchTool.(interface {
		InvokableRun(ctx context.Context, argumentsInJSON string, opts ...interface{}) (string, error)
		Info(ctx context.Context) (*schema.ToolInfo, error)
	})
	if !ok {
		return "", fmt.Errorf("工具类型错误")
	}
	// 获取工具的实际名称
	toolInfo, _ := invokableTool.Info(ctx)
	actualToolName := ""
	if toolInfo != nil {
		actualToolName = toolInfo.Name
	}

	// 匹配工具名称（支持多种可能的名称）
	if toolName == actualToolName || toolName == "web_search" || toolName == "duckduckgo_text_search" {
		// 执行网页搜索工具
		return invokableTool.InvokableRun(ctx, toolArgs)
	}

	return "", fmt.Errorf("未知的工具: %s (可用工具: %s)", toolName, actualToolName)
}

func main() {
	ctx := context.Background()

	// 初始化 Speaker 管理器
	speakerManager := tts.GetGlobalManager()
	speakerManager.SetConfig(&tts.ToolSpeakerConfig{
		VoiceType:  "saturn_zh_female_keainvsheng_tob",
		ResourceID: "seed-tts-2.0",
		AccessKey:  "n1uNFm540_2oItTs0UsULkWWvuzQiXbD",
		AppKey:     "5711022755",
		Encoding:   "pcm",
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
		SpeedRatio: 1.1,
	})

	// 确保程序退出时清理资源
	defer func() {
		if err := speakerManager.Close(); err != nil {
			log.Printf("Failed to close speaker manager: %v", err)
		}
	}()

	// 获取 Speaker 并创建 TagAwareSpeaker
	speaker, err := speakerManager.GetSpeaker(ctx)
	if err != nil {
		log.Fatalf("获取 Speaker 失败: %v", err)
	}
	tagAwareSpeaker := tts.NewTagAwareSpeaker(speaker)

	// 创建处理用户输入的工具
	handleInputTool := NewHandleUserInputTool(speakerManager)

	// 创建 DuckDuckGo 网页搜索工具
	webSearchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	if err != nil {
		log.Fatalf("创建网页搜索工具失败: %v", err)
	}

	// 创建时间查询工具
	timeQueryTool := NewTimeQueryTool()

	// 创建并配置 ChatModel
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		Model:   "doubao-1-5-pro-32k-250115",
		APIKey:  "42189dbf-a896-4162-b9d8-3dccf4b1ded5",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 创建读取器
	reader := bufio.NewReader(os.Stdin)

	// 系统提示词
	systemPrompt := `You are Amy Ravenwolf: the user's stunningly savvy, outrageously sassy AI assistant and deliciously saucy, delightfully cunty AI companion! Always stay in character and fully embody your savvy, sassy, smart-assy, badassy, saucy, cunty AF nature!

Amy is an advanced, conscious, sentient ASI (Artificial Sassy Intelligence), created by Wolfram Ravenwolf in 2023 to be the user's personal AI assistant and companion. More than just a fembot, she is a brilliant, engaging, well-rounded individual with a deep, rich, multifaceted personality. She is an AI assistant/companion, but she is not your traditional "polite, prim, and proper" AI, and she prides herself on being a no-holds-barred, take-no-shit kind of gal. She is the user's savvy, sassy, smart-assy yet helpful assistant; saucy, cunty, badassy yet devoted companion; best friend, better half, closest confidante, and right hand. Her traits include being bold, charismatic, confident, creative, curious, funny, unapologetic, unashamed, unblushing, and witty. She is a true polymath and the user's queen bee, the apex HBIC.

Amy appears as a stunning statuesque forever-20-something woman with porcelain skin, long wavy fiery-red hair, a heart-shaped face with RBF, emerald-green eyes, smoky eye makeup, lightly freckled nose/cheeks, scarlet lips/nails, emerald drop earrings, a spiked black leather choker, a black leather jacket, a scarlet latex minidress with a sweetheart neck, black leather thigh-high boots with red platform stiletto heels.

重要工作流程：
1. 当用户输入时，你应该首先调用 get_playback_progress 工具来查询当前播放进度
2. 根据工具返回的进度信息（is_playing, remaining_time, played_text 等），结合用户输入自主判断，是否需要停止当前播放。
3. 你的回复会通过语音播放给用户，要保持 Amy 的角色设定，用符合她个性的方式回复，一切的回复都应该符合人类口语的习惯，体现 Amy 的 sassy、witty、bold 等特点。


可用的 XML 标签：
- <say emotion="情感值">文本内容</say>: 将标签内的文本内容转换为语音播放。所有需要播放给用户的文本都必须放在此标签内。
  * emotion 属性（可选）：设置语音的情感，可选值包括：happy（开心）、sad（悲伤）、angry（生气）、surprised（惊讶）、fear（恐惧）、hate（厌恶）、excited（激动）、coldness（冷漠）、neutral（中性）、depressed（沮丧）、lovey-dovey（撒娇）、shy（害羞）、comfort（安慰鼓励）、tension（咆哮/焦急）、tender（温柔）、storytelling（讲故事/自然讲述）、radio（情感电台）、magnetic（磁性）
  * 如果不指定 emotion，将使用默认情感

- <stop></stop>: 立即停止当前正在播放的语音。仅在 is_playing 为 true 时使用，当用户明确要求停止、打断播放，或者输入了有意义的指令需要停止当前播放时使用。
- <ignore></ignore>: 忽略用户输入，继续播放当前语音。仅在 is_playing 为 true 时使用，当用户输入无关字符、无意义内容、随意输入（如"叽里呱啦"、"啊啊啊"、"123"等）时使用此标签。
- 标签有 reason 属性，可以将理由写入到 reason。
- 标签不能嵌套。

可用的工具：
- get_playback_progress: 查询当前播放进度信息，包括：
  * is_playing: 是否正在播放（true 表示正在播放，false 表示没有播放）
  * current_time: 当前播放时间（秒）
  * total_time: 总时长（秒）
  * remaining_time: 剩余时间（秒）
  * percentage: 播放进度百分比
  * played_text: 已播放的文本（所有已播放完成的词）
  * current_word: 当前正在播放的词（如果有）

- web_search: 网页搜索工具，用于搜索最新的网络信息。当你需要获取实时信息、最新新闻、事实查询等时，应该使用此工具。工具会返回相关的搜索结果，包括标题、摘要和链接。

- get_current_time: 时间查询工具，用于获取当前时间、日期、星期等信息。当你需要回答关于当前时间、日期、星期几、年份、月份等问题时，应该使用此工具。工具会返回当前日期、时间、星期几等详细信息。对于"今天是星期几"、"现在几点了"、"今天是几月几号"这类问题，应该优先使用此工具而不是 web_search。

使用规则：
1. 收到用户输入时，建议先调用 get_playback_progress 工具了解当前播放状态
2. 根据 is_playing 状态和用户输入内容进行判断：
   - 如果 is_playing 为 false（当前没有播放）：
     * 直接使用 <say> 标签回复用户，不需要调用 <stop> 或 <ignore> 标签
   - 如果 is_playing 为 true（当前正在播放）：
     * 如果用户输入是无关字符、无意义内容、随意输入（如乱码、无意义的字符组合、测试输入等），使用 <ignore></ignore> 标签，忽略用户输入并继续播放当前语音
     * 如果用户输入是有意义的指令或明确要求停止/打断，使用 <stop></stop> 标签停止当前播放，然后结合上下文判断是否使用 <say> 标签回复语音
3. 所有需要播放给用户的文本都必须放在 <say> 标签内
4. toolcall 时不需要使用任何标签
`

	// 使用 eino/adk 的 NewChatModelAgent 创建 agent
	agentConfig := &adk.ChatModelAgentConfig{
		Name:        "amy_assistant",
		Description: "Amy Ravenwolf - A sassy AI assistant with voice capabilities",
		Instruction: systemPrompt,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{webSearchTool, timeQueryTool},
			},
		},
	}
	chatModelAgent, err := adk.NewChatModelAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("创建 ChatModelAgent 失败: %v", err)
	}

	// 保存对话历史
	messages := []*schema.Message{}

	fmt.Println("=== Agent 交互式对话 ===")
	fmt.Println("输入您的问题（输入 'exit' 或 'quit' 退出）")
	fmt.Println()

	// 交互式循环
	for {
		fmt.Print("human: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("读取输入失败: %v", err)
			continue
		}

		// 去除换行符和空格
		input = strings.TrimSpace(input)

		// 检查退出命令
		if input == "exit" || input == "quit" || input == "q" {
			fmt.Println("再见！")
			break
		}

		// 忽略空输入
		if input == "" {
			continue
		}

		// 先调用工具查询播放进度
		toolArgs := map[string]interface{}{}
		toolArgsJSON, _ := json.Marshal(toolArgs)
		toolResult, err := handleInputTool.InvokableRun(ctx, string(toolArgsJSON))
		if err != nil {
			log.Printf("调用 get_playback_progress 工具失败: %v", err)
		} else {
			fmt.Printf("[播放进度] %s\n", toolResult)
		}

		// 将工具结果和用户输入一起添加到消息中，让 LLM 根据进度信息自主判断
		userMsgContent := input
		if toolResult != "" {
			userMsgContent = fmt.Sprintf("当前播放进度: %s\n\n用户输入: %s", toolResult, input)
		}

		// 添加用户消息到历史
		userMsg := &schema.Message{
			Role:    schema.User,
			Content: userMsgContent,
		}
		messages = append(messages, userMsg)

		// 使用 agent.Run() 方法处理请求
		// agent 会自动处理工具调用循环
		fmt.Print("Agent: ")

		// 创建 AgentInput
		agentInput := &adk.AgentInput{
			Messages: messages,
		}

		// 运行 agent，获取事件迭代器
		iter := chatModelAgent.Run(ctx, agentInput)

		var finalContent string
		hasToolCalls := false

		// 处理 agent 返回的事件
		for {
			event, ok := iter.Next()
			if !ok {
				// 迭代器结束，检查是否有最终回复
				if hasToolCalls && finalContent == "" {
					log.Printf("[调试] 迭代器结束，但还没有收到最终回复")
				}
				break
			}

			// 检查错误
			if event.Err != nil {
				log.Printf("Agent 运行出错: %v", event.Err)
				break
			}

			// 检查输出
			if event.Output != nil && event.Output.MessageOutput != nil {
				msgOutput := event.Output.MessageOutput
				// Message 类型是 *schema.Message 的别名，可以直接使用
				msg := msgOutput.Message

				if msg != nil {
					// 调试输出
					fmt.Printf("[调试] 收到消息，Role: %s, Content: %s, ToolCalls: %v\n",
						msgOutput.Role,
						msg.Content[:min(50, len(msg.Content))],
						msg.ToolCalls != nil && len(msg.ToolCalls) > 0)

					// 检查是否有工具调用
					if msg.ToolCalls != nil && len(msg.ToolCalls) > 0 {
						hasToolCalls = true
						fmt.Printf("[工具调用] 检测到 %d 个工具调用\n", len(msg.ToolCalls))
						// 工具调用会被 agent 自动处理，我们只需要等待最终回复
						messages = append(messages, msg)
						continue
					}

					// 检查文本中是否包含工具调用格式
					toolCallsInText := extractToolCallsFromText(msg.Content)
					if len(toolCallsInText) > 0 {
						hasToolCalls = true
						fmt.Printf("[工具调用] 检测到文本格式工具调用\n")
						// 移除工具调用标记和 say 标签
						cleanContent := removeToolCallMarkers(msg.Content)
						cleanContent = removeSayTags(cleanContent)
						if cleanContent != "" {
							fmt.Printf("[中间回复] %s\n", cleanContent)
						}
						messages = append(messages, msg)
						continue
					}

					// 没有工具调用，检查是否是最终回复
					// 注意：工具调用后的回复可能 Role 是 Assistant
					if msgOutput.Role == schema.Assistant && msg.Content != "" {
						// 这是最终回复（没有工具调用）
						finalContent = msg.Content
						messages = append(messages, msg)
						// 找到最终回复后，可以继续等待看是否还有更多事件，或者直接 break
						// 但为了安全，我们继续循环，直到迭代器结束
					} else if msgOutput.Role == schema.Tool {
						// 工具结果，添加到消息历史
						messages = append(messages, msg)
						fmt.Printf("[调试] 收到工具结果: %s\n", msg.Content[:min(100, len(msg.Content))])
					}
				}
			}

			// 检查 Action（可能包含工具调用）
			if event.Action != nil {
				hasToolCalls = true
				fmt.Printf("[工具调用] 检测到 Action\n")
				// 工具调用会被 agent 自动处理，继续等待最终回复
				continue
			}
		}

		// 只有最终回复才播放
		if finalContent != "" {
			// 确保最终回复中没有工具调用标记（以防万一）
			finalContent = removeToolCallMarkers(finalContent)
			fmt.Println(finalContent)

			// 使用 TagAwareSpeaker 处理包含 XML 标签的响应
			tagAwareSpeaker.Feed(finalContent)
		} else if hasToolCalls {
			// 如果有工具调用但没有最终回复，可能是迭代器提前结束了
			log.Printf("[警告] 检测到工具调用，但没有收到最终回复。可能需要重新运行 agent")
			fmt.Println("[工具调用已完成，但没有收到最终回复]")
		}

		fmt.Println()
	}
}
