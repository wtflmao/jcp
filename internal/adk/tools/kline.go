package tools

import (
	"fmt"

	"github.com/run-bigpig/jcp/internal/indicators"
	"github.com/run-bigpig/jcp/internal/services"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// GetKLineInput K线数据输入参数
type GetKLineInput struct {
	Code   string `json:"code" jsonschema:"股票代码，如 sh600519"`
	Period string `json:"period,omitempty" jsonschema:"K线周期: 1m(5分钟), 1d(日线), 1w(周线), 1mo(月线)，默认1d"`
	Days   int    `json:"days,omitzero" jsonschema:"K线根数，不传则按周期自动设置合理默认值"`
	Mode   string `json:"mode,omitempty" jsonschema:"输出模式: raw(原始OHLCV,默认), analysis(含完整技术指标，仅日线有效)"`
}

// GetKLineOutput K线数据输出
type GetKLineOutput struct {
	Data string `json:"data" jsonschema:"K线数据"`
}

// createKLineTool 创建K线数据工具
func (r *Registry) createKLineTool() (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetKLineInput) (GetKLineOutput, error) {
		if input.Code == "" {
			fmt.Println("[Tool:get_kline_data] 错误: 未提供股票代码")
			return GetKLineOutput{Data: "请提供股票代码"}, nil
		}

		period := input.Period
		if period == "" {
			period = "1d"
		}

		// analysis 模式：日线 + 完整技术指标
		if input.Mode == "analysis" && period == "1d" {
			return r.handleAnalysisMode(input.Code)
		}

		// raw 模式（默认）：原始 OHLCV
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

		return GetKLineOutput{Data: result}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_kline_data",
		Description: "获取股票K线数据，支持5分钟线、日线、周线、月线。设置mode=analysis可获取含MACD/KDJ/BOLL/DMI等完整技术指标的分析数据（仅日线有效）",
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

// handleAnalysisMode 处理 analysis 模式
func (r *Registry) handleAnalysisMode(code string) (GetKLineOutput, error) {
	// 获取 250 根日K（为 EMA/MACD/ADX 等递推型指标提供充足预热期）
	klines, err := r.marketService.GetKLineData(code, "1d", 250)
	if err != nil {
		fmt.Printf("[Tool:get_kline_data:analysis] K线获取错误: %v\n", err)
		return GetKLineOutput{}, err
	}

	// 获取流通股本（非ETF），用于计算换手率和成交额
	var floatShares float64
	if !services.IsETF(code) && r.stockInfoService != nil {
		info, err := r.stockInfoService.GetExtendedInfo(code)
		if err == nil && info.FloatMarketCap > 0 && len(klines) > 0 {
			lastClose := klines[len(klines)-1].Close
			if lastClose > 0 {
				floatShares = info.FloatMarketCap / lastClose
			}
		}
	}

	// 补算成交额（新浪K线API不返回amount）和换手率序列
	var turnoverRates []float64
	if floatShares > 0 {
		turnoverRates = make([]float64, len(klines))
	}
	for i := range klines {
		// amount ≈ (open+close)/2 * volume（用均价近似）
		if klines[i].Amount == 0 && klines[i].Volume > 0 {
			avgPrice := (klines[i].Open + klines[i].Close) / 2
			klines[i].Amount = avgPrice * float64(klines[i].Volume)
		}
		if floatShares > 0 {
			turnoverRates[i] = float64(klines[i].Volume) / floatShares * 100
		}
	}

	// 计算全部技术指标
	analysis := indicators.ComputeAll(klines, 30, turnoverRates)

	// 填充外部数据到 snapshot
	r.fillSnapshotExternalData(code, analysis)

	// 格式化输出
	result := indicators.FormatFullAnalysis(analysis)
	return GetKLineOutput{Data: result}, nil
}

// fillSnapshotExternalData 填充快照的外部数据
func (r *Registry) fillSnapshotExternalData(code string, analysis *indicators.FullAnalysis) {
	if analysis == nil {
		return
	}

	isETF := services.IsETF(code)

	// 流通市值/流通股本（非ETF）
	if !isETF && r.stockInfoService != nil {
		info, err := r.stockInfoService.GetExtendedInfo(code)
		if err == nil {
			analysis.Snapshot.FloatCap = indicators.FormatMarketCap(info.FloatMarketCap)
			if info.FloatMarketCap > 0 {
				// 从最后一根K线取收盘价估算流通股本
				if len(analysis.Series) > 0 {
					lastClose := analysis.Series[len(analysis.Series)-1].Close
					if lastClose > 0 {
						floatShares := info.FloatMarketCap / lastClose
						analysis.Snapshot.FloatShares = indicators.FormatShares(floatShares)
					}
				}
			}
		}
	}

	// 板块/概念（非ETF）
	if !isETF && r.sectorService != nil {
		sectorData, err := r.sectorService.GetStockSectors(r.getStockIndustry(code))
		if err == nil && sectorData != nil {
			analysis.Snapshot.Sector = fmt.Sprintf("%s %.2f%%",
				sectorData.Industry.Name, sectorData.Industry.ChangePercent)
			for _, c := range sectorData.Concepts {
				analysis.Snapshot.Concepts = append(analysis.Snapshot.Concepts,
					fmt.Sprintf("%s %.2f%%", c.Name, c.ChangePercent))
			}
		}
	}

	// 全市场涨跌统计
	if r.marketBreadthService != nil {
		breadth, err := r.marketBreadthService.GetMarketBreadth()
		if err == nil && breadth != nil {
			analysis.Snapshot.MarketBreadth = &indicators.MarketBreadthData{
				AdvanceCount:   breadth.AdvanceCount,
				DeclineCount:   breadth.DeclineCount,
				FlatCount:      breadth.FlatCount,
				LimitUpCount:   breadth.LimitUpCount,
				LimitDownCount: breadth.LimitDownCount,
				TotalCount:     breadth.TotalCount,
			}
		}
	}
}

// getStockIndustry 从嵌入数据获取个股行业
func (r *Registry) getStockIndustry(code string) string {
	if r.configService == nil {
		return ""
	}
	// 从 configService 获取股票基础信息中的行业字段
	// 代码格式转换: sh600519 → 600519
	symbol := code
	if len(code) > 2 {
		symbol = code[2:]
	}
	info := r.configService.GetStockBasicInfo(symbol)
	if info != nil {
		return info.Industry
	}
	return ""
}
