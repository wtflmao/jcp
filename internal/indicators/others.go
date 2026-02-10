package indicators

import "math"

// ATR 计算平均真实波动范围 (周期14)
func ATR(highs, lows, closes []float64) []float64 {
	n := len(closes)
	result := make([]float64, n)
	if n < 2 {
		return result
	}

	// 计算 True Range
	tr := make([]float64, n)
	tr[0] = highs[0] - lows[0]
	for i := 1; i < n; i++ {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(hl, math.Max(hc, lc))
	}

	period := 14
	if n < period {
		return result
	}

	// 初始 ATR = 前14日 TR 平均
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += tr[i]
	}
	result[period-1] = sum / float64(period)

	// Wilder 平滑
	for i := period; i < n; i++ {
		result[i] = (result[i-1]*float64(period-1) + tr[i]) / float64(period)
	}
	return result
}

// BIAS 计算乖离率 (周期6)
// BIAS6 = (Close - MA6) / MA6 * 100
func BIAS(closes []float64) []float64 {
	n := len(closes)
	result := make([]float64, n)
	ma6 := SMA(closes, 6)

	for i := 5; i < n; i++ {
		if ma6[i] > 0 {
			result[i] = (closes[i] - ma6[i]) / ma6[i] * 100
		}
	}
	return result
}

// BRARResult 单日 BRAR 结果
type BRARResult struct {
	BR float64
	AR float64
}

// BRAR 计算人气意愿指标 (周期26)
// AR = SUM(H-O, 26) / SUM(O-L, 26) * 100
// BR = SUM(H-PC, 26) / SUM(PC-L, 26) * 100  (负值取0)
func BRAR(opens, highs, lows, closes []float64) []BRARResult {
	n := len(closes)
	result := make([]BRARResult, n)
	if n < 27 {
		return result
	}

	for i := 26; i < n; i++ {
		var arUp, arDn, brUp, brDn float64
		for j := i - 25; j <= i; j++ {
			arUp += highs[j] - opens[j]
			arDn += opens[j] - lows[j]

			pc := closes[j-1] // 前收
			hpc := highs[j] - pc
			if hpc < 0 {
				hpc = 0
			}
			pcl := pc - lows[j]
			if pcl < 0 {
				pcl = 0
			}
			brUp += hpc
			brDn += pcl
		}

		if arDn > 0 {
			result[i].AR = arUp / arDn * 100
		}
		if brDn > 0 {
			result[i].BR = brUp / brDn * 100
		}
	}
	return result
}
