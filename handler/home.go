package handler

import (
	"database/sql"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atopos31/llmio/common"
	"github.com/atopos31/llmio/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MetricsRes struct {
	Reqs   int64 `json:"reqs"`
	Tokens int64 `json:"tokens"`
}

func Metrics(c *gin.Context) {
	days, err := strconv.Atoi(c.Param("days"))
	if err != nil {
		common.BadRequest(c, "Invalid days parameter")
		return
	}

	now := time.Now()
	year, month, day := now.Date()
	chain := gorm.G[models.ChatLog](models.DB).Where("created_at >= ?", time.Date(year, month, day, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -days))

	reqs, err := chain.Count(c.Request.Context(), "id")
	if err != nil {
		common.InternalServerError(c, "Failed to count requests: "+err.Error())
		return
	}
	var tokens sql.NullInt64
	if err := chain.Select("sum(total_tokens) as tokens").Scan(c.Request.Context(), &tokens); err != nil {
		common.InternalServerError(c, "Failed to sum tokens: "+err.Error())
		return
	}
	common.Success(c, MetricsRes{
		Reqs:   reqs,
		Tokens: tokens.Int64,
	})
}

type DailyMetric struct {
	Date   string `json:"date"`
	Reqs   int64  `json:"reqs"`
	Tokens int64  `json:"tokens"`
}

// DailyMetrics returns statistics grouped by day for the specified number of days
// The endpoint returns data for the last N days including today (N+1 days total)
func DailyMetrics(c *gin.Context) {
	days, err := strconv.Atoi(c.Param("days"))
	if err != nil {
		common.BadRequest(c, "Invalid days parameter")
		return
	}

	now := time.Now()
	year, month, day := now.Date()
	// Get data from N days ago to today (inclusive)
	startDate := time.Date(year, month, day, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -days)

	// Query to group by date
	type dailyResult struct {
		Date   string `gorm:"column:date"` // SQLite DATE() returns string in YYYY-MM-DD format
		Reqs   int64  `gorm:"column:reqs"`
		Tokens int64  `gorm:"column:tokens"`
	}

	var results []dailyResult
	err = models.DB.
		Model(&models.ChatLog{}).
		Select("DATE(created_at) as date, COUNT(*) as reqs, COALESCE(SUM(total_tokens), 0) as tokens").
		Where("created_at >= ?", startDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&results).Error

	if err != nil {
		common.InternalServerError(c, "Failed to query daily metrics: "+err.Error())
		return
	}

	// Convert to response format
	dailyMetrics := make([]DailyMetric, len(results))
	for i, result := range results {
		dailyMetrics[i] = DailyMetric{
			Date:   result.Date, // Already in YYYY-MM-DD format
			Reqs:   result.Reqs,
			Tokens: result.Tokens,
		}
	}

	common.Success(c, dailyMetrics)
}

type HourlyMetric struct {
	Hour   string `json:"hour"`
	Reqs   int64  `json:"reqs"`
	Tokens int64  `json:"tokens"`
}

// HourlyMetrics returns statistics grouped by hour for the specified number of hours
// The endpoint returns data for the last N hours including current hour
func HourlyMetrics(c *gin.Context) {
	hours, err := strconv.Atoi(c.Param("hours"))
	if err != nil {
		common.BadRequest(c, "Invalid hours parameter")
		return
	}

	now := time.Now()
	// Get data from N hours ago to now (inclusive)
	startTime := now.Add(-time.Duration(hours) * time.Hour)

	// Query to group by hour
	type hourlyResult struct {
		Hour   string `gorm:"column:hour"` // SQLite strftime returns string
		Reqs   int64  `gorm:"column:reqs"`
		Tokens int64  `gorm:"column:tokens"`
	}

	var results []hourlyResult
	err = models.DB.
		Model(&models.ChatLog{}).
		Select("strftime('%Y-%m-%d %H:00:00', created_at) as hour, COUNT(*) as reqs, COALESCE(SUM(total_tokens), 0) as tokens").
		Where("created_at >= ?", startTime).
		Group("strftime('%Y-%m-%d %H:00:00', created_at)").
		Order("hour ASC").
		Scan(&results).Error

	if err != nil {
		common.InternalServerError(c, "Failed to query hourly metrics: "+err.Error())
		return
	}

	// Convert to response format
	hourlyMetrics := make([]HourlyMetric, len(results))
	for i, result := range results {
		hourlyMetrics[i] = HourlyMetric{
			Hour:   result.Hour,
			Reqs:   result.Reqs,
			Tokens: result.Tokens,
		}
	}

	common.Success(c, hourlyMetrics)
}

type Count struct {
	Model string `json:"model"`
	Calls int64  `json:"calls"`
}

func Counts(c *gin.Context) {
	results := make([]Count, 0)
	if err := models.DB.
		Model(&models.ChatLog{}).
		Select("name as model, COUNT(*) as calls").
		Group("name").
		Order("calls DESC").
		Scan(&results).Error; err != nil {
		common.InternalServerError(c, err.Error())
		return
	}
	const topN = 5
	if len(results) > topN {
		var othersCalls int64
		for _, item := range results[topN:] {
			othersCalls += item.Calls
		}
		othersCount := Count{
			Model: "others",
			Calls: othersCalls,
		}
		results = append(results[:topN], othersCount)
	}

	common.Success(c, results)
}

type ProjectCount struct {
	Project string `json:"project"`
	Calls   int64  `json:"calls"`
}

func ProjectCounts(c *gin.Context) {
	type authKeyCount struct {
		AuthKeyID uint  `gorm:"column:auth_key_id"`
		Calls     int64 `gorm:"column:calls"`
	}

	rows := make([]authKeyCount, 0)
	if err := models.DB.
		Model(&models.ChatLog{}).
		Select("auth_key_id, COUNT(*) as calls").
		Group("auth_key_id").
		Order("calls DESC").
		Scan(&rows).Error; err != nil {
		common.InternalServerError(c, err.Error())
		return
	}

	ids := make([]uint, 0)
	for _, row := range rows {
		if row.AuthKeyID == 0 {
			continue
		}
		ids = append(ids, row.AuthKeyID)
	}

	keys := make([]models.AuthKey, 0)
	if len(ids) > 0 {
		if err := models.DB.
			Model(&models.AuthKey{}).
			Where("id IN ?", ids).
			Find(&keys).Error; err != nil {
			common.InternalServerError(c, err.Error())
			return
		}
	}

	keyMap := make(map[uint]string, len(keys))
	for _, key := range keys {
		keyMap[key.ID] = strings.TrimSpace(key.Name)
	}

	projectCalls := make(map[string]int64)
	for _, row := range rows {
		project := "-"
		if row.AuthKeyID == 0 {
			project = "admin"
		} else if name, ok := keyMap[row.AuthKeyID]; ok && name != "" {
			project = name
		}
		projectCalls[project] += row.Calls
	}

	results := make([]ProjectCount, 0, len(projectCalls))
	for project, calls := range projectCalls {
		results = append(results, ProjectCount{
			Project: project,
			Calls:   calls,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Calls > results[j].Calls })

	const topN = 5
	if len(results) > topN {
		var othersCalls int64
		for _, item := range results[topN:] {
			othersCalls += item.Calls
		}
		othersCount := ProjectCount{
			Project: "others",
			Calls:   othersCalls,
		}
		results = append(results[:topN], othersCount)
	}

	common.Success(c, results)
}
