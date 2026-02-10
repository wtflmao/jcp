package indicators

import (
	"fmt"
	"math"
	"sort"

	"github.com/run-bigpig/jcp/internal/models"
)

// MarketBreadthData 全市场涨跌统计
type MarketBreadthData struct {
	AdvanceCount   int `json:"advance"`
	DeclineCount   int `json:"decline"`
	FlatCount      int `json:"flat"`
	LimitUpCount   int `json:"limit_up"`
	LimitDownCount int `json:"limit_down"`
	TotalCount     int `json:"total"`
}

// TechnicalSnapshot 全局状态快照（当天单点值）
type TechnicalSnapshot struct {
	MA60          float64            `json:"ma60"`
	MA120         float64            `json:"ma120"`
	High60        float64            `json:"high60"`
	Low60         float64            `json:"low60"`
	Pos60         float64            `json:"pos60"`
	FloatCap      string             `json:"float_cap,omitempty"`
	FloatShares   string             `json:"float_shares,omitempty"`
	Sector        string             `json:"sector,omitempty"`
	Concepts      []string           `json:"concepts,omitempty"`
	MarketBreadth *MarketBreadthData `json:"market,omitempty"`
}

// StatusSummary 预处理状态字段
type StatusSummary struct {
	MATrend     string  `json:"ma_trend"`
	MACDCross   string  `json:"macd_cross,omitempty"`
	MACDStatus  string  `json:"macd_status,omitempty"`
	KDJStatus   string  `json:"kdj_status,omitempty"`
	BOLLSqueeze bool    `json:"boll_squeeze,omitempty"`
	TrendMode   string  `json:"trend_mode,omitempty"`
	OBVSlope    string  `json:"obv_slope,omitempty"`
	VolRatio    float64 `json:"vol_ratio"`
	BandWidth   float64 `json:"band_width"`
}

// DayRow 单日时序数据行
type DayRow struct {
	Date          string
	Open          float64
	High          float64
	Low           float64
	Close         float64
	ChangePct     float64
	Volume        int64
	Amount        float64
	MA5           float64
	MA10          float64
	MA20          float64
	DIF           float64
	DEA           float64
	MACDHist      float64
	MACDSignal    string // MACD信号：gold/dead/top_div/bot_div
	K             float64
	D             float64
	J             float64
	KDJSignal     string // KDJ信号：gold/dead/ob/os
	BOLLUpper     float64
	BOLLMid       float64
	BOLLLower     float64
	BOLLWidth     float64 // 带宽 (Upper-Lower)/Mid
	ADX           float64
	VolMA5        float64
	TurnoverRate  float64
	TurnoverLevel string
	OBVVal        float64
	ATRVal        float64
	BIASVal       float64
	BRVal         float64
	ARVal         float64
}

// FullAnalysis 完整分析结果
type FullAnalysis struct {
	Snapshot TechnicalSnapshot
	Status   StatusSummary
	Series   []DayRow
}

// ComputeAll 计算全部技术指标
// klines: 应传入 250+ 根日K数据（为 EMA/MACD/ADX 等递推型指标提供充足预热期）
// outputDays: 输出最近多少天的时序数据（通常30）
// turnoverRates: 每日换手率序列（与 klines 等长），无数据传 nil
func ComputeAll(klines []models.KLineData, outputDays int, turnoverRates []float64) *FullAnalysis {
	n := len(klines)
	if n == 0 {
		return &FullAnalysis{}
	}

	// 提取 OHLCV 序列
	opens := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]int64, n)
	amounts := make([]float64, n)
	for i, k := range klines {
		opens[i] = k.Open
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
		volumes[i] = k.Volume
		amounts[i] = k.Amount
	}

	// 计算所有指标序列
	ma5 := SMA(closes, 5)
	ma10 := SMA(closes, 10)
	ma20 := SMA(closes, 20)
	ma60 := SMA(closes, 60)
	ma120 := SMA(closes, 120)
	macdAll := MACD(closes)
	kdjAll := KDJ(highs, lows, closes)
	bollAll := BOLL(closes)
	dmiAll := DMI(highs, lows, closes)
	obvAll := OBV(closes, volumes)
	volMA5 := VolMA(volumes, 5)
	atrAll := ATR(highs, lows, closes)
	biasAll := BIAS(closes)
	brarAll := BRAR(opens, highs, lows, closes)

	// 构建 Snapshot
	last := n - 1
	snapshot := buildSnapshot(closes, highs, lows, ma60, ma120, last)

	// 构建 Status
	status := buildStatus(
		ma5, ma10, ma20, macdAll, kdjAll, bollAll, dmiAll,
		obvAll, volumes, volMA5, last,
	)

	// 构建 Series（最近 outputDays 天）
	start := n - outputDays
	if start < 0 {
		start = 0
	}
	series := buildSeries(
		klines, closes, ma5, ma10, ma20,
		macdAll, kdjAll, bollAll, dmiAll,
		obvAll, volMA5, atrAll, biasAll, brarAll,
		turnoverRates, start, n,
	)

	return &FullAnalysis{
		Snapshot: snapshot,
		Status:   status,
		Series:   series,
	}
}

