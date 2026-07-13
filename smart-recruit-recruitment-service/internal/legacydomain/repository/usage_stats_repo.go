package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"smart-recruit-recruitment-service/internal/legacydomain/model"
)

// UsageStatsRepo handles aggregation queries on third_party_usage_logs.
type UsageStatsRepo struct {
	db *gorm.DB
}

func NewUsageStatsRepo(db *gorm.DB) *UsageStatsRepo {
	return &UsageStatsRepo{db: db}
}

// UsageStatsRow holds aggregated usage statistics for a dimension value.
type UsageStatsRow struct {
	Name          string
	TotalTokens   int64
	CallCount     int64
	AvgCostMs     float64
	EstimatedCost float64
}

// GetStatsByModel aggregates usage by model name.
func (r *UsageStatsRepo) GetStatsByModel(ctx context.Context, startTime, endTime time.Time) ([]UsageStatsRow, error) {
	var rows []UsageStatsRow
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Select(`COALESCE(model, 'unknown') AS name,
			COALESCE(SUM(estimated_tokens), 0) AS total_tokens,
			COUNT(*) AS call_count,
			COALESCE(AVG(cost_ms), 0) AS avg_cost_ms,
			COALESCE(SUM(estimated_tokens) * 0.002 / 1000, 0) AS estimated_cost`).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("model").
		Order("total_tokens DESC").
		Find(&rows).Error
	return rows, err
}

// GetStatsByUser aggregates usage by user_id.
func (r *UsageStatsRepo) GetStatsByUser(ctx context.Context, startTime, endTime time.Time) ([]UsageStatsRow, error) {
	var rows []UsageStatsRow
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Select(`CAST(user_id AS CHAR) AS name,
			COALESCE(SUM(estimated_tokens), 0) AS total_tokens,
			COUNT(*) AS call_count,
			COALESCE(AVG(cost_ms), 0) AS avg_cost_ms,
			COALESCE(SUM(estimated_tokens) * 0.002 / 1000, 0) AS estimated_cost`).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("user_id").
		Order("total_tokens DESC").
		Limit(10).
		Find(&rows).Error
	return rows, err
}

// GetStatsBySession aggregates usage by request_id (session).
func (r *UsageStatsRepo) GetStatsBySession(ctx context.Context, startTime, endTime time.Time) ([]UsageStatsRow, error) {
	var rows []UsageStatsRow
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Select(`COALESCE(request_id, 'unknown') AS name,
			COALESCE(SUM(estimated_tokens), 0) AS total_tokens,
			COUNT(*) AS call_count,
			COALESCE(AVG(cost_ms), 0) AS avg_cost_ms,
			COALESCE(SUM(estimated_tokens) * 0.002 / 1000, 0) AS estimated_cost`).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Where("request_id != ''").
		Group("request_id").
		Order("total_tokens DESC").
		Limit(10).
		Find(&rows).Error
	return rows, err
}

// UsageTrendRow holds aggregated usage data for a time bucket.
type UsageTrendRow struct {
	Date          string
	TotalTokens   int64
	CallCount     int64
	AvgCostMs     float64
	EstimatedCost float64
}

// GetTrend returns daily/weekly/monthly aggregated usage trend.
func (r *UsageStatsRepo) GetTrend(ctx context.Context, startTime, endTime time.Time, granularity string) ([]UsageTrendRow, error) {
	var dateExpr string
	switch granularity {
	case "week":
		dateExpr = "DATE_FORMAT(created_at, '%x-W%v')"
	case "month":
		dateExpr = "DATE_FORMAT(created_at, '%Y-%m')"
	default: // day
		dateExpr = "DATE(created_at)"
	}

	selectExpr := fmt.Sprintf(`%s AS date,
		COALESCE(SUM(estimated_tokens), 0) AS total_tokens,
		COUNT(*) AS call_count,
		COALESCE(AVG(cost_ms), 0) AS avg_cost_ms,
		COALESCE(SUM(estimated_tokens) * 0.002 / 1000, 0) AS estimated_cost`, dateExpr)

	var rows []UsageTrendRow
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Select(selectExpr).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("date").
		Order("date ASC").
		Find(&rows).Error
	return rows, err
}

// GetSummary returns overall aggregated statistics.
func (r *UsageStatsRepo) GetSummary(ctx context.Context, startTime, endTime time.Time) (*UsageStatsRow, error) {
	var row UsageStatsRow
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Select(`'total' AS name,
			COALESCE(SUM(estimated_tokens), 0) AS total_tokens,
			COUNT(*) AS call_count,
			COALESCE(AVG(cost_ms), 0) AS avg_cost_ms,
			COALESCE(SUM(estimated_tokens) * 0.002 / 1000, 0) AS estimated_cost`).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Find(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
