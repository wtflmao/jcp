package indicators

// MACDResult 单日 MACD 结果
type MACDResult struct {
	DIF  float64
	DEA  float64
	Hist float64 // MACD 柱 = 2 * (DIF - DEA)
}

// MACD 计算 MACD 指标 (12, 26, 9)
// 返回与 closes 等长的 MACDResult 序列
func MACD(closes []float64) []MACDResult {
	n := len(closes)
	result := make([]MACDResult, n)
	if n < 26 {
		return result
	}

	ema12 := EMA(closes, 12)
	ema26 := EMA(closes, 26)

	// DIF = EMA12 - EMA26，从第 26 个值开始有效
	dif := make([]float64, n)
	for i := 25; i < n; i++ {
		dif[i] = ema12[i] - ema26[i]
	}

	// DEA = EMA(DIF, 9)，从第 26+9-1=34 个值开始有效
	// 手动计算 DEA 的 EMA，因为 dif 前面有零值
	dea := make([]float64, n)
	if n >= 34 {
		// DEA 初始值 = dif[25..33] 的平均
		sum := 0.0
		for i := 25; i < 34; i++ {
			sum += dif[i]
		}
		dea[33] = sum / 9.0

		k := 2.0 / 10.0 // 2/(9+1)
		for i := 34; i < n; i++ {
			dea[i] = dif[i]*k + dea[i-1]*(1-k)
		}
	}

	// 组装结果
	for i := 33; i < n; i++ {
		result[i] = MACDResult{
			DIF:  dif[i],
			DEA:  dea[i],
			Hist: 2 * (dif[i] - dea[i]),
		}
	}
	return result
}
