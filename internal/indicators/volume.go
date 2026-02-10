package indicators

// OBV 计算能量潮
// 返回与输入等长的 OBV 序列
func OBV(closes []float64, volumes []int64) []float64 {
	n := len(closes)
	result := make([]float64, n)
	if n == 0 {
		return result
	}

	result[0] = float64(volumes[0])
	for i := 1; i < n; i++ {
		if closes[i] > closes[i-1] {
			result[i] = result[i-1] + float64(volumes[i])
		} else if closes[i] < closes[i-1] {
			result[i] = result[i-1] - float64(volumes[i])
		} else {
			result[i] = result[i-1]
		}
	}
	return result
}

// OBVSlopeDir 判断 OBV 5日斜率方向
// 使用简单线性回归方向
func OBVSlopeDir(obv []float64, idx int) string {
	if idx < 4 || idx >= len(obv) {
		return ""
	}
	// 取最近5个值做线性回归
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := 0; i < 5; i++ {
		x := float64(i)
		y := obv[idx-4+i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	slope := (5*sumXY - sumX*sumY) / (5*sumX2 - sumX*sumX)

	if slope > 0 {
		return "up"
	} else if slope < 0 {
		return "down"
	}
	return "flat"
}

// VolMA 计算成交量移动平均
func VolMA(volumes []int64, period int) []float64 {
	n := len(volumes)
	result := make([]float64, n)
	if n < period || period <= 0 {
		return result
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += float64(volumes[i])
	}
	result[period-1] = sum / float64(period)

	for i := period; i < n; i++ {
		sum += float64(volumes[i]) - float64(volumes[i-period])
		result[i] = sum / float64(period)
	}
	return result
}

// VolRatio 计算量比 = 当日成交量 / 5日均量
func VolRatio(volume int64, volMA5 float64) float64 {
	if volMA5 <= 0 {
		return 0
	}
	return float64(volume) / volMA5
}

// TurnoverLevel 根据60日换手率分位数判断换手水平
// rates: 最近60日换手率序列, current: 当日换手率
func TurnoverLevel(rates []float64, current float64) string {
	if len(rates) == 0 || current <= 0 {
		return ""
	}

	// 计算当前值在序列中的分位数
	count := 0
	for _, r := range rates {
		if r <= current {
			count++
		}
	}
	percentile := float64(count) / float64(len(rates))

	switch {
	case percentile >= 0.9:
		return "extreme"
	case percentile >= 0.8:
		return "high"
	case percentile < 0.2:
		return "low"
	default:
		return "normal"
	}
}
