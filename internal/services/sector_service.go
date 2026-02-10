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
	sinaIndustryURL = "http://money.finance.sina.com.cn/quotes_service/api/json_v2.php/Market_Center.getHQNodeData?page=1&num=40&sort=changepercent&asc=0&node=%s"
)

// SectorInfo 板块信息
type SectorInfo struct {
	Name          string  `json:"name"`
	ChangePercent float64 `json:"chg"`
}

// StockSectorData 个股板块数据
type StockSectorData struct {
	Industry SectorInfo   `json:"industry"`
	Concepts []SectorInfo `json:"concepts,omitempty"`
}

// sectorCache 板块缓存
type sectorCache struct {
	data      *StockSectorData
	timestamp time.Time
}

// SectorService 板块/概念服务
type SectorService struct {
	client   *http.Client
	cache    map[string]*sectorCache
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

// NewSectorService 创建板块/概念服务
func NewSectorService() *SectorService {
	return &SectorService{
		client:   proxy.GetManager().GetClientWithTimeout(10 * time.Second),
		cache:    make(map[string]*sectorCache),
		cacheTTL: 30 * time.Second,
	}
}

// GetStockSectors 获取个股所属板块数据（带缓存）
// industry: 从 stock_basic.json 获取的行业名称
func (s *SectorService) GetStockSectors(industry string) (*StockSectorData, error) {
	if industry == "" {
		return nil, fmt.Errorf("industry is empty")
	}

	// 检查缓存
	s.cacheMu.RLock()
	if cached, ok := s.cache[industry]; ok {
		if time.Since(cached.timestamp) < s.cacheTTL {
			s.cacheMu.RUnlock()
			return cached.data, nil
		}
	}
	s.cacheMu.RUnlock()

	// 获取行业板块涨跌数据
	data, err := s.fetchIndustryData(industry)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	s.cacheMu.Lock()
	s.cache[industry] = &sectorCache{
		data:      data,
		timestamp: time.Now(),
	}
	s.cacheMu.Unlock()

	return data, nil
}

// fetchIndustryData 获取行业板块涨跌数据
func (s *SectorService) fetchIndustryData(industry string) (*StockSectorData, error) {
	// 使用新浪行业板块接口
	node := "hangye_" + industry
	url := fmt.Sprintf(sinaIndustryURL, node)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "http://finance.sina.com.cn")

	resp, err := s.client.Do(req)
	if err != nil {
		// API 失败时返回仅行业名称
		return &StockSectorData{
			Industry: SectorInfo{Name: industry},
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &StockSectorData{
			Industry: SectorInfo{Name: industry},
		}, nil
	}

	return s.parseIndustryData(industry, body)
}

// parseIndustryData 解析行业板块数据
func (s *SectorService) parseIndustryData(industry string, body []byte) (*StockSectorData, error) {
	var items []struct {
		Symbol        string  `json:"symbol"`
		Name          string  `json:"name"`
		ChangePercent float64 `json:"changepercent"`
	}

	if err := json.Unmarshal(body, &items); err != nil {
		return &StockSectorData{
			Industry: SectorInfo{Name: industry},
		}, nil
	}

	// 计算板块平均涨跌幅
	var totalChg float64
	for _, item := range items {
		totalChg += item.ChangePercent
	}
	avgChg := 0.0
	if len(items) > 0 {
		avgChg = totalChg / float64(len(items))
	}

	return &StockSectorData{
		Industry: SectorInfo{
			Name:          industry,
			ChangePercent: avgChg,
		},
	}, nil
}
