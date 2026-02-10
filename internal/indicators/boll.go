package indicators

import "math"

// BOLLResult 单日布林线结果
type BOLLResult struct {
	Upper float64
	Mid   float64
	Lower float64
}

// BOLL 计算布林线 (20, 2)
func BOLL(closes []float64) []BOLLResult {
	n := len(closes)
	result := make([]BOLLResult, n)
	if n < 20 {
		return result
	}

	ma20 := SMA(closes, 20)

	for i := 19; i < n; i++ {
		// 计算 20 日标准差
		sum := 0.0
		for j := i - 19; j <= i; j++ {
			diff := closes[j] - ma20[i]
			sum += diff * diff
		}
		std := math.Sqrt(sum / 20.0)

		result[i] = BOLLResult{
			Upper: ma20[i] + 2*std,
			Mid:   ma20[i],
			Lower: ma20[i] - 2*std,
		}
	}
	return result
}

// BandWidth 计算布林带宽百分比
// BandWidth = (Upper - Lower) / Mid * 100
func BandWidth(boll BOLLResult) float64 {
	if boll.Mid <= 0 {
		return 0
	}
	return (boll.Upper - boll.Lower) / boll.Mid * 100
}
