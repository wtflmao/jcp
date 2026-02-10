package indicators

import "math"

// DMIResult 单日 DMI 结果
type DMIResult struct {
	PDI  float64 // +DI
	MDI  float64 // -DI
	ADX  float64
	ADXR float64
}

// DMI 计算 DMI 指标 (周期14)
func DMI(highs, lows, closes []float64) []DMIResult {
	n := len(closes)
	result := make([]DMIResult, n)
	if n < 15 {
		return result
	}

	// 计算 TR, +DM, -DM
	tr := make([]float64, n)
	pdm := make([]float64, n)
	mdm := make([]float64, n)

	for i := 1; i < n; i++ {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(hl, math.Max(hc, lc))

		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]

		if upMove > downMove && upMove > 0 {
			pdm[i] = upMove
		}
		if downMove > upMove && downMove > 0 {
			mdm[i] = downMove
		}
	}

	// Wilder 平滑 (14期)
	period := 14
	smoothTR := make([]float64, n)
	smoothPDM := make([]float64, n)
	smoothMDM := make([]float64, n)

	// 初始值: 前14期之和
	for i := 1; i <= period; i++ {
		smoothTR[period] += tr[i]
		smoothPDM[period] += pdm[i]
		smoothMDM[period] += mdm[i]
	}

	for i := period + 1; i < n; i++ {
		smoothTR[i] = smoothTR[i-1] - smoothTR[i-1]/float64(period) + tr[i]
		smoothPDM[i] = smoothPDM[i-1] - smoothPDM[i-1]/float64(period) + pdm[i]
		smoothMDM[i] = smoothMDM[i-1] - smoothMDM[i-1]/float64(period) + mdm[i]
	}

	// 计算 +DI, -DI, DX
	dx := make([]float64, n)
	for i := period; i < n; i++ {
		if smoothTR[i] > 0 {
			result[i].PDI = smoothPDM[i] / smoothTR[i] * 100
			result[i].MDI = smoothMDM[i] / smoothTR[i] * 100
		}
		diSum := result[i].PDI + result[i].MDI
		if diSum > 0 {
			dx[i] = math.Abs(result[i].PDI-result[i].MDI) / diSum * 100
		}
	}

	// ADX = EMA(DX, 14)，使用 Wilder 平滑
	adxStart := period + period // 需要 14 个 DX 值
	if adxStart >= n {
		return result
	}

	// ADX 初始值
	adxSum := 0.0
	for i := period; i < adxStart; i++ {
		adxSum += dx[i]
	}
	result[adxStart-1].ADX = adxSum / float64(period)

	for i := adxStart; i < n; i++ {
		result[i].ADX = (result[i-1].ADX*float64(period-1) + dx[i]) / float64(period)
	}

	// ADXR = (ADX_today + ADX_14days_ago) / 2
	for i := adxStart + period - 1; i < n; i++ {
		result[i].ADXR = (result[i].ADX + result[i-period].ADX) / 2
	}

	return result
}