// buildSnapshot 构建全局状态快照
func buildSnapshot(closes, highs, lows, ma60, ma120 []float64, last int) TechnicalSnapshot {
	snap := TechnicalSnapshot{}
	if last < 0 {
		return snap
	}

	snap.MA60 = round2(ma60[last])
	snap.MA120 = round2(ma120[last])

	// 近60日最高/最低
	start60 := last - 59
	if start60 < 0 {
		start60 = 0
	}
	snap.High60 = highs[start60]
	snap.Low60 = lows[start60]
	for i := start60; i <= last; i++ {
		if highs[i] > snap.High60 {
			snap.High60 = highs[i]
		}
		if lows[i] < snap.Low60 {
			snap.Low60 = lows[i]
		}
	}

	// 当前价在60日区间的位置
	range60 := snap.High60 - snap.Low60
	if range60 > 0 {
		snap.Pos60 = round2((closes[last] - snap.Low60) / range60 * 100)
	}

	return snap
}

// buildStatus 构建预处理状态摘要
func buildStatus(
	ma5, ma10, ma20 []float64,
	macdAll []MACDResult,
	kdjAll []KDJResult,
	bollAll []BOLLResult,
	dmiAll []DMIResult,
	obvAll []float64,
	volumes []int64,
	volMA5 []float64,
	last int,
) StatusSummary {
	s := StatusSummary{}
	if last < 1 {
		return s
	}

	// MA 趋势
	s.MATrend = MATrend(ma5[last], ma10[last], ma20[last])

	// MACD 交叉检测
	s.MACDCross = detectMACDCross(macdAll, last)
	s.MACDStatus = detectMACDStatus(macdAll, last)

	// KDJ 状态
	s.KDJStatus = detectKDJStatus(kdjAll, last)

	// BOLL 收窄检测
	s.BandWidth = round2(BandWidth(bollAll[last]))
	s.BOLLSqueeze = detectBOLLSqueeze(bollAll, last)

	// 趋势模式
	if dmiAll[last].ADX > 25 {
		s.TrendMode = "trend"
	} else if dmiAll[last].ADX < 20 {
		s.TrendMode = "choppy"
	} else {
		s.TrendMode = "transition"
	}

	// OBV 斜率
	s.OBVSlope = OBVSlopeDir(obvAll, last)

	// 量比
	s.VolRatio = round2(VolRatio(volumes[last], volMA5[last]))

	return s
}

// detectMACDCross 检测 MACD 金叉/死叉及持续天数
func detectMACDCross(macd []MACDResult, last int) string {
	if last < 1 {
		return ""
	}
	// 向前搜索最近一次交叉
	for days := 0; days < 10 && last-days >= 1; days++ {
		i := last - days
		prevDIF := macd[i-1].DIF
		prevDEA := macd[i-1].DEA
		curDIF := macd[i].DIF
		curDEA := macd[i].DEA

		wasBelow := prevDIF <= prevDEA
		isAbove := curDIF > curDEA
		wasAbove := prevDIF >= prevDEA
		isBelow := curDIF < curDEA

		if wasBelow && isAbove {
			return fmt.Sprintf("gold_%d", days+1)
		}
		if wasAbove && isBelow {
			return fmt.Sprintf("dead_%d", days+1)
		}
	}
	return ""
}

