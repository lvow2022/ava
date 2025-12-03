package main

import (
	"ava/internal/tts"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

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

	// 创建并配置 ChatModel（不绑定任何工具）
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		Model:   "doubao-1-5-pro-32k-250115",
		APIKey:  "42189dbf-a896-4162-b9d8-3dccf4b1ded5",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 创建一个只包含 ChatModel 的 chain（用于手动处理逻辑）
	// ChatModel 输出 *schema.Message，所以 chain 类型应该是 []*schema.Message -> *schema.Message
	chain := compose.NewChain[[]*schema.Message, *schema.Message]()
	chain.AppendChatModel(chatModel, compose.WithNodeName("chat_model"))
	chatModelAgent, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 创建读取器
	reader := bufio.NewReader(os.Stdin)

	// 保存对话历史，首先添加系统提示词
	systemPrompt := `你是一个智能语音助手。你的回复会通过语音播放给用户，你要尽量假装自己是真人，一切的回复都应该符合人类口语的习惯。

重要工作流程：
1. 当用户输入时，你应该首先调用 get_playback_progress 工具来查询当前播放进度
2. 根据工具返回的进度信息（is_playing, remaining_time, played_text 等），结合用户输入自主判断，是否需要停止当前播放。


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

使用规则：
1. 收到用户输入时，建议先调用 get_playback_progress 工具了解当前播放状态
2. 根据 is_playing 状态和用户输入内容进行判断：
   - 如果 is_playing 为 false（当前没有播放）：
     * 直接使用 <say> 标签回复用户，不需要调用 <stop> 或 <ignore> 标签
   - 如果 is_playing 为 true（当前正在播放）：
     * 如果用户输入是无关字符、无意义内容、随意输入（如乱码、无意义的字符组合、测试输入等），使用 <ignore></ignore> 标签，忽略用户输入并继续播放当前语音
     * 如果用户输入是有意义的指令或明确要求停止/打断，使用 <stop></stop> 标签停止当前播放，然后结合上下文判断是否使用 <say> 标签回复语音
3. 所有需要播放给用户的文本都必须放在 <say> 标签内
`
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: systemPrompt,
		},
	}

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

		// 调用 ChatModel
		fmt.Print("Agent: ")
		chatMsg, err := chatModelAgent.Invoke(ctx, messages)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			// 移除最后一条用户消息，因为处理失败
			messages = messages[:len(messages)-1]
			continue
		}

		// 输出并处理 Assistant 的回复
		if chatMsg.Role == schema.Assistant && chatMsg.Content != "" {
			fmt.Println(chatMsg.Content)

			// 使用 TagAwareSpeaker 处理包含 XML 标签的响应
			// Feed 方法会解析 XML 标签并自动调用相应的 speaker 方法
			tagAwareSpeaker.Feed(chatMsg.Content)

			// 将 agent 的响应添加到历史中
			messages = append(messages, chatMsg)
		}

		fmt.Println()
	}
}
