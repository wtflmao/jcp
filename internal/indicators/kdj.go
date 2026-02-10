package indicators

// KDJResult 单日 KDJ 结果
type KDJResult struct {
	K float64
	D float64
	J float64
}

// KDJ 计算 KDJ 指标 (9, 3, 3)
// 初始 K=D=50
func KDJ(highs, lows, closes []float64) []KDJResult {
	n := len(closes)
	result := make([]KDJResult, n)
	if n < 9 {
		return result
	}

	// 计算 RSV 序列
	rsv := make([]float64, n)
	for i := 8; i < n; i++ {
		high9 := highs[i]
		low9 := lows[i]
		for j := i - 8; j < i; j++ {
			if highs[j] > high9 {
				high9 = highs[j]
			}
			if lows[j] < low9 {
				low9 = lows[j]
			}
		}
		if high9-low9 > 0 {
			rsv[i] = (closes[i] - low9) / (high9 - low9) * 100
		} else {
			rsv[i] = 50
		}
	}

	// K, D 递推，初始值 K=D=50
	k := 50.0
	d := 50.0
	for i := 8; i < n; i++ {
		k = 2.0/3.0*k + 1.0/3.0*rsv[i]
		d = 2.0/3.0*d + 1.0/3.0*k
		j := 3*k - 2*d
		result[i] = KDJResult{K: k, D: d, J: j}
	}

	return result
}
