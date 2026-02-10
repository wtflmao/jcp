package tools

import (
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// GetKLineInput K线数据输入参数
type GetKLineInput struct {
	Code   string `json:"code" jsonschema:"股票代码，如 sh600519"`
	Period string `json:"period,omitempty" jsonschema:"K线周期: 1m(5分钟), 1d(日线), 1w(周线), 1mo(月线)，默认1d"`
	Days   int    `json:"days,omitzero" jsonschema:"K线根数，不传则按周期自动设置合理默认值(1m=240,1d=60,1w=52,1mo=24)"`
}

// GetKLineOutput K线数据输出
type GetKLineOutput struct {
	Data string `json:"data" jsonschema:"K线数据"`
}

// createKLineTool 创建K线数据工具
func (r *Registry) createKLineTool() (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetKLineInput) (GetKLineOutput, error) {
		fmt.Printf("[Tool:get_kline_data] 调用开始, code=%s, period=%s, days=%d\n", input.Code, input.Period, input.Days)

		if input.Code == "" {
			fmt.Println("[Tool:get_kline_data] 错误: 未提供股票代码")
			return GetKLineOutput{Data: "请提供股票代码"}, nil
		}

		period := input.Period
		if period == "" {
			period = "1d"
		}
		defaultDatalen, maxOutput := periodDefaults(period)
		datalen := input.Days
		if datalen == 0 {
			datalen = defaultDatalen
		}

		klines, err := r.marketService.GetKLineData(input.Code, period, datalen)
		if err != nil {
			fmt.Printf("[Tool:get_kline_data] 错误: %v\n", err)
			return GetKLineOutput{}, err
		}

		// 格式化输出（按周期截断避免过长）
		var result string
		start := 0
		if len(klines) > maxOutput {
			start = len(klines) - maxOutput
		}
		for _, k := range klines[start:] {
			result += fmt.Sprintf("%s: 开%.2f 高%.2f 低%.2f 收%.2f 量%d\n",
				k.Time, k.Open, k.High, k.Low, k.Close, k.Volume)
		}

		fmt.Printf("[Tool:get_kline_data] 调用完成, 返回%d条数据\n", len(klines))
		return GetKLineOutput{Data: result}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_kline_data",
		Description: "获取股票K线数据，支持5分钟线、日线、周线、月线",
	}, handler)
}

// periodDefaults 根据K线周期返回合理的默认请求根数和最大输出条数
func periodDefaults(period string) (defaultDatalen, maxOutput int) {
	switch period {
	case "1m":
		return 240, 48
	case "1w":
		return 52, 20
	case "1mo":
		return 24, 12
	default: // "1d"
		return 60, 30
	}
}
