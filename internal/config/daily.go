/**
  @author: Hanhai
  @since: 2025/3/17 14:30:00
  @desc: 每日API请求统计数据管理
**/

package config

import (
	"encoding/json"
	"flowsilicon/internal/logger"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DailyStats 每日统计数据结构
type DailyStats struct {
	Date     string                `json:"date"`
	Requests DailyRequestStats     `json:"requests"`
	Tokens   DailyTokenStats       `json:"tokens"`
	Models   map[string]ModelStats `json:"models"`
	Hourly   []HourlyStats         `json:"hourly"`
}

// DailyRequestStats 每日请求统计
type DailyRequestStats struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// DailyTokenStats 每日令牌统计
type DailyTokenStats struct {
	Total      int `json:"total"`
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
}

// ModelStats 模型使用统计
type ModelStats struct {
	Requests int `json:"requests"`
	Tokens   int `json:"tokens"`
}

// HourlyStats 每小时统计
type HourlyStats struct {
	Hour     int `json:"hour"`
	Requests int `json:"requests"`
	Tokens   int `json:"tokens"`
}

// KeyUsage 密钥使用统计
type KeyUsage struct {
	Requests int `json:"requests"`
	Tokens   int `json:"tokens"`
}

// DailyData 每日数据文件结构
type DailyData struct {
	Version     string                         `json:"version"`
	Description string                         `json:"description"`
	LastUpdated string                         `json:"last_updated"`
	DailyStats  []DailyStats                   `json:"daily_stats"`
	KeysUsage   map[string]map[string]KeyUsage `json:"keys_usage"`
}

var (
	dailyData     *DailyData
	dailyDataLock sync.RWMutex
	dailyFilePath = "./data/daily.json"
)

// InitDailyStats 初始化每日统计数据
func InitDailyStats() error {
	// 确保data目录存在
	dataDir := filepath.Dir(dailyFilePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Error("创建数据目录失败: %v", err)
		return err
	}

	// 尝试加载现有数据
	if err := loadDailyData(); err != nil {
		// 如果文件不存在，创建新的数据结构
		if os.IsNotExist(err) {
			dailyData = createDefaultDailyData()
			// 立即保存到文件
			if err := saveDailyData(); err != nil {
				logger.Error("保存每日统计数据失败: %v", err)
				return err
			}
			logger.Info("创建了新的每日统计数据文件")
		} else {
			logger.Error("加载每日统计数据失败: %v", err)
			return err
		}
	} else {
		logger.Info("成功加载每日统计数据")
	}

	// 确保今天的数据存在
	ensureTodayDataExists()

	return nil
}

// loadDailyData 从文件加载每日统计数据
func loadDailyData() error {
	dailyDataLock.Lock()
	defer dailyDataLock.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(dailyFilePath); os.IsNotExist(err) {
		return err
	}

	// 读取文件内容
	data, err := os.ReadFile(dailyFilePath)
	if err != nil {
		return err
	}

	// 解析JSON
	var loadedData DailyData
	if err := json.Unmarshal(data, &loadedData); err != nil {
		return err
	}

	dailyData = &loadedData
	return nil
}

// saveDailyData 保存每日统计数据到文件
func saveDailyData() error {
	dailyDataLock.RLock()
	defer dailyDataLock.RUnlock()

	if dailyData == nil {
		return nil
	}

	// 更新最后更新时间
	dailyData.LastUpdated = time.Now().Format(time.RFC3339)

	// 序列化为JSON
	data, err := json.MarshalIndent(dailyData, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(dailyFilePath, data, 0644)
}

// createDefaultDailyData 创建默认的每日统计数据结构
func createDefaultDailyData() *DailyData {
	today := time.Now().Format("2006-01-02")

	// 创建24小时的统计数据
	hourlyStats := make([]HourlyStats, 24)
	for i := 0; i < 24; i++ {
		hourlyStats[i] = HourlyStats{
			Hour:     i,
			Requests: 0,
			Tokens:   0,
		}
	}

	return &DailyData{
		Version:     "1.0",
		Description: "每日API请求统计数据",
		LastUpdated: time.Now().Format(time.RFC3339),
		DailyStats: []DailyStats{
			{
				Date: today,
				Requests: DailyRequestStats{
					Total:   0,
					Success: 0,
					Failed:  0,
				},
				Tokens: DailyTokenStats{
					Total:      0,
					Prompt:     0,
					Completion: 0,
				},
				Models: make(map[string]ModelStats),
				Hourly: hourlyStats,
			},
		},
		KeysUsage: make(map[string]map[string]KeyUsage),
	}
}

// ensureTodayDataExists 确保今天的数据存在
func ensureTodayDataExists() {
	dailyDataLock.Lock()
	defer dailyDataLock.Unlock()

	if dailyData == nil {
		dailyData = createDefaultDailyData()
		return
	}

	today := time.Now().Format("2006-01-02")

	// 检查今天的数据是否存在
	for _, stats := range dailyData.DailyStats {
		if stats.Date == today {
			return
		}
	}

	// 创建24小时的统计数据
	hourlyStats := make([]HourlyStats, 24)
	for i := 0; i < 24; i++ {
		hourlyStats[i] = HourlyStats{
			Hour:     i,
			Requests: 0,
			Tokens:   0,
		}
	}

	// 添加今天的数据
	dailyData.DailyStats = append(dailyData.DailyStats, DailyStats{
		Date: today,
		Requests: DailyRequestStats{
			Total:   0,
			Success: 0,
			Failed:  0,
		},
		Tokens: DailyTokenStats{
			Total:      0,
			Prompt:     0,
			Completion: 0,
		},
		Models: make(map[string]ModelStats),
		Hourly: hourlyStats,
	})

	// 如果数据超过30天，删除最旧的数据
	if len(dailyData.DailyStats) > 30 {
		dailyData.DailyStats = dailyData.DailyStats[len(dailyData.DailyStats)-30:]
	}
}

// AddDailyRequestStat 添加每日请求统计
func AddDailyRequestStat(apiKey, model string, requestCount, promptTokens, completionTokens int, isSuccess bool) {
	dailyDataLock.Lock()
	defer dailyDataLock.Unlock()

	// 确保今天的数据存在
	today := time.Now().Format("2006-01-02")
	currentHour := time.Now().Hour()

	var todayStats *DailyStats
	var todayIndex int

	// 查找今天的数据
	for i, stats := range dailyData.DailyStats {
		if stats.Date == today {
			todayStats = &dailyData.DailyStats[i]
			todayIndex = i
			break
		}
	}

	// 如果今天的数据不存在，创建新的
	if todayStats == nil {
		// 创建24小时的统计数据
		hourlyStats := make([]HourlyStats, 24)
		for i := 0; i < 24; i++ {
			hourlyStats[i] = HourlyStats{
				Hour:     i,
				Requests: 0,
				Tokens:   0,
			}
		}

		dailyData.DailyStats = append(dailyData.DailyStats, DailyStats{
			Date: today,
			Requests: DailyRequestStats{
				Total:   0,
				Success: 0,
				Failed:  0,
			},
			Tokens: DailyTokenStats{
				Total:      0,
				Prompt:     0,
				Completion: 0,
			},
			Models: make(map[string]ModelStats),
			Hourly: hourlyStats,
		})

		todayIndex = len(dailyData.DailyStats) - 1
		todayStats = &dailyData.DailyStats[todayIndex]
	}

	// 更新请求统计
	todayStats.Requests.Total += requestCount
	if isSuccess {
		todayStats.Requests.Success += requestCount
	} else {
		todayStats.Requests.Failed += requestCount
	}

	// 更新令牌统计
	totalTokens := promptTokens + completionTokens
	todayStats.Tokens.Total += totalTokens
	todayStats.Tokens.Prompt += promptTokens
	todayStats.Tokens.Completion += completionTokens

	// 更新模型统计
	if model != "" {
		if _, exists := todayStats.Models[model]; !exists {
			todayStats.Models[model] = ModelStats{
				Requests: 0,
				Tokens:   0,
			}
		}

		modelStats := todayStats.Models[model]
		modelStats.Requests += requestCount
		modelStats.Tokens += totalTokens
		todayStats.Models[model] = modelStats
	}

	// 更新小时统计
	todayStats.Hourly[currentHour].Requests += requestCount
	todayStats.Hourly[currentHour].Tokens += totalTokens

	// 更新API密钥使用统计
	if apiKey != "" {
		maskedKey := maskAPIKey(apiKey)

		if _, exists := dailyData.KeysUsage[maskedKey]; !exists {
			dailyData.KeysUsage[maskedKey] = make(map[string]KeyUsage)
		}

		if _, exists := dailyData.KeysUsage[maskedKey][today]; !exists {
			dailyData.KeysUsage[maskedKey][today] = KeyUsage{
				Requests: 0,
				Tokens:   0,
			}
		}

		keyUsage := dailyData.KeysUsage[maskedKey][today]
		keyUsage.Requests += requestCount
		keyUsage.Tokens += totalTokens
		dailyData.KeysUsage[maskedKey][today] = keyUsage
	}

	// 更新数据库中的数据
	dailyData.DailyStats[todayIndex] = *todayStats

	// 异步保存数据
	go func() {
		if err := saveDailyData(); err != nil {
			logger.Error("保存每日统计数据失败: %v", err)
		}
	}()
}

// GetDailyStats 获取指定日期的统计数据
func GetDailyStats(date string) (*DailyStats, error) {
	dailyDataLock.RLock()
	defer dailyDataLock.RUnlock()

	if dailyData == nil {
		return nil, nil
	}

	// 如果未指定日期，使用今天的日期
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 查找指定日期的数据
	for _, stats := range dailyData.DailyStats {
		if stats.Date == date {
			// 返回副本以避免外部修改
			statsCopy := stats
			return &statsCopy, nil
		}
	}

	return nil, nil
}

// GetKeyUsageStats 获取指定密钥在指定日期的使用统计
func GetKeyUsageStats(apiKey, date string) (*KeyUsage, error) {
	dailyDataLock.RLock()
	defer dailyDataLock.RUnlock()

	if dailyData == nil {
		return nil, nil
	}

	// 如果未指定日期，使用今天的日期
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	maskedKey := maskAPIKey(apiKey)

	// 查找指定密钥和日期的数据
	if keyData, exists := dailyData.KeysUsage[maskedKey]; exists {
		if usageData, exists := keyData[date]; exists {
			// 返回副本以避免外部修改
			usageCopy := usageData
			return &usageCopy, nil
		}
	}

	return nil, nil
}

// GetAllDailyStats 获取所有日期的统计数据
func GetAllDailyStats() (map[string]*DailyStats, error) {
	dailyDataLock.RLock()
	defer dailyDataLock.RUnlock()

	if dailyData == nil {
		return nil, nil
	}

	// 创建一个副本以避免并发问题
	result := make(map[string]*DailyStats)
	for _, stats := range dailyData.DailyStats {
		statsCopy := stats
		result[stats.Date] = &statsCopy
	}
	return result, nil
}

// maskAPIKey 掩盖API密钥
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 6 {
		return "******"
	}
	return apiKey[:6] + "******"
}
