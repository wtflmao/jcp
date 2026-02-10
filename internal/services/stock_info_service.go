package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/run-bigpig/jcp/internal/pkg/proxy"
)

const (
	// 东方财富 Push2 API：单股查询市值/换手率/PE
	eastmoneyStockURL = "https://push2.eastmoney.com/api/qt/stock/get?secid=%s&fields=f57,f58,f116,f117,f162,f168"
)

// StockExtendedInfo 个股扩展信息
type StockExtendedInfo struct {
	FloatMarketCap float64 `json:"floatMarketCap"` // 流通市值（元）
	TotalMarketCap float64 `json:"totalMarketCap"` // 总市值（元）
	TurnoverRate   float64 `json:"turnoverRate"`   // 换手率(%)
	PE             float64 `json:"pe"`             // 市盈率
}

// stockInfoCache 缓存条目
type stockInfoCache struct {
	data      *StockExtendedInfo
	timestamp time.Time
}

// StockInfoService 个股扩展信息服务
type StockInfoService struct {
	client   *http.Client
	cache    map[string]*stockInfoCache
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

// NewStockInfoService 创建个股扩展信息服务
func NewStockInfoService() *StockInfoService {
	return &StockInfoService{
		client:   proxy.GetManager().GetClientWithTimeout(10 * time.Second),
		cache:    make(map[string]*stockInfoCache),
		cacheTTL: 30 * time.Second,
	}
}

// IsETF 判断是否为场内ETF
func IsETF(code string) bool {
	return strings.HasPrefix(code, "sh51") ||
		strings.HasPrefix(code, "sz15") ||
		strings.HasPrefix(code, "sh58")
}

// GetExtendedInfo 获取个股扩展信息（带缓存）
func (s *StockInfoService) GetExtendedInfo(code string) (*StockExtendedInfo, error) {
	// 检查缓存
	s.cacheMu.RLock()
	if cached, ok := s.cache[code]; ok {
		if time.Since(cached.timestamp) < s.cacheTTL {
			s.cacheMu.RUnlock()
			return cached.data, nil
		}
	}
	s.cacheMu.RUnlock()

	// 从 API 获取
	info, err := s.fetchExtendedInfo(code)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	s.cacheMu.Lock()
	s.cache[code] = &stockInfoCache{
		data:      info,
		timestamp: time.Now(),
	}
	s.cacheMu.Unlock()

	return info, nil
}

// toSecID 将 sh600519/sz002195 转为东方财富 secid 格式 1.600519/0.002195
func toSecID(code string) string {
	if len(code) < 3 {
		return code
	}
	prefix := code[:2]
	num := code[2:]
	switch prefix {
	case "sh":
		return "1." + num
	default: // sz, bj
		return "0." + num
	}
}

// fetchExtendedInfo 从东方财富API获取个股扩展信息
func (s *StockInfoService) fetchExtendedInfo(code string) (*StockExtendedInfo, error) {
	secid := toSecID(code)
	url := fmt.Sprintf(eastmoneyStockURL, secid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseExtendedInfo(body)
}

// parseExtendedInfo 解析东方财富API返回的个股扩展信息
func (s *StockInfoService) parseExtendedInfo(body []byte) (*StockExtendedInfo, error) {
	var result struct {
		Data struct {
			F116 float64 `json:"f116"` // 总市值（元）
			F117 float64 `json:"f117"` // 流通市值（元）
			F162 float64 `json:"f162"` // 市盈率 * 100
			F168 float64 `json:"f168"` // 换手率 * 100
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse stock info error: %w", err)
	}

	return &StockExtendedInfo{
		FloatMarketCap: result.Data.F117,
		TotalMarketCap: result.Data.F116,
		TurnoverRate:   result.Data.F168 / 100,
		PE:             result.Data.F162 / 100,
	}, nil
}

// truncateBytes 截断字节切片用于日志
func truncateBytes(b []byte, maxLen int) string {
	if len(b) <= maxLen {
		return string(b)
	}
	return string(b[:maxLen]) + "..."
}
