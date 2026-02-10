package indicators

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatSnapshot 将快照格式化为紧凑 JSON（无缩进无换行）
func FormatSnapshot(snap TechnicalSnapshot) string {
	data, err := json.Marshal(snap)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// FormatStatus 将状态摘要格式化为紧凑 JSON
func FormatStatus(status StatusSummary) string {
	data, err := json.Marshal(status)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// formatCoreSeries 核心序列：OHLCV + 涨跌幅
func formatCoreSeries(rows []DayRow) string {
	var sb strings.Builder
	sb.WriteString("Date,Open,High,Low,Close,Change%,Volume,Amount\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f,%s,%s,%s\n",
			r.Date, r.Open, r.High, r.Low, r.Close,
			fmtSign(r.ChangePct),
			formatVolume(r.Volume), formatAmount(r.Amount)))
	}
	return sb.String()
}

// formatTrendSeries 趋势组：MA + ADX
func formatTrendSeries(rows []DayRow) string {
	var sb strings.Builder
	sb.WriteString("Date,MA5,MA10,MA20,ADX\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f\n",
			r.Date, r.MA5, r.MA10, r.MA20, r.ADX))
	}
	return sb.String()
}

// formatMomentumSeries 动能组：MACD + 信号
func formatMomentumSeries(rows []DayRow) string {
	var sb strings.Builder
	sb.WriteString("Date,DIF,DEA,MACD_Hist,Signal\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
			r.Date, fmtSign(r.DIF), fmtSign(r.DEA), fmtSign(r.MACDHist), r.MACDSignal))
	}
	return sb.String()
}

// formatOscillatorSeries 摆动组：KDJ + 信号
func formatOscillatorSeries(rows []DayRow) string {
	var sb strings.Builder
	sb.WriteString("Date,K,D,J,Signal\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%s\n",
			r.Date, r.K, r.D, r.J, r.KDJSignal))
	}
	return sb.String()
}

// formatVolatilitySeries 波动组：BOLL + BandWidth + BIAS + ATR
func formatVolatilitySeries(rows []DayRow) string {
	var sb strings.Builder
	sb.WriteString("Date,BOLL_Upper,BOLL_Mid,BOLL_Lower,BandWidth%,BIAS,ATR\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f,%s,%.2f\n",
			r.Date, r.BOLLUpper, r.BOLLMid, r.BOLLLower, r.BOLLWidth,
			fmtSign(r.BIASVal), r.ATRVal))
	}
	return sb.String()
}

// formatVolumeSeries 量能组：Vol_MA5 + 换手率 + OBV
func formatVolumeSeries(rows []DayRow) string {
	var sb strings.Builder
	sb.WriteString("Date,Vol_MA5,Turnover%,Turnover_Level,OBV_Delta\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s,%s,%.2f,%s,%s\n",
			r.Date, formatVolFloat(r.VolMA5),
			r.TurnoverRate, r.TurnoverLevel, formatOBVSigned(r.OBVVal)))
	}
	return sb.String()
}

// formatOtherSeries 其他组：BRAR
func formatOtherSeries(rows []DayRow) string {
	var sb strings.Builder
	sb.WriteString("Date,BR,AR\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f\n",
			r.Date, r.BRVal, r.ARVal))
	}
	return sb.String()
}

// FormatFullAnalysis 格式化完整分析结果（分组CSV）
func FormatFullAnalysis(analysis *FullAnalysis) string {
	var sb strings.Builder
	sb.WriteString("[Snapshot]\n")
	sb.WriteString(FormatSnapshot(analysis.Snapshot))
	sb.WriteString("\n\n[Status]\n")
	sb.WriteString(FormatStatus(analysis.Status))
	sb.WriteString("\n\n[OHLCV]\n")
	sb.WriteString(formatCoreSeries(analysis.Series))
	sb.WriteString("\n[Trend]\n")
	sb.WriteString(formatTrendSeries(analysis.Series))
	sb.WriteString("\n[MACD]\n")
	sb.WriteString(formatMomentumSeries(analysis.Series))
	sb.WriteString("\n[KDJ]\n")
	sb.WriteString(formatOscillatorSeries(analysis.Series))
	sb.WriteString("\n[Volatility]\n")
	sb.WriteString(formatVolatilitySeries(analysis.Series))
	sb.WriteString("\n[Volume]\n")
	sb.WriteString(formatVolumeSeries(analysis.Series))
	sb.WriteString("\n[BRAR]\n")
	sb.WriteString(formatOtherSeries(analysis.Series))
	return sb.String()
}

// formatVolume 格式化成交量（股），使用 K/M 量词
func formatVolume(v int64) string {
	f := float64(v)
	switch {
	case f >= 1e9:
		return fmt.Sprintf("%.2fG", f/1e9)
	case f >= 1e6:
		return fmt.Sprintf("%.2fM", f/1e6)
	case f >= 1e3:
		return fmt.Sprintf("%.2fK", f/1e3)
	default:
		return fmt.Sprintf("%d", v)
	}
}

// formatAmount 格式化成交额（元），使用 K/M/G/T 量词
func formatAmount(v float64) string {
	switch {
	case v >= 1e12:
		return fmt.Sprintf("%.2fT", v/1e12)
	case v >= 1e9:
		return fmt.Sprintf("%.2fG", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.2fM", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.2fK", v/1e3)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

// formatVolFloat 格式化浮点成交量（均量等）
func formatVolFloat(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.2fG", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.2fM", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.2fK", v/1e3)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// formatOBV 格式化 OBV 值
func formatOBV(v float64) string {
	neg := ""
	abs := v
	if v < 0 {
		neg = "-"
		abs = -v
	}
	switch {
	case abs >= 1e12:
		return fmt.Sprintf("%s%.2fT", neg, abs/1e12)
	case abs >= 1e9:
		return fmt.Sprintf("%s%.2fG", neg, abs/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("%s%.2fM", neg, abs/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("%s%.2fK", neg, abs/1e3)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// FormatMarketCap 格式化市值（元 → 带量词）
func FormatMarketCap(v float64) string {
	switch {
	case v >= 1e12:
		return fmt.Sprintf("%.2fT", v/1e12)
	case v >= 1e9:
		return fmt.Sprintf("%.2fG", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.2fM", v/1e6)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// FormatShares 格式化股本数（股 → 带量词）
func FormatShares(v float64) string {
	return FormatMarketCap(v)
}

// fmtSign 格式化带符号的浮点数，正数加+号
func fmtSign(v float64) string {
	if v > 0 {
		return fmt.Sprintf("+%.2f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// formatOBVSigned 格式化带符号的OBV值
func formatOBVSigned(v float64) string {
	sign := ""
	abs := v
	if v > 0 {
		sign = "+"
	} else if v < 0 {
		sign = "-"
		abs = -v
	}
	switch {
	case abs >= 1e12:
		return fmt.Sprintf("%s%.2fT", sign, abs/1e12)
	case abs >= 1e9:
		return fmt.Sprintf("%s%.2fG", sign, abs/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("%s%.2fM", sign, abs/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("%s%.2fK", sign, abs/1e3)
	default:
		if v > 0 {
			return fmt.Sprintf("+%.0f", v)
		}
		return fmt.Sprintf("%.0f", v)
	}
}
