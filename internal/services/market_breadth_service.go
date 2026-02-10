package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/run-bigpig/jcp/internal/pkg/proxy"
)

const (
	sinaStockCountURL = "http://vip.stock.finance.sina.com.cn/quotes_service/api/json_v2.php/Market_Center.getHQNodeStockCount?node=hs_a"
)

// MarketBreadth 全市场涨跌统计
type MarketBreadth struct {
	AdvanceCount   int `json:"advance"`
	DeclineCount   int `json:"decline"`
	FlatCount      int `json:"flat"`
	LimitUpCount   int `json:"limit_up"`
	LimitDownCount int `json:"limit_down"`
	TotalCount     int `json:"total"`
}

// breadthCache 缓存条目
type breadthCache struct {
	data      *MarketBreadth
	timestamp time.Time
}

// MarketBreadthService 全市场涨跌统计服务
type MarketBreadthService struct {
	client   *http.Client
	cache    *breadthCache
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

// NewMarketBreadthService 创建全市场涨跌统计服务
func NewMarketBreadthService() *MarketBreadthService {
	return &MarketBreadthService{
		client:   proxy.GetManager().GetClientWithTimeout(10 * time.Second),
		cacheTTL: 10 * time.Second,
	}
}

// GetMarketBreadth 获取全市场涨跌统计（带缓存）
func (s *MarketBreadthService) GetMarketBreadth() (*MarketBreadth, error) {
	s.cacheMu.RLock()
	if s.cache != nil && time.Since(s.cache.timestamp) < s.cacheTTL {
		defer s.cacheMu.RUnlock()
		return s.cache.data, nil
	}
	s.cacheMu.RUnlock()

	data, err := s.fetchMarketBreadth()
	if err != nil {
		return nil, err
	}

	s.cacheMu.Lock()
	s.cache = &breadthCache{
		data:      data,
		timestamp: time.Now(),
	}
	s.cacheMu.Unlock()

	return data, nil
}

// fetchMarketBreadth 从新浪API获取全市场涨跌统计
func (s *MarketBreadthService) fetchMarketBreadth() (*MarketBreadth, error) {
	req, err := http.NewRequest("GET", sinaStockCountURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "http://finance.sina.com.cn")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseMarketBreadth(body)
}

// parseMarketBreadth 解析新浪API返回的涨跌统计
func (s *MarketBreadthService) parseMarketBreadth(body []byte) (*MarketBreadth, error) {
	// 新浪API返回格式: {"count":"5000","upcount":"1800","downcount":"2100",...}
	var raw struct {
		Count     json.Number `json:"count"`
		UpCount   json.Number `json:"upcount"`
		DownCount json.Number `json:"downcount"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse market breadth error: %w", err)
	}

	total, _ := raw.Count.Int64()
	up, _ := raw.UpCount.Int64()
	down, _ := raw.DownCount.Int64()
	flat := total - up - down

	return &MarketBreadth{
		AdvanceCount: int(up),
		DeclineCount: int(down),
		FlatCount:    int(flat),
		TotalCount:   int(total),
	}, nil
}
