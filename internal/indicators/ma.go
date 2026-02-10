package indicators

// SMA 计算简单移动平均线
// closes: 收盘价序列, period: 周期
// 返回与 closes 等长的 SMA 序列，前 period-1 个值为 0
func SMA(closes []float64, period int) []float64 {
	n := len(closes)
	result := make([]float64, n)
	if n < period || period <= 0 {
		return result
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	result[period-1] = sum / float64(period)

	for i := period; i < n; i++ {
		sum += closes[i] - closes[i-period]
		result[i] = sum / float64(period)
	}
	return result
}

// EMA 计算指数移动平均线
// closes: 收盘价序列, period: 周期
// 初始值使用前 period 个值的 SMA
func EMA(closes []float64, period int) []float64 {
	n := len(closes)
	result := make([]float64, n)
	if n < period || period <= 0 {
		return result
	}

	k := 2.0 / float64(period+1)

	// 初始值 = 前 period 个值的 SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	result[period-1] = sum / float64(period)

	for i := period; i < n; i++ {
		result[i] = closes[i]*k + result[i-1]*(1-k)
	}
	return result
}

// MATrend 判断均线多空排列
// 返回: "bull"(多头排列 MA5>MA10>MA20), "bear"(空头排列), "cross"(交叉/纠缠)
func MATrend(ma5, ma10, ma20 float64) string {
	if ma5 <= 0 || ma10 <= 0 || ma20 <= 0 {
		return ""
	}
	if ma5 > ma10 && ma10 > ma20 {
		return "bull"
	}
	if ma5 < ma10 && ma10 < ma20 {
		return "bear"
	}
	return "cross"
}
