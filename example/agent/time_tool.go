package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// TimeQueryTool 时间查询工具，用于获取当前时间、日期、星期等信息
type TimeQueryTool struct{}

func NewTimeQueryTool() *TimeQueryTool {
	return &TimeQueryTool{}
}

func (t *TimeQueryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_current_time",
		Desc: `查询当前时间、日期、星期等信息。当你需要回答关于当前时间、日期、星期几、年份、月份等问题时，应该使用此工具。
工具会返回：
- 当前日期（年-月-日格式）
- 当前时间（时:分:秒格式）
- 星期几（中文）
- 年份、月份、日期
- 时区信息`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Desc:     "查询类型，可选值：date（日期）、time（时间）、weekday（星期）、datetime（日期时间）、all（全部信息）。默认为 all",
				Type:     schema.String,
				Required: false,
			},
		}),
	}, nil
}

func (t *TimeQueryTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}

	_ = json.Unmarshal([]byte(argumentsInJSON), &args)

	// 获取当前时间（使用中国时区）
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600) // 如果加载失败，使用固定时区 UTC+8
	}
	now := time.Now().In(loc)

	// 星期几的中文映射
	weekdays := map[time.Weekday]string{
		time.Sunday:    "星期日",
		time.Monday:    "星期一",
		time.Tuesday:   "星期二",
		time.Wednesday: "星期三",
		time.Thursday:  "星期四",
		time.Friday:    "星期五",
		time.Saturday:  "星期六",
	}

	// 月份的中文映射
	months := map[time.Month]string{
		time.January:   "一月",
		time.February:  "二月",
		time.March:     "三月",
		time.April:     "四月",
		time.May:       "五月",
		time.June:      "六月",
		time.July:      "七月",
		time.August:    "八月",
		time.September: "九月",
		time.October:   "十月",
		time.November:  "十一月",
		time.December:  "十二月",
	}

	result := make(map[string]interface{})

	query := args.Query
	if query == "" {
		query = "all"
	}

	switch query {
	case "date":
		result["date"] = now.Format("2006-01-02")
		result["year"] = now.Year()
		result["month"] = int(now.Month())
		result["month_cn"] = months[now.Month()]
		result["day"] = now.Day()
		result["weekday"] = weekdays[now.Weekday()]
		result["weekday_en"] = now.Weekday().String()

	case "time":
		result["time"] = now.Format("15:04:05")
		result["hour"] = now.Hour()
		result["minute"] = now.Minute()
		result["second"] = now.Second()

	case "weekday":
		result["weekday"] = weekdays[now.Weekday()]
		result["weekday_en"] = now.Weekday().String()
		result["date"] = now.Format("2006-01-02")

	case "datetime":
		result["datetime"] = now.Format("2006-01-02 15:04:05")
		result["date"] = now.Format("2006-01-02")
		result["time"] = now.Format("15:04:05")
		result["weekday"] = weekdays[now.Weekday()]

	default: // all
		result["datetime"] = now.Format("2006-01-02 15:04:05")
		result["date"] = now.Format("2006-01-02")
		result["time"] = now.Format("15:04:05")
		result["year"] = now.Year()
		result["month"] = int(now.Month())
		result["month_cn"] = months[now.Month()]
		result["day"] = now.Day()
		result["weekday"] = weekdays[now.Weekday()]
		result["weekday_en"] = now.Weekday().String()
		result["hour"] = now.Hour()
		result["minute"] = now.Minute()
		result["second"] = now.Second()
		result["timezone"] = loc.String()
		result["timestamp"] = now.Unix()
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