// detectMACDStatus 检测 MACD 柱状态
func detectMACDStatus(macd []MACDResult, last int) string {
	if last < 1 {
		return ""
	}
	cur := macd[last].Hist
	prev := macd[last-1].Hist

	if cur > 0 {
		if cur > prev {
			return "widen_up"
		}
		return "narrow_up"
	} else if cur < 0 {
		if cur < prev {
			return "widen_dn"
		}
		return "narrow_dn"
	}
	return ""
}

// detectKDJStatus 检测 KDJ 状态
func detectKDJStatus(kdj []KDJResult, last int) string {
	if last < 1 {
		return ""
	}

	cur := kdj[last]
	prev := kdj[last-1]

	// J值超买钝化检测（J>100持续天数）
	if cur.J > 100 {
		days := 1
		for i := last - 1; i >= 0 && kdj[i].J > 100; i-- {
			days++
		}
		return fmt.Sprintf("j_ob_%d", days)
	}

	// J值超卖钝化检测（J<0持续天数）
	if cur.J < 0 {
		days := 1
		for i := last - 1; i >= 0 && kdj[i].J < 0; i-- {
			days++
		}
		return fmt.Sprintf("j_os_%d", days)
	}

	// 低位金叉（K<30区域，K上穿D）
	if prev.K <= prev.D && cur.K > cur.D && cur.K < 30 {
		return "bottom_gold"
	}

	// 高位死叉（K>70区域，K下穿D）
	if prev.K >= prev.D && cur.K < cur.D && cur.K > 70 {
		return "top_dead"
	}

	return ""
}

// detectBOLLSqueeze 检测布林带收窄（带宽 < 60日10%分位）
func detectBOLLSqueeze(boll []BOLLResult, last int) bool {
	start := last - 59
	if start < 0 {
		start = 0
	}

	var bws []float64
	for i := start; i <= last; i++ {
		bw := BandWidth(boll[i])
		if bw > 0 {
			bws = append(bws, bw)
		}
	}
	if len(bws) < 10 {
		return false
	}

	sorted := make([]float64, len(bws))
	copy(sorted, bws)
	sort.Float64s(sorted)

	p10 := sorted[len(sorted)/10]
	currentBW := BandWidth(boll[last])
	return currentBW <= p10
}

// buildSeries 构建时序数据
func buildSeries(
	klines []models.KLineData,
	closes, ma5, ma10, ma20 []float64,
	macdAll []MACDResult,
	kdjAll []KDJResult,
	bollAll []BOLLResult,
	dmiAll []DMIResult,
	obvAll, volMA5, atrAll, biasAll []float64,
	brarAll []BRARResult,
	turnoverRates []float64,
	start, end int,
) []DayRow {
	rows := make([]DayRow, 0, end-start)

	for i := start; i < end; i++ {
		k := klines[i]
		row := DayRow{
			Date:   k.Time,
			Open:   k.Open,
			High:   k.High,
			Low:    k.Low,
			Close:  k.Close,
			Volume: k.Volume,
			Amount: k.Amount,
		}

		// 涨跌幅
		if i > 0 && closes[i-1] > 0 {
			row.ChangePct = (closes[i] - closes[i-1]) / closes[i-1] * 100
		}

		// 均线
		row.MA5 = ma5[i]
		row.MA10 = ma10[i]
		row.MA20 = ma20[i]

		// MACD
		row.DIF = macdAll[i].DIF
		row.DEA = macdAll[i].DEA
		row.MACDHist = macdAll[i].Hist
		row.MACDSignal = detectDayMACDSignal(macdAll, closes, i)

		// KDJ
		row.K = kdjAll[i].K
		row.D = kdjAll[i].D
		row.J = kdjAll[i].J
		row.KDJSignal = detectDayKDJSignal(kdjAll, i)

		// BOLL
		row.BOLLUpper = bollAll[i].Upper
		row.BOLLMid = bollAll[i].Mid
		row.BOLLLower = bollAll[i].Lower
		if bollAll[i].Mid > 0 {
			row.BOLLWidth = round2((bollAll[i].Upper - bollAll[i].Lower) / bollAll[i].Mid * 100)
		}

		// DMI
		row.ADX = dmiAll[i].ADX

		// Volume
		row.VolMA5 = volMA5[i]
		row.OBVVal = obvAll[i] - obvAll[start] // 相对于 series 窗口起点的增量

		// 换手率
		if turnoverRates != nil && i < len(turnoverRates) {
			row.TurnoverRate = turnoverRates[i]
			// 计算换手率分位
			start60 := i - 59
			if start60 < 0 {
				start60 = 0
			}
			row.TurnoverLevel = TurnoverLevel(
				turnoverRates[start60:i+1],
				turnoverRates[i],
			)
		}

		// ATR, BIAS, BRAR
		row.ATRVal = atrAll[i]
		row.BIASVal = biasAll[i]
		if i < len(brarAll) {
			row.BRVal = brarAll[i].BR
			row.ARVal = brarAll[i].AR
		}

		rows = append(rows, row)
	}
	return rows
}

