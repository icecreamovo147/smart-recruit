// Command time-preflight performs the read-only UTC+8 migration gate.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/mysqltime"
)

type columnReport struct {
	Table    string `json:"table"`
	Column   string `json:"column"`
	DataType string `json:"data_type"`
}

type checkReport struct {
	Name        string `json:"name"`
	Count       int64  `json:"count"`
	Blocking    bool   `json:"blocking"`
	Description string `json:"description"`
}

type report struct {
	GeneratedAt     string         `json:"generated_at"`
	Database        string         `json:"database"`
	DateTimeColumns []columnReport `json:"datetime_columns"`
	Checks          []checkReport  `json:"checks"`
	Ready           bool           `json:"ready"`
}

type countCheck struct {
	name, description string
	query             string
	blocking          bool
}

func main() {
	businessclock.Configure()
	dsnFlag := flag.String("dsn", "", "MySQL DSN; defaults to MYSQL_DSN")
	timeoutFlag := flag.Duration("timeout", 30*time.Second, "preflight timeout")
	flag.Parse()

	dsn := strings.TrimSpace(*dsnFlag)
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	}
	normalized, err := mysqltime.NormalizeDSN(dsn)
	if err != nil {
		exit(err)
	}
	db, err := sql.Open("mysql", normalized)
	if err != nil {
		exit(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), *timeoutFlag)
	defer cancel()
	if err := mysqltime.ValidateSession(ctx, db); err != nil {
		exit(err)
	}

	result, err := inspect(ctx, db)
	if err != nil {
		exit(err)
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		exit(err)
	}
	fmt.Println(string(encoded))
	if !result.Ready {
		os.Exit(1)
	}
}

