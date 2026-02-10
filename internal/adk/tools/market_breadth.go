package tools

import (
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// GetMarketBreadthInput 市场广度输入参数
type GetMarketBreadthInput struct{}

// GetMarketBreadthOutput 市场广度输出
type GetMarketBreadthOutput struct {
	Data string `json:"data" jsonschema:"全市场涨跌统计数据"`
}

// createMarketBreadthTool 创建市场广度工具
func (r *Registry) createMarketBreadthTool() (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetMarketBreadthInput) (GetMarketBreadthOutput, error) {

		if r.marketBreadthService == nil {
			return GetMarketBreadthOutput{Data: "市场广度服务不可用"}, nil
		}

		breadth, err := r.marketBreadthService.GetMarketBreadth()
		if err != nil {
			return GetMarketBreadthOutput{}, err
		}

		result := fmt.Sprintf(
			"全市场: 上涨%d家 下跌%d家 平盘%d家 | 涨停%d家 跌停%d家 | 共%d家",
			breadth.AdvanceCount, breadth.DeclineCount, breadth.FlatCount,
			breadth.LimitUpCount, breadth.LimitDownCount, breadth.TotalCount,
		)

		return GetMarketBreadthOutput{Data: result}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_market_breadth",
		Description: "获取全市场涨跌统计数据，包括上涨/下跌/平盘家数、涨停/跌停家数",
	}, handler)
}