// detectDayMACDSignal 检测单日MACD信号
func detectDayMACDSignal(macd []MACDResult, closes []float64, i int) string {
	if i < 1 {
		return ""
	}
	prev := macd[i-1]
	cur := macd[i]

	// 金叉：DIF从下穿上DEA
	if prev.DIF <= prev.DEA && cur.DIF > cur.DEA {
		return "gold"
	}
	// 死叉：DIF从上穿下DEA
	if prev.DIF >= prev.DEA && cur.DIF < cur.DEA {
		return "dead"
	}

	// 顶背离：价格创新高但DIF没创新高（向前看20根）
	if i >= 5 {
		topDiv := detectMACDDivergence(macd, closes, i, true)
		if topDiv {
			return "top_div"
		}
		botDiv := detectMACDDivergence(macd, closes, i, false)
		if botDiv {
			return "bot_div"
		}
	}

	return ""
}

// detectMACDDivergence 检测MACD背离
// isTop=true检测顶背离，false检测底背离
func detectMACDDivergence(macd []MACDResult, closes []float64, i int, isTop bool) bool {
	lookback := 20
	if i < lookback {
		lookback = i
	}

	if isTop {
		// 顶背离：价格创新高但DIF没创新高
		if closes[i] <= closes[i-1] {
			return false
		}
		for j := i - 5; j >= i-lookback; j-- {
			if j > 0 && j < len(closes)-1 && closes[j] > closes[j-1] && closes[j] > closes[j+1] {
				if closes[i] > closes[j] && macd[i].DIF < macd[j].DIF {
					return true
				}
				break
			}
		}
	} else {
		// 底背离：价格创新低但DIF没创新低
		if closes[i] >= closes[i-1] {
			return false
		}
		for j := i - 5; j >= i-lookback; j-- {
			if j > 0 && j < len(closes)-1 && closes[j] < closes[j-1] && closes[j] < closes[j+1] {
				if closes[i] < closes[j] && macd[i].DIF > macd[j].DIF {
					return true
				}
				break
			}
		}
	}
	return false
}

// detectDayKDJSignal 检测单日KDJ信号
func detectDayKDJSignal(kdj []KDJResult, i int) string {
	if i < 1 {
		return ""
	}
	prev := kdj[i-1]
	cur := kdj[i]

	// 金叉：K上穿D
	if prev.K <= prev.D && cur.K > cur.D {
		if cur.K < 30 {
			return "gold_low" // 低位金叉
		}
		return "gold"
	}
	// 死叉：K下穿D
	if prev.K >= prev.D && cur.K < cur.D {
		if cur.K > 70 {
			return "dead_high" // 高位死叉
		}
		return "dead"
	}
	// 超买/超卖
	if cur.J > 100 {
		return "ob"
	}
	if cur.J < 0 {
		return "os"
	}
	return ""
}

// round2 保留2位小数
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