func inspect(ctx context.Context, db *sql.DB) (report, error) {
	var databaseName string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&databaseName); err != nil {
		return report{}, fmt.Errorf("resolve database: %w", err)
	}
	rows, err := db.QueryContext(ctx, `SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND DATA_TYPE IN ('datetime', 'timestamp')
		ORDER BY TABLE_NAME, ORDINAL_POSITION`)
	if err != nil {
		return report{}, fmt.Errorf("list time columns: %w", err)
	}
	defer rows.Close()
	result := report{GeneratedAt: businessclock.FormatRFC3339(businessclock.Now()), Database: databaseName, Ready: true}
	for rows.Next() {
		var column columnReport
		if err := rows.Scan(&column.Table, &column.Column, &column.DataType); err != nil {
			return report{}, err
		}
		result.DateTimeColumns = append(result.DateTimeColumns, column)
	}
	if err := rows.Err(); err != nil {
		return report{}, err
	}
	var auditTableExists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'utc8_time_conversion_audit'
	)`).Scan(&auditTableExists); err != nil {
		return report{}, fmt.Errorf("detect UTC+8 audit table: %w", err)
	}
	checks := []countCheck{
		{name: "convert_platform_seed_publications", description: "可证明由迁移发布的平台套餐时间", query: `SELECT COUNT(*) FROM platform_plan_versions WHERE effective_at IS NOT NULL AND published_by IS NULL AND status IN ('published','retired')`},
		{name: "convert_immediate_prices", description: "000081 可证明的即时价格发布时间", query: `SELECT COUNT(*) FROM billing_price_versions WHERE effective_at IS NOT NULL AND status IN ('published','retired') AND ABS(TIMESTAMPDIFF(SECOND, created_at, effective_at)) <= 5`},
		{name: "convert_immediate_rate_cards", description: "000081 可证明的即时费率卡发布时间", query: `SELECT COUNT(*) FROM ai_rate_cards WHERE effective_at IS NOT NULL AND status IN ('published','retired') AND ABS(TIMESTAMPDIFF(SECOND, created_at, effective_at)) <= 5`},
		{name: "ambiguous_platform_publications", description: "由运行时用户发布、无法仅凭数据证明旧时区语义", query: `SELECT COUNT(*) FROM platform_plan_versions WHERE effective_at IS NOT NULL AND published_by IS NOT NULL`, blocking: true},
		{name: "ambiguous_scheduled_prices", description: "非即时价格发布时间需要人工确认", query: `SELECT COUNT(*) FROM billing_price_versions WHERE effective_at IS NOT NULL AND status IN ('published','retired') AND ABS(TIMESTAMPDIFF(SECOND, created_at, effective_at)) > 5`, blocking: true},
		{name: "ambiguous_scheduled_rate_cards", description: "非即时费率卡发布时间需要人工确认", query: `SELECT COUNT(*) FROM ai_rate_cards WHERE effective_at IS NOT NULL AND status IN ('published','retired') AND ABS(TIMESTAMPDIFF(SECOND, created_at, effective_at)) > 5`, blocking: true},
		{name: "invalid_subscription_period", description: "订阅结束时间不晚于开始时间", query: `SELECT COUNT(*) FROM billing_subscriptions WHERE current_period_end <= current_period_start`, blocking: true},
		{name: "invalid_order_lifecycle", description: "订单支付/关闭/过期时间早于创建时间", query: `SELECT COUNT(*) FROM billing_orders WHERE expires_at < created_at OR (paid_at IS NOT NULL AND paid_at < created_at) OR (closed_at IS NOT NULL AND closed_at < created_at)`, blocking: true},
		{name: "future_open_order_clock", description: "未完成订单创建时间位于当前数据库时间五分钟之后", query: `SELECT COUNT(*) FROM billing_orders WHERE status IN ('pending','paying') AND created_at > DATE_ADD(NOW(3), INTERVAL 5 MINUTE)`, blocking: true},
		{name: "negative_payment_age", description: "待处理支付产生负年龄", query: `SELECT COUNT(*) FROM billing_payments WHERE status IN ('pending','closing','unknown') AND TIMESTAMPDIFF(SECOND, created_at, NOW(3)) < 0`, blocking: true},
	}
	if auditTableExists {
		checks[1].query = `SELECT COUNT(*) FROM utc8_time_conversion_audit WHERE batch_id = '000082_standardize_utc8_time_semantics' AND table_name = 'billing_price_versions' AND column_name = 'effective_at'`
		checks[2].query = `SELECT COUNT(*) FROM utc8_time_conversion_audit WHERE batch_id = '000082_standardize_utc8_time_semantics' AND table_name = 'ai_rate_cards' AND column_name = 'effective_at'`
		checks[4].query = `SELECT COUNT(*) FROM billing_price_versions source
			WHERE source.effective_at IS NOT NULL AND source.status IN ('published','retired')
			  AND ABS(TIMESTAMPDIFF(SECOND, source.created_at, source.effective_at)) > 5
			  AND NOT EXISTS (SELECT 1 FROM utc8_time_conversion_audit audit
				WHERE audit.batch_id = '000082_standardize_utc8_time_semantics'
				  AND audit.table_name = 'billing_price_versions' AND audit.row_pk = CAST(source.id AS CHAR)
				  AND audit.column_name = 'effective_at')`
		checks[5].query = `SELECT COUNT(*) FROM ai_rate_cards source
			WHERE source.effective_at IS NOT NULL AND source.status IN ('published','retired')
			  AND ABS(TIMESTAMPDIFF(SECOND, source.created_at, source.effective_at)) > 5
			  AND NOT EXISTS (SELECT 1 FROM utc8_time_conversion_audit audit
				WHERE audit.batch_id = '000082_standardize_utc8_time_semantics'
				  AND audit.table_name = 'ai_rate_cards' AND audit.row_pk = CAST(source.id AS CHAR)
				  AND audit.column_name = 'effective_at')`
	}
	for _, check := range checks {
		var count int64
		if err := db.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return report{}, fmt.Errorf("run %s: %w", check.name, err)
		}
		item := checkReport{Name: check.name, Count: count, Blocking: check.blocking && count > 0, Description: check.description}
		result.Checks = append(result.Checks, item)
		if item.Blocking {
			result.Ready = false
		}
	}
	return result, nil
}

func exit(err error) {
	fmt.Fprintf(os.Stderr, "UTC+8 time preflight failed: %v\n", err)
	os.Exit(2)
}
